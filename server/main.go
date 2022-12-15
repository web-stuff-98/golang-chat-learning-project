package main

import (
	"context"
	"cookie-session/api/controllers"
	"cookie-session/api/routes"
	"cookie-session/db"
	"fmt"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/websocket/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type InboundMessage struct {
	Content   string          `json:"msg"`
	SenderUid string          `json:"uid"`
	WsConn    *websocket.Conn `json:"-"`
}

type ChatServer struct {
	connections      map[*websocket.Conn]bool
	connectionsByUid map[string]*websocket.Conn

	inbound chan InboundMessage

	registerConn   chan *websocket.Conn
	unregisterConn chan *websocket.Conn

	//Rooms currently not set up
	roomConnections      map[*websocket.Conn]bool
	roomConnectionsByUid map[string]*websocket.Conn

	registerRoomConn   chan *websocket.Conn
	unregisterRoomConn chan *websocket.Conn
}

func NewChatServer() *ChatServer {
	return &ChatServer{
		connections:      make(map[*websocket.Conn]bool),
		connectionsByUid: make(map[string]*websocket.Conn),

		inbound: make(chan InboundMessage),

		registerConn:   make(chan *websocket.Conn),
		unregisterConn: make(chan *websocket.Conn),

		//Rooms currently not set up
		roomConnections:      make(map[*websocket.Conn]bool),
		roomConnectionsByUid: make(map[string]*websocket.Conn),

		registerRoomConn:   make(chan *websocket.Conn),
		unregisterRoomConn: make(chan *websocket.Conn),
	}
}

func main() {
	app := fiber.New()

	db.Connect()

	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
	}))

	chatServer, closeWsChan, err := controllers.SetupChatServer()
	if err != nil {
		log.Fatal(fmt.Printf("Failed to setup chat server : %d", err))
	}

	/* Cleanup interval to delete expired sessions still in the database */
	cleanupTicker := time.NewTicker(240 * time.Second)
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
