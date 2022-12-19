package controllers

import (
	"context"
	"cookie-session/api/helpers"
	"cookie-session/api/validator"
	"cookie-session/db"
	"cookie-session/db/models"
	"fmt"
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

	chatRooms []*ChatRoom

	registerRoomConn   chan ChatRoomConnectionRegistration
	unregisterRoomConn chan ChatRoomConnectionRegistration
}

type ChatRoom struct {
	connections      map[*websocket.Conn]bool
	connectionsByUid map[string]*websocket.Conn
	roomId           string
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

type ChatRoomConnectionRegistration struct {
	id  string
	uid string
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

		registerRoomConn:   make(chan ChatRoomConnectionRegistration), //register connection by uid
		unregisterRoomConn: make(chan ChatRoomConnectionRegistration), //unregister connection by uid

		chatRooms: make([]*ChatRoom, 0),
	}

	go func() {
		for {
			msg := <-chatServer.inbound
			for i := range chatServer.chatRooms {
				conn := chatServer.chatRooms[i].connectionsByUid[msg.SenderUid]
				if conn != nil {
					for connI := range chatServer.chatRooms[i].connections {
						if conn != connI && connI != nil {
							println("Send msg : ", msg.Content)
							connI.WriteJSON(msg.Content)
						}
					}
				}
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
		go func() {
			c := <-chatServer.registerRoomConn
			conn := chatServer.connectionsByUid[c.uid]
			foundRoom := false
			for i := range chatServer.chatRooms {
				if chatServer.chatRooms[i].roomId == c.id {
					chatServer.chatRooms[i].connections[conn] = true
					chatServer.chatRooms[i].connectionsByUid[c.uid] = conn
					foundRoom = true
				}
			}
			if !foundRoom {
				//if the room is not found, that means there are no active connections,
				//but the room is there in the database. So go look for it and create the chatServer application data
				connections := make(map[*websocket.Conn]bool)
				connectionsByUid := make(map[string]*websocket.Conn)
				connections[conn] = true
				connectionsByUid[c.uid] = conn
				roomId := c.id
				chatServer.chatRooms = append(chatServer.chatRooms, &ChatRoom{
					connections,
					connectionsByUid,
					roomId,
				})
			}
		}()
		go func() {
			c := <-chatServer.unregisterRoomConn
			conn := chatServer.connectionsByUid[c.uid]
			for i := range chatServer.chatRooms {
				if chatServer.chatRooms[i].roomId == c.id {
					delete(chatServer.chatRooms[i].connections, conn)
					delete(chatServer.chatRooms[i].connectionsByUid, c.uid)
				}
			}
		}()

		chatServer.registerConn <- c
		for {
			var (
				_   int //unused "mt / message type" parameter... dont get the point of this. doesn't seem to be any way to change it
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
			// Find room and write message to db
			for i := range chatServer.chatRooms {
				for connI := range chatServer.chatRooms[i].connections {
					if connI == c {
						oid, err := primitive.ObjectIDFromHex(chatServer.chatRooms[i].roomId)
						if err != nil {
							break
						}
						var room models.Room
						found := db.RoomCollection.FindOne(context.TODO(), bson.M{"_id": oid})
						if found == nil {
							break
						} else {
							found.Decode(&room)
						}
						msg := models.Message{
							Content:   string(msg),
							Uid:       c.Locals("uid").(primitive.ObjectID).Hex(),
							Timestamp: primitive.NewDateTimeFromTime(time.Now()),
						}
						println("Push array")
						db.RoomCollection.UpdateOne(context.TODO(), bson.M{"_id": oid}, bson.M{"$push": bson.M{"messages": msg}})
					}
				}
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
	if c.Query("own") == "true" {
		uid, err := helpers.DecodeTokenAndGetUID(c)
		if err != nil {
			c.Status(fiber.StatusUnauthorized)
			return c.JSON(fiber.Map{
				"message": "Unauthorized",
			})
		}
		findFilter = bson.M{"author_id": uid}
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

	fmt.Println("Room ID : ", room.ID)

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
		Messages:  []models.Message{},
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

func UpdateRoom(c *fiber.Ctx) error {
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

func JoinRoom(chatServer *ChatServer) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		uid, err := helpers.DecodeTokenAndGetUID(c)
		if err != nil {
			c.Status(fiber.StatusUnauthorized)
			return c.JSON(fiber.Map{
				"message": "Unauthorized",
			})
		}

		if c.Params("id") == "" {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "Bad request",
			})
		}
		id, err := primitive.ObjectIDFromHex(c.Params("id"))
		if err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "Bad request",
			})
		}

		var room models.Room
		found := db.RoomCollection.FindOne(c.Context(), bson.M{"_id": id})
		if found == nil {
			c.Status(fiber.StatusNotFound)
			return c.JSON(fiber.Map{
				"message": "Room not found",
			})
		} else {
			found.Decode(&room)
		}

		chatServer.registerRoomConn <- ChatRoomConnectionRegistration{id: c.Params("id"), uid: uid.Hex()}

		fmt.Println("Room ID : ", room.ID)

		c.Status(fiber.StatusOK)
		return c.JSON(room)
	}
}

func LeaveRoom(chatServer *ChatServer) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		uid, err := helpers.DecodeTokenAndGetUID(c)
		if err != nil {
			c.Status(fiber.StatusUnauthorized)
			return c.JSON(fiber.Map{
				"message": "Unauthorized",
			})
		}

		if c.Params("id") == "" {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "Bad request",
			})
		}

		id, err := primitive.ObjectIDFromHex(c.Params("id"))
		if err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "Bad request",
			})
		}

		var room bson.M
		db.RoomCollection.FindOne(c.Context(), bson.M{"_id": id}).Decode(&room)
		if len(room) == 0 {
			c.Status(fiber.StatusNotFound)
			return c.JSON(fiber.Map{
				"message": "Room not found",
			})
		}

		chatServer.unregisterRoomConn <- ChatRoomConnectionRegistration{id: c.Params("id"), uid: uid.Hex()}

		println("Left room")

		c.Status(fiber.StatusOK)
		return c.JSON(fiber.Map{
			"message": "Left room",
		})
	}
}
