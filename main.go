package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/web-stuff-98/golang-chat-learning-project/api/controllers"
	"github.com/web-stuff-98/golang-chat-learning-project/api/routes"
	"github.com/web-stuff-98/golang-chat-learning-project/api/seed"
	"github.com/web-stuff-98/golang-chat-learning-project/db"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {
	dotEnvErr := godotenv.Load()

	app := fiber.New()

	db.Connect()

	var uids, rids map[primitive.ObjectID]struct{}
	var seedErr error
	if dotEnvErr != nil {
		log.Println("No .env file detected. Continuing as in production mode...")
		uids, rids, seedErr = seed.GenerateSeed(20, 100)
	} else {
		log.Println("Loaded .env file. Continuing as in development mode...")
		uids, rids, seedErr = seed.GenerateSeed(5, 10)
	}
	if seedErr != nil {
		log.Fatal("Seed error : ", seedErr)
	}

	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
	}))

	chatServer, closeWsChan, deleteUserChan, err := controllers.NewServer()
	if err != nil {
		log.Fatal(fmt.Printf("Failed to setup chat server : %d", err))
	}

	/* -------- Clean up expired sessions still in the database every 960 seconds -------- */
	cleanupTicker := time.NewTicker(960 * time.Second)
	quitCleanup := make(chan struct{})
	go func() {
		for {
			select {
			case <-cleanupTicker.C:
				db.SessionCollection.DeleteMany(context.TODO(), bson.M{"exp": bson.M{"$lt": primitive.NewDateTimeFromTime(time.Now())}})
			case <-quitCleanup:
				cleanupTicker.Stop()
				return
			}
		}
	}()
	defer func() {
		close(quitCleanup)
	}()

	go watchForDeletesInUserCollection(db.UserCollection, deleteUserChan)

	routes.Setup(app, chatServer, closeWsChan, uids, rids)
	log.Fatal(app.Listen(":8080"))
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
