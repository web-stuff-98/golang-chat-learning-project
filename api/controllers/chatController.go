package controllers

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"strings"
	"time"

	"github.com/web-stuff-98/golang-chat-learning-project/api/helpers"
	"github.com/web-stuff-98/golang-chat-learning-project/api/validator"
	"github.com/web-stuff-98/golang-chat-learning-project/db"
	"github.com/web-stuff-98/golang-chat-learning-project/db/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
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

func NewServer() (*ChatServer, chan string, error) {
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
							connI.WriteJSON(msg)
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
				//but the room is there in the database. So go look for it and add it to memory
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
		//find the room image if it has one, and add the base64 to the data
		found := db.RoomImageCollection.FindOne(c.Context(), bson.M{"_id": elem.ID})
		if found.Err() != nil {
			if found.Err() != mongo.ErrNoDocuments {
				cur.Close(context.TODO())
				c.Status(fiber.StatusInternalServerError)
				return c.JSON(fiber.Map{
					"message": "Internal error",
				})
			}
		} else {
			var imageDoc models.RoomImage
			found.Decode(&imageDoc)
			elem.Base64image = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(imageDoc.Binary.Data)
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

	var roomImage models.RoomImage
	found := db.RoomImageCollection.FindOne(c.Context(), bson.M{"_id": c.Params("id")})
	if found.Err() != nil {
		if found.Err() != mongo.ErrNoDocuments {
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(fiber.Map{
				"message": "Internal error",
			})
		}
	} else {
		found.Decode(&roomImage)
		room.Base64image = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(roomImage.Binary.Data)
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
		"ID":         res.InsertedID.(primitive.ObjectID).Hex(),
		"name":       body.Name,
		"created_at": primitive.NewDateTimeFromTime(time.Now()),
		"updated_at": primitive.NewDateTimeFromTime(time.Now()),
		"author_id":  uid.Hex(),
	})
}

// Updates the room name only
func UpdateRoom(c *fiber.Ctx) error {
	if c.Params("id") == "" {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Bad request",
		})
	}

	var body validator.Room

	if err := c.BodyParser(&body); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Bad request",
		})
	}

	uid, err := helpers.DecodeTokenAndGetUID(c)
	if err != nil {
		c.Status(fiber.StatusUnauthorized)
		println("UID err")
		return c.JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}

	oid, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invalid ID",
		})
	}

	found := db.RoomCollection.FindOne(c.Context(), bson.M{"_id": oid})
	if found.Err() != nil {
		if found.Err() == mongo.ErrNoDocuments {
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
	} else {
		var room models.Room
		found.Decode(&room)
		if room.Author != uid {
			c.Status(fiber.StatusUnauthorized)
			return c.JSON(fiber.Map{
				"message": "Unauthorized",
			})
		}
	}

	db.RoomCollection.UpdateByID(c.Context(), oid, bson.D{{"$set", bson.D{{"name", body.Name}}}})

	c.Status(fiber.StatusOK)
	return c.JSON(fiber.Map{
		"message": "Room name updated",
	})
}

const maxRoomImageSize = 20 * 1024 * 1024 //20mb

func UploadRoomImage(chatServer *ChatServer) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		file, err := c.FormFile("file")
		if err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": err,
			})
		}

		if file.Size > maxRoomImageSize {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "File too large. Max 20mb.",
			})
		}

		if !strings.HasPrefix(file.Header.Get("Content-Type"), "image/") {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "File is not an image",
			})
		}

		uid, err := helpers.DecodeTokenAndGetUID(c)
		if err != nil {
			c.Status(fiber.StatusNotFound)
			return c.JSON(fiber.Map{
				"message": "Your session could not be found",
			})
		}

		if c.Params("id") == "" {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "Bad request",
			})
		}
		roomId, err := primitive.ObjectIDFromHex(c.Params("id"))
		if err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "Invalid ID",
			})
		}
		var room models.Room
		found := db.RoomCollection.FindOne(c.Context(), bson.M{"_id": roomId})
		if found.Err() == mongo.ErrNoDocuments {
			c.Status(fiber.StatusNotFound)
			return c.JSON(fiber.Map{
				"message": "Room not found",
			})
		} else if found.Err() == nil {
			found.Decode(&room)
		} else {
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(fiber.Map{
				"message": "Internal error",
			})
		}

		if room.Author != uid {
			c.Status(fiber.StatusUnauthorized)
			return c.JSON(fiber.Map{
				"message": "Unauthorized",
			})
		}

		src, err := file.Open()
		if err != nil {
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(fiber.Map{
				"message": "Error opening file",
			})
		}
		defer src.Close()

		var img image.Image
		var decodeErr error
		if file.Header.Get("Content-Type") == "image/jpeg" {
			img, decodeErr = jpeg.Decode(src)
		} else if file.Header.Get("Content-Type") == "image/png" {
			img, decodeErr = png.Decode(src)
		} else {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "Unrecognized / unsupported format",
			})
		}
		if decodeErr != nil {
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(fiber.Map{
				"message": "Internal error",
			})
		}

		img = resize.Resize(200, 0, img, resize.Lanczos2)
		buf := &bytes.Buffer{}
		if err := jpeg.Encode(buf, img, nil); err != nil {
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(fiber.Map{
				"message": "Internal error",
			})
		}

		count, err := db.RoomImageCollection.CountDocuments(c.Context(), bson.M{"_id": roomId})
		if count != 0 {
			db.RoomImageCollection.UpdateByID(c.Context(), roomId, bson.M{"$set": bson.M{"binary": primitive.Binary{Data: buf.Bytes()}}})
		} else {
			db.RoomImageCollection.InsertOne(c.Context(), models.RoomImage{
				ID:     roomId,
				Binary: primitive.Binary{Data: buf.Bytes()},
			})
			println("Created image")
		}

		//send the updated chatroom image to all users through websocket api
		for conn := range chatServer.connections {
			if conn.Locals("uid").(primitive.ObjectID) != uid {
				conn.WriteJSON(fiber.Map{
					"ID":          roomId.Hex(),
					"base64image": "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()),
					"event_type":  "chatroom_update",
				})
			}
		}

		//clear the buffer. garbage collection does this automatically but this might be a little faster
		buf = nil

		c.Status(fiber.StatusOK)
		return c.JSON(fiber.Map{
			"message": "Updated image",
		})
	}
}

func DeleteRoom(chatServer *ChatServer) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		if c.Params("id") == "" {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "Bad request",
			})
		}

		oid, err := primitive.ObjectIDFromHex(c.Params("id"))
		if err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "Bad request",
			})
		}

		var uid primitive.ObjectID

		var room models.Room
		found := db.RoomCollection.FindOne(c.Context(), bson.M{"_id": oid})
		if found.Err() != nil {
			if found.Err() == mongo.ErrNoDocuments {
				c.Status(fiber.StatusNotFound)
				return c.JSON(fiber.Map{
					"message": "Room not found",
				})
			}
		} else {
			found.Decode(&room)
			uid, err := helpers.DecodeTokenAndGetUID(c)
			if err != nil {
				c.Status(fiber.StatusNotFound)
				return c.JSON(fiber.Map{
					"message": "Your session could not be found",
				})
			}
			if room.Author != uid {
				c.Status(fiber.StatusUnauthorized)
				return c.JSON(fiber.Map{
					"message": "Unauthorized",
				})
			}
		}

		res, err := db.RoomCollection.DeleteOne(c.Context(), bson.M{"_id": oid})
		db.RoomImageCollection.DeleteOne(c.Context(), bson.M{"_id": oid})

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

		//send the socket event that removes the chatroom for other users
		for conn := range chatServer.connections {
			if conn.Locals("uid").(primitive.ObjectID) != uid {
				conn.WriteJSON(fiber.Map{
					"ID":         oid.Hex(),
					"event_type": "chatroom_delete",
				})
			}
		}

		c.Status(fiber.StatusOK)
		return c.JSON(fiber.Map{
			"message": "Room deleted",
		})
	}
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