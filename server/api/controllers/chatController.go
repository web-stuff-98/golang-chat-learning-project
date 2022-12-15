package controllers

import (
	"context"
	"cookie-session/api/helpers"
	"cookie-session/api/validator"
	"cookie-session/db"
	"cookie-session/db/models"
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
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

func HandleWsUpgrade(c *fiber.Ctx) error {
	if websocket.IsWebSocketUpgrade(c) {
		sessionId, err := helpers.DecodeTokenIssuer(c)
		if err == nil {
			user, err := helpers.GetUserFromSID(c, sessionId)
			if err != nil {
				c.Status(fiber.StatusUnauthorized)
				return c.JSON(fiber.Map{
					"message": "Unauthorized",
				})
			}
			c.Locals("uid", user["_id"])
		} else {
			c.Status(fiber.StatusUnauthorized)
			return c.JSON(fiber.Map{
				"message": "Unauthorized",
			})
		}
		socketId := uuid.New().String()
		c.Locals("socketId", socketId)
		helpers.AddSocketIdToSession(c, socketId)
		return c.Next()
	}
	return fiber.ErrUpgradeRequired
}

func SetupChatServer() (*ChatServer, chan string, error) {
	//closeWsChan can be used to close websockets using the users id
	closeWsChan := make(chan string)

	chatServer := &ChatServer{
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

	go func() {
		for {
			msg := <-chatServer.inbound
			for wsConn := range chatServer.connections {
				wsConn.WriteJSON(msg)
			}
		}
	}()

	go func() {
		uid := <-closeWsChan
		conn := chatServer.connectionsByUid[uid]
		delete(chatServer.connectionsByUid, uid)
		delete(chatServer.connections, conn)
	}()

	return chatServer, closeWsChan, nil
}

/* ------------------ WS HTTP API ROUTES ------------------ */

func HandleWsConn(chatServer *ChatServer, closeWsChan chan string) func(*fiber.Ctx) error {
	return websocket.New(func(c *websocket.Conn) {
		go func() {
			c := <-chatServer.registerConn
			chatServer.connections[c] = true
			chatServer.connectionsByUid[c.Locals("uid").(primitive.ObjectID).Hex()] = c
		}()
		go func() {
			c := <-chatServer.unregisterConn
			delete(chatServer.connections, c)
			delete(chatServer.connectionsByUid, c.Locals("uid").(primitive.ObjectID).Hex())
		}()

		chatServer.registerConn <- c

		for {
			var (
				_   int //unused "mt / message type" parameter... dont get the point of this.
				msg []byte
				err error
			)
			if _, msg, err = c.ReadMessage(); err != nil {
				if !websocket.IsCloseError(err) {
					log.Println("ReadErr: ", err)
				} else {
					log.Println("Websocket connection closed")
				}
				break
			}
			chatServer.inbound <- InboundMessage{
				WsConn:    c,
				Content:   string(msg),
				SenderUid: c.Locals("uid").(primitive.ObjectID).Hex(),
			}
		}
		defer func() {
			if c.Locals("uid") != "" {
				closeWsChan <- c.Locals("uid").(primitive.ObjectID).Hex()
			}
		}()
	})
}

/* ------------------ HTTP API ROUTES ------------------ */

func GetRooms(c *fiber.Ctx) error {
	var rooms []models.Room
	var findFilter bson.M = bson.M{}
	println("QUERY OWN : ", c.Query("own"))
	if c.Query("own") == "true" {
		uid, err := helpers.DecodeTokenAndGetUID(c)
		if err != nil {
			c.Status(fiber.StatusUnauthorized)
			return c.JSON(fiber.Map{
				"message": "Unauthorized",
			})
		}
		findFilter = bson.M{
			"author_id": uid.Hex(),
		}
	}
	cur, err := db.RoomCollection.Find(c.Context(), findFilter)
	if err != nil {
		log.Println("Error finding in rooms collection : ", err)
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Internal error",
		})
	}
	for cur.Next(context.TODO()) {
		var elem models.Room
		err := cur.Decode(&elem)
		if err != nil {
			log.Fatal(err)
		}
		rooms = append(rooms, elem)
	}
	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	cur.Close(context.TODO())

	c.Status(fiber.StatusOK)

	return c.JSON(rooms)
}

func GetRoom(c *fiber.Ctx) error {
	if c.Params("id") == "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Bad request",
		})
	}

	var room models.Room
	err := db.RoomCollection.FindOne(c.Context(), bson.M{"_id": c.Params("id")}).Decode(&room)

	if err != nil {
		if err == mongo.ErrNilDocument || err == mongo.ErrNoDocuments {
			c.Status(fiber.StatusNotFound)
			return c.JSON(fiber.Map{
				"message": "Room not found",
			})
		} else {
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(fiber.Map{
				"message": "Internal error",
			})
		}
	}

	c.Status(fiber.StatusOK)
	return c.JSON(room)
}

func CreateRoom(c *fiber.Ctx) error {
	var body validator.Room
	if err := c.BodyParser(&body); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invalid request",
		})
	}

	uid, err := helpers.DecodeTokenAndGetUID(c)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}

	res, err := db.RoomCollection.InsertOne(c.Context(), models.Room{
		Name:      body.Name,
		CreatedAt: primitive.NewDateTimeFromTime(time.Now()),
		UpdatedAt: primitive.NewDateTimeFromTime(time.Now()),
		Author:    uid,
	})

	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Internal error",
		})
	}

	c.Status(fiber.StatusCreated)
	return c.JSON(fiber.Map{
		"_id":        res.InsertedID.(primitive.ObjectID).Hex(),
		"name":       body.Name,
		"created_at": primitive.NewDateTimeFromTime(time.Now()),
		"updated_at": primitive.NewDateTimeFromTime(time.Now()),
		"author_id":  uid.Hex(),
	})
}

func UpdateRoomName(c *fiber.Ctx) error {
	if c.Params("id") == "" {
		c.Status(fiber.StatusBadRequest)
		c.JSON(fiber.Map{
			"message": "Bad request",
		})
	}

	var body struct {
		Name string `json:"name"`
	}

	if err := c.BodyParser(&body); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Bad request",
		})
	}

	res, err := db.RoomCollection.UpdateOne(c.Context(), bson.M{"_id": c.Params("id"), "author_id": c.Locals("uid").(primitive.ObjectID).Hex()}, bson.D{{"$set", bson.D{{"name", body.Name}}}})

	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Internal error",
		})
	}

	if res.ModifiedCount == 0 {
		//Room could not be found, or the user trying to modify the room is not the owner
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Bad request",
		})
	}

	c.Status(fiber.StatusOK)
	return c.JSON(fiber.Map{
		"message": "Room name updated",
	})
}

func DeleteRoom(c *fiber.Ctx) error {
	if c.Params("id") == "" {
		c.Status(fiber.StatusBadRequest)
		c.JSON(fiber.Map{
			"message": "Bad request",
		})
	}

	res, err := db.RoomCollection.DeleteOne(c.Context(), bson.M{"_id": c.Params("id"), "author_id": c.Locals("uid").(primitive.ObjectID).Hex()})

	if res.DeletedCount == 0 {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Bad request",
		})
	}

	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Internal error",
		})
	}

	c.Status(fiber.StatusOK)
	return c.JSON(fiber.Map{
		"message": "Room deleted",
	})
}
