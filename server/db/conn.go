package db

import (
	"context"
	"log"
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

func Connect() {
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost:27017"))
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
	MongoClient = client
	DB = client.Database("session-cookie-go")

	UserCollection = DB.Collection("users")
	PfpCollection = DB.Collection("pfps")
	SessionCollection = DB.Collection("sessions")
	RoomCollection = DB.Collection("rooms")
}
