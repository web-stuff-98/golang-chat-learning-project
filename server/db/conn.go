package db

import (
	"context"
	"log"
	"os"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

var MongoClient *mongo.Client
var DB *mongo.Database

var UserCollection *mongo.Collection
var PfpCollection *mongo.Collection
var SessionCollection *mongo.Collection
var RoomCollection *mongo.Collection
var RoomImageCollection *mongo.Collection
var AttachmentCollection *mongo.Collection

func Connect() {
	log.Println("Connecting to MongoDB...")
	client, err := mongo.NewClient(options.Client().ApplyURI(os.Getenv("MONGODB_URI")))
	if err != nil {
		log.Fatal(err)
	}
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		log.Fatal(err)
	}
	log.Println("MongoDB connected")
	MongoClient = client
	DB = client.Database(os.Getenv("MONGODB_DB"))

	UserCollection = DB.Collection("users")
	PfpCollection = DB.Collection("pfps")
	SessionCollection = DB.Collection("sessions")
	RoomCollection = DB.Collection("rooms")
	RoomImageCollection = DB.Collection("roompics")
	AttachmentCollection = DB.Collection("attachments")
}
