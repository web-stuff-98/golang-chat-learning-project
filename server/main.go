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
	"github.com/web-stuff-98/golang-chat-learning-project/db/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	dotEnvErr := godotenv.Load()

	app := fiber.New()

	app.Static("/", "./build")

	db.Connect()

	/* -------- Create map to store client IP addresses and associated data used by rate limiter -------- */
	ipBlockInfoMap := make(map[string]map[string]mylimiter.BlockInfo)

	uids := make(map[primitive.ObjectID]struct{})
	rids := make(map[primitive.ObjectID]struct{})

	var production bool = false
	if dotEnvErr != nil {
		log.Fatal("DOTENV ERROR : ", dotEnvErr)
	}
	log.Println("Loaded environment variables...")
	if os.Getenv("PRODUCTION") == "true" {
		production = true
	}
	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
	}))

	chatServer, closeWsChan, deleteUserChan, err := controllers.NewServer()
	if err != nil {
		log.Fatal(fmt.Printf("Failed to setup chat server : %d", err))
	}

	/* -------- Generate seed and store ids in memory -------- */
	var seedErr error
	go func() {
		if !production {
			uids, rids, seedErr = seed.GenerateSeed(5, 20)
		} else {
			// need to change this back to 50 accs and 255 rooms when everything is working as expected
			uids, rids, seedErr = seed.GenerateSeed(1, 5)
		}
		if seedErr != nil {
			log.Fatal("Seed error : ", seedErr)
		}

		log.Println(len(uids))
		log.Println(len(rids))
	}()

	/* -------- Set up routes and send the complete maps of protected ids -------- */
	routes.Setup(app, chatServer, closeWsChan, &uids, &rids, ipBlockInfoMap, production)

	/* -------- Clean up expired sessions still in the database every 960 seconds -------- */
	sessionCleanupTicker := time.NewTicker(960 * time.Second)
	quitSessionCleanup := make(chan struct{})
	go func() {
		for {
			select {
			case <-sessionCleanupTicker.C:
				db.SessionCollection.DeleteMany(context.TODO(), bson.M{"exp": bson.M{"$lt": primitive.NewDateTimeFromTime(time.Now())}})
			case <-quitSessionCleanup:
				sessionCleanupTicker.Stop()
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
		close(quitSessionCleanup)
		close(quitOldAccountCleanup)
	}()

	go watchForDeletesInUserCollection(db.UserCollection, deleteUserChan)

	log.Fatal(app.Listen(fmt.Sprint(":", os.Getenv("PORT"))))
}

// Watch for deletions in users collection... need to delete their messages and rooms and send the delete ws event to other users
func watchForDeletesInUserCollection(collection *mongo.Collection, deleteUserChan chan string) {
	userDeletePipeline := bson.D{
		{
			"$match", bson.D{
				{"operationType", "delete"},
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
