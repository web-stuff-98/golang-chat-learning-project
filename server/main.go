package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/web-stuff-98/golang-chat-learning-project/api/controllers"
	"github.com/web-stuff-98/golang-chat-learning-project/api/mylimiter"
	"github.com/web-stuff-98/golang-chat-learning-project/api/routes"
	"github.com/web-stuff-98/golang-chat-learning-project/api/seed"
	"github.com/web-stuff-98/golang-chat-learning-project/db"
	"github.com/web-stuff-98/golang-chat-learning-project/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	dotEnvErr := godotenv.Load()
	if dotEnvErr != nil {
		log.Fatal("DOTENV ERROR : ", dotEnvErr)
	}

	app := fiber.New(fiber.Config{
		BodyLimit: 20 * 1024 * 1024, //largest file allowed to upload is 20mb
	})

	app.Static("/", "./build")

	db.Connect()

	/* -------- Create map to store client IP addresses and associated data used by rate limiter -------- */
	ipBlockInfoMap := make(map[string]map[string]mylimiter.BlockInfo)

	/* -------- Create maps to store IDs of example rooms and users so they cant be modified -------- */
	uids := make(map[primitive.ObjectID]struct{})
	rids := make(map[primitive.ObjectID]struct{})

	var production bool = false
	production = os.Getenv("PRODUCTION") == "true"
	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
	}))

	chatServer, removeChatServerConnByUID, removeChatServerConn, deleteUserChan, deleteMsgChan, err := controllers.NewServer()
	if err != nil {
		log.Fatal(fmt.Printf("Failed to setup chat server : %d", err))
	}

	/* -------- Generate seed and store ids in memory -------- */
	var seedErr error
	go func() {
		if !production {
			uids, rids, seedErr = seed.GenerateSeed(5, 3)
		} else {
			uids, rids, seedErr = seed.GenerateSeed(50, 255)
		}
		if seedErr != nil {
			log.Fatal("Seed error : ", seedErr)
		}
	}()

	/* -------- Set up routes with all the data needed sent down -------- */
	routes.Setup(app, chatServer, removeChatServerConnByUID, removeChatServerConn, &uids, &rids, ipBlockInfoMap, production)

	/* -------- Every 2 minutes clean up sessions, ipBlockInfo, and delete old messages -------- */
	cleanupTicker := time.NewTicker(2 * time.Minute)
	quitCleanup := make(chan struct{})
	go func() {
		for {
			select {
			case <-cleanupTicker.C:
				db.SessionCollection.DeleteMany(context.TODO(), bson.M{"exp": bson.M{"$lt": primitive.NewDateTimeFromTime(time.Now())}})
				for ip, routeBlockInfoMap := range ipBlockInfoMap {
					for routeName, blockInfo := range routeBlockInfoMap {
						if blockInfo.RequestsInWindow >= blockInfo.OptsUsed.MaxReqs && time.Now().After(blockInfo.LastRequest.Add(blockInfo.OptsUsed.BlockDuration)) {
							delete(routeBlockInfoMap, routeName)
						}
					}
					if len(routeBlockInfoMap) == 0 {
						delete(ipBlockInfoMap, ip)
					}
				}
				findOpts := options.Find().SetBatchSize(10)
				cursor, err := db.RoomCollection.Find(context.TODO(), bson.D{}, findOpts)
				if err != nil {
					log.Fatal("CURSOR ERR : ", err)
				}
				for cursor.Next(context.TODO()) {
					var room models.Room
					err := cursor.Decode(&room)
					if err != nil {
						log.Fatal("ERROR DECODING : ", err)
					}
					for _, m := range room.Messages {
						if m.Timestamp.Time().Before(time.Now().Add(-time.Minute * 20)) {
							deleteMsgChan <- controllers.RoomIdMessageId{
								RoomId:    room.ID,
								MessageId: m.ID,
							}
						}
					}
				}
				defer cursor.Close(context.TODO())
			case <-quitCleanup:
				cleanupTicker.Stop()
				return
			}
		}
	}()

	/* -------- Delete accounts older than 20 minutes (changestream delete event will trigger deleting the users rooms and messages also) -------- */
	oldAccountCleanupTicker := time.NewTicker(120 * time.Second)
	quitOldAccountCleanup := make(chan struct{})
	go func() {
		for {
			select {
			case <-oldAccountCleanupTicker.C:
				findOpts := options.Find().SetBatchSize(10)
				cursor, err := db.UserCollection.Find(context.TODO(), bson.D{}, findOpts)
				if err != nil {
					log.Fatal("CURSOR ERR : ", err)
				}
				for cursor.Next(context.TODO()) {
					var user models.User
					err := cursor.Decode(&user)
					if err != nil {
						log.Fatal("ERROR DECODING : ", err)
					}
					_, ok := uids[user.ID]
					if !ok {
						if user.ID.Timestamp().Add(time.Minute * 20).After(time.Now()) {
							deleteUserChan <- user.ID.Hex()
						}
					}
				}
				defer cursor.Close(context.TODO())
			case <-quitOldAccountCleanup:
				oldAccountCleanupTicker.Stop()
				return
			}
		}
	}()
	defer func() {
		close(quitCleanup)
		close(quitOldAccountCleanup)
	}()

	go watchForDeletesInUserCollection(db.UserCollection, deleteUserChan)

	log.Fatal(app.Listen(fmt.Sprint(":", os.Getenv("PORT"))))
}

// Watch for deletions in users collection... need to delete their messages and rooms and send the delete ws event to other users
func watchForDeletesInUserCollection(collection *mongo.Collection, deleteUserChan chan string) {
	userDeletePipeline := bson.D{
		{
			Key: "$match", Value: bson.D{
				{Key: "operationType", Value: "delete"},
			},
		},
	}
	cs, err := collection.Watch(context.TODO(), mongo.Pipeline{userDeletePipeline})
	if err != nil {
		log.Fatal("CS ERR : ", err.Error())
	}
	for cs.Next(context.TODO()) {
		var changeEv bson.M
		err := cs.Decode(&changeEv)
		if err != nil {
			log.Fatal(err)
		}
		uid := changeEv["documentKey"].(bson.M)["_id"].(primitive.ObjectID)
		deleteUserChan <- uid.Hex()
	}
}
