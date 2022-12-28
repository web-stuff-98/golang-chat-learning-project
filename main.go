package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/joho/godotenv"
	"github.com/web-stuff-98/golang-chat-learning-project/api/controllers"
	"github.com/web-stuff-98/golang-chat-learning-project/api/routes"
	"github.com/web-stuff-98/golang-chat-learning-project/db"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		println("No .env file detected. env vars should be set in docker container for production. Continuing...")
	}

	app := fiber.New()

	db.Connect()

	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
	}))

	chatServer, closeWsChan, err := controllers.NewServer()
	if err != nil {
		log.Fatal(fmt.Printf("Failed to setup chat server : %d", err))
	}

	/* Cleanup interval to delete expired sessions still in the db */
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

	routes.Setup(app, chatServer, closeWsChan)

	log.Fatal(app.Listen(":8080"))
}
