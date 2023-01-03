package controllers

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/jpeg"
	"image/png"
	"io/ioutil"
	"log"
	"math"
	"strings"
	"time"

	"github.com/web-stuff-98/golang-chat-learning-project/api/helpers"
	"github.com/web-stuff-98/golang-chat-learning-project/api/validator"
	"github.com/web-stuff-98/golang-chat-learning-project/db"
	"github.com/web-stuff-98/golang-chat-learning-project/models"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
	"github.com/google/uuid"
	"github.com/nfnt/resize"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type InboundMessage struct {
	ID                primitive.ObjectID `bson:"_id" json:"ID"`
	Content           string             `json:"content"`
	SenderUid         string             `json:"uid"`
	WsConn            *websocket.Conn    `json:"-"`
	HasAttachment     bool               `json:"has_attachment"`
	AttachmentPending bool               `json:"attachment_pending"`
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

func HandleTestRateLimit(c *fiber.Ctx) error {
	c.Status(fiber.StatusOK)
	return c.JSON(fiber.Map{
		"message": "OK",
	})
}

func NewServer() (*ChatServer, chan string, chan string, error) {
	//closeWsChan can be used to close websockets using the users id
	closeWsChan := make(chan string)
	//deleteUser channel is used in changestream, when a user is deleted delete all their messages, rooms and send ws event to other users0
	deleteUserChan := make(chan string)

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

	/* ------------------ Handle inbound messages ------------------ */
	go func() {
		for {
			msg := <-chatServer.inbound
			log.Println("Inbound message")
			if msg.Content == "" {
				msg.WsConn.WriteJSON(fiber.Map{
					"event_type": "chatroom_err",
					"content":    "You cannot submit an empty message",
				})
				return
			}
			if len(msg.Content) > 200 {
				msg.WsConn.WriteJSON(fiber.Map{
					"event_type": "chatroom_err",
					"content":    "Message too long. Max 200 characters",
				})
				return
			}
			for i := range chatServer.chatRooms {
				conn := chatServer.chatRooms[i].connectionsByUid[msg.SenderUid]
				if conn != nil {
					for connI := range chatServer.chatRooms[i].connections {
						if conn != connI && connI != nil {
							defer func() {
								if r := recover(); r != nil {
									log.Println("Recovered from nil pointer dereference: ", r)
								}
							}()
							err := connI.WriteJSON(msg)
							if err != nil {
								log.Println(err)
							} else {
								log.Println("Wrote inbound message to connection")
							}
						}
					}
				}
			}
		}
	}()

	/* ------------------ Close websocket channel ------------------ */
	go func() {
		for {
			uid := <-closeWsChan
			conn := chatServer.connectionsByUid[uid]
			delete(chatServer.connectionsByUid, uid)
			delete(chatServer.connections, conn)
		}
	}()

	/* ------------------ Delete user channel ------------------ */
	go func() {
		for {
			uid := <-deleteUserChan

			for conn := range chatServer.connections {
				if conn.Locals("uid").(primitive.ObjectID).Hex() != uid {
					conn.WriteJSON(fiber.Map{
						"ID":         uid,
						"event_type": "user_delete",
					})
				}
			}

			//iterate over each chatroom using mongodb cursor, delete rooms owned by the deleted user, and delete messages by the deleted user
			findOpts := options.Find().SetBatchSize(10)
			cursor, err := db.RoomCollection.Find(context.TODO(), bson.D{}, findOpts)
			if err != nil {
				log.Fatal("CURSOR ERR : ", err)
			}

			for cursor.Next(context.TODO()) {
				var doc models.Room
				err := cursor.Decode(&doc)
				if err != nil {
					log.Fatal("ERROR DECODING : ", err)
				}
				//delete rooms. cannot use deleteMany because the room image needs to be deleted too.
				if doc.Author.Hex() == uid {
					db.RoomCollection.DeleteOne(context.TODO(), bson.M{"_id": doc.ID})
					db.RoomImageCollection.DeleteOne(context.TODO(), bson.M{"_id": doc.ID})
				} else {
					//delete users messages in room. chatgpt for pipeline.
					pipeline := bson.D{
						{Key: "$set", Value: bson.D{
							{Key: "messages", Value: bson.A{
								bson.D{
									{Key: "$filter", Value: bson.D{
										{Key: "input", Value: "$messages"},
										{Key: "as", Value: "m"},
										{Key: "cond", Value: bson.D{
											{Key: "$ne", Value: bson.A{"$$m.uid", uid}},
										}},
									}},
								},
							}},
						}},
					}
					db.RoomCollection.UpdateOne(context.TODO(), bson.M{"_id": doc.ID}, pipeline)
				}
				//delete users pfp
				db.PfpCollection.DeleteOne(context.TODO(), bson.M{"_id": doc.ID})
			}

			closeWsChan <- uid
		}
	}()

	/* ------------------ Register ws connection channel ------------------ */
	go func() {
		for {
			c := <-chatServer.registerConn
			log.Println("Register connection")
			chatServer.connections[c] = true
			chatServer.connectionsByUid[c.Locals("uid").(primitive.ObjectID).Hex()] = c
		}
	}()
	/* ------------------ Unregister ws connection channel ------------------ */
	go func() {
		for {
			c := <-chatServer.unregisterConn
			log.Println("Unregister connection")
			delete(chatServer.connections, c)
			delete(chatServer.connectionsByUid, c.Locals("uid").(primitive.ObjectID).Hex())
		}
	}()
	/* ------------------ Register room connection channel ------------------ */
	go func() {
		for {
			c := <-chatServer.registerRoomConn
			log.Println("Register room connection")
			conn := chatServer.connectionsByUid[c.uid]
			foundRoom := false
			for i := range chatServer.chatRooms {
				if chatServer.chatRooms[i].roomId == c.id {
					log.Println("Room connection added")
					chatServer.chatRooms[i].connections[conn] = true
					chatServer.chatRooms[i].connectionsByUid[c.uid] = conn
					foundRoom = true
					break
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
				log.Println("Room connection added")
			}
		}
	}()
	/* ------------------ Unregister room connection channel ------------------ */
	go func() {
		for {
			c := <-chatServer.unregisterRoomConn
			log.Println("Unregister room connection")
			conn := chatServer.connectionsByUid[c.uid]
			for i := range chatServer.chatRooms {
				if chatServer.chatRooms[i].roomId == c.id {
					delete(chatServer.chatRooms[i].connections, conn)
					delete(chatServer.chatRooms[i].connectionsByUid, c.uid)
					break
				}
			}
		}
	}()

	return chatServer, closeWsChan, deleteUserChan, nil
}

/* ------------------ WS HTTP API ROUTES ------------------ */

func HandleWsConn(chatServer *ChatServer, closeWsChan chan string) func(*fiber.Ctx) error {
	return websocket.New(func(c *websocket.Conn) {
		chatServer.registerConn <- c
		for {
			var Msg models.MessageEvent
			if err := c.ReadJSON(&Msg); err != nil {
				break
			}
			msgId := primitive.NewObjectID()
			chatServer.inbound <- InboundMessage{
				WsConn:            c,
				Content:           Msg.Content,
				SenderUid:         c.Locals("uid").(primitive.ObjectID).Hex(),
				ID:                msgId,
				HasAttachment:     Msg.HasAttachment,
				AttachmentPending: Msg.HasAttachment,
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
							Content:           Msg.Content,
							Uid:               c.Locals("uid").(primitive.ObjectID).Hex(),
							Timestamp:         primitive.NewDateTimeFromTime(time.Now()),
							ID:                msgId,
							HasAttachment:     Msg.HasAttachment,
							AttachmentPending: Msg.HasAttachment,
						}
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

func HandleGetRooms(c *fiber.Ctx) error {
	var rooms []models.Room
	var findFilter bson.M = bson.M{}
	if c.Query("own") == "true" {
		findFilter = bson.M{"author_id": c.Locals("uid").(primitive.ObjectID)}
	}
	cur, err := db.RoomCollection.Find(c.Context(), findFilter)
	if err != nil {
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

func HandleGetRoom(c *fiber.Ctx) error {
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

func HandleGetRoomImage(c *fiber.Ctx) error {
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
			"message": "Invalid ID",
		})
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var img models.RoomImage
	found := db.RoomImageCollection.FindOne(ctx, bson.M{"_id": oid})
	if found.Err() != nil {
		if found.Err() != mongo.ErrNoDocuments {
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(fiber.Map{
				"message": "Internal error",
			})
		} else {
			c.Status(fiber.StatusNotFound)
			return c.JSON(fiber.Map{
				"message": "Room has no image",
			})
		}
	}
	found.Decode(&img)
	c.Status(fiber.StatusOK)
	c.Type("image/jpeg")
	return c.Send(img.Binary.Data)
}

func HandleCreateRoom(chatServer *ChatServer) fiber.Handler {
	return func(c *fiber.Ctx) error {
		var body validator.Room
		if err := c.BodyParser(&body); err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "Invalid request",
			})
		}

		found := db.RoomCollection.FindOne(c.Context(), bson.M{"author_id": c.Locals("uid").(primitive.ObjectID), "name": bson.M{"$regex": body.Name, "$options": "i"}})
		if found.Err() != nil {
			if found.Err() != mongo.ErrNoDocuments {
				c.Status(fiber.StatusInternalServerError)
				return c.JSON(fiber.Map{
					"message": "Internal error",
				})
			}
		} else {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "You already have another room by that name",
			})
		}

		res, err := db.RoomCollection.InsertOne(c.Context(), models.Room{
			Name:      body.Name,
			CreatedAt: primitive.NewDateTimeFromTime(time.Now()),
			UpdatedAt: primitive.NewDateTimeFromTime(time.Now()),
			Author:    c.Locals("uid").(primitive.ObjectID),
			Messages:  []models.Message{},
		})

		if err != nil {
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(fiber.Map{
				"message": "Internal error",
			})
		}

		for conn := range chatServer.connections {
			if conn.Locals("uid").(primitive.ObjectID) != c.Locals("uid").(primitive.ObjectID) {
				conn.WriteJSON(fiber.Map{
					"ID":         res.InsertedID.(primitive.ObjectID).Hex(),
					"name":       body.Name,
					"author_id":  c.Locals("uid").(primitive.ObjectID).Hex(),
					"event_type": "chatroom_update",
				})
			}
		}

		c.Status(fiber.StatusCreated)
		return c.JSON(fiber.Map{
			"ID":         res.InsertedID.(primitive.ObjectID).Hex(),
			"name":       body.Name,
			"created_at": primitive.NewDateTimeFromTime(time.Now()),
			"updated_at": primitive.NewDateTimeFromTime(time.Now()),
			"author_id":  c.Locals("uid").(primitive.ObjectID).Hex(),
		})
	}
}

// Updates the room name only
func HandleUpdateRoom(protectedRids *map[primitive.ObjectID]struct{}, chatServer *ChatServer) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var rids = *protectedRids

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

		oid, err := primitive.ObjectIDFromHex(c.Params("id"))
		if err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "Invalid ID",
			})
		}

		_, ok := rids[oid]
		if ok {
			c.Status(fiber.StatusUnauthorized)
			return c.JSON(fiber.Map{
				"message": "You cannot modify test rooms.",
			})
		}

		foundRoomsCursor, err := db.RoomCollection.Find(c.Context(), bson.M{"author_id": c.Locals("uid").(primitive.ObjectID), "name": bson.M{"$regex": body.Name, "$options": "i"}})
		if err != nil {
			if err != mongo.ErrNoDocuments {
				c.Status(fiber.StatusInternalServerError)
				return c.JSON(fiber.Map{
					"message": "Internal error",
				})
			}
		} else {
			for foundRoomsCursor.Next(c.Context()) {
				var room models.Room
				foundRoomsCursor.Decode(&room)
				if room.ID.Hex() != c.Params("id") {
					foundRoomsCursor.Close(c.Context())
					c.Status(fiber.StatusBadRequest)
					return c.JSON(fiber.Map{
						"message": "You already have a room by that name",
					})
				}
			}
		}
		defer foundRoomsCursor.Close(c.Context())

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
			if room.Author != c.Locals("uid").(primitive.ObjectID) {
				c.Status(fiber.StatusUnauthorized)
				return c.JSON(fiber.Map{
					"message": "Unauthorized",
				})
			}
		}

		db.RoomCollection.UpdateByID(c.Context(), oid, bson.D{{Key: "$set", Value: bson.D{{Key: "name", Value: body.Name}}}})

		for conn := range chatServer.connections {
			if conn.Locals("uid").(primitive.ObjectID) != c.Locals("uid").(primitive.ObjectID) {
				conn.WriteJSON(fiber.Map{
					"ID":   oid.Hex(),
					"name": body.Name,
				})
			}
		}

		c.Status(fiber.StatusOK)
		return c.JSON(fiber.Map{
			"message": "Room name updated",
		})
	}
}

const maxAttachmentSize = 20 * 1024 * 1024 //20mb

// Upload attachment for the users last message
func HandleUploadAttachment(chatServer *ChatServer) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		file, err := c.FormFile("file")
		if err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": err,
			})
		}

		if file.Size > maxAttachmentSize {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "File too large. Max 20mb.",
			})
		}

		if c.Params("roomId") == "" {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "Bad request",
			})
		}

		roomId, err := primitive.ObjectIDFromHex(c.Params("roomId"))
		if err != nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "Invalid room ID",
			})
		}

		var room models.Room
		foundRoom := db.RoomCollection.FindOne(c.Context(), bson.M{"_id": roomId})
		if foundRoom.Err() != nil {
			if foundRoom.Err() == mongo.ErrNoDocuments {
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
		foundRoom.Decode(&room)

		// Iterate through all the rooms messages to find the users most recent message
		var matchingMsg models.Message
		var foundMsg bool
		for _, msg := range room.Messages {
			if msg.Uid == c.Locals("uid").(primitive.ObjectID).Hex() {
				foundMsg = true
				matchingMsg = msg
			}
		}
		if !foundMsg {
			c.Status(fiber.StatusNotFound)
			return c.JSON(fiber.Map{
				"message": "Message not found",
			})
		}

		src, err := file.Open()
		if err != nil {
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(fiber.Map{
				"message": "Internal error",
			})
		}

		foundAttach := db.AttachmentCollection.FindOne(c.Context(), bson.M{"_id": matchingMsg.ID})
		if foundAttach.Err() == nil {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "Attachment already exists",
			})
		}

		var isJPEG, isPNG bool
		isJPEG = file.Header.Get("Content-Type") == "image/jpeg"
		isPNG = file.Header.Get("Content-Type") == "image/png"
		if isJPEG || isPNG {
			/* ----- Save file to db as resized image ----- */
			var img image.Image
			var decodeErr error
			if isJPEG {
				img, decodeErr = jpeg.Decode(src)
			}
			if isPNG {
				img, decodeErr = png.Decode(src)
			}
			if decodeErr != nil {
				c.Status(fiber.StatusInternalServerError)
				return c.JSON(fiber.Map{
					"message": "Internal error",
				})
			}
			buf := &bytes.Buffer{}
			width := math.Min(float64(img.Bounds().Dx()), 350)
			img = resize.Resize(uint(width), 0, img, resize.Lanczos2)
			db.AttachmentCollection.InsertOne(c.Context(), models.Attachment{
				ID:       matchingMsg.ID,
				MimeType: file.Header.Get("Content-Type"),
				Binary:   primitive.Binary{Data: buf.Bytes()},
			})
		} else {
			/* ----- Save file to db as misc downloadable file (no video player) ----- */
			data, err := ioutil.ReadAll(src)
			if err != nil {
				c.Status(fiber.StatusInternalServerError)
				return c.JSON(fiber.Map{
					"message": "Internal error",
				})
			}
			db.AttachmentCollection.InsertOne(c.Context(), models.Attachment{
				ID:       matchingMsg.ID,
				MimeType: file.Header.Get("Content-Type"),
				Binary:   primitive.Binary{Data: data},
			})
		}
		src.Close()

		/*I used chatgpt to help me figure this out... it got stuff wrong probably because
		its training data uses older versions of mongodb... had to correct it */
		arrayFilters := options.ArrayFilters{
			Filters: []interface{}{
				bson.M{"_id": matchingMsg.ID}, // remember try changing _id to i._id if it doesnt work
			},
		}
		db.RoomCollection.UpdateByID(c.Context(), roomId, []bson.M{
			{
				"$set": bson.M{
					"messages.$[i].attachment_pending": false,
				},
			},
		}, options.Update().SetArrayFilters(arrayFilters))

		// Emit attachment complete message to all connected clients in room
		for r := range chatServer.chatRooms {
			if chatServer.chatRooms[r].roomId == roomId.Hex() {
				for conn := range chatServer.chatRooms[r].connections {
					conn.WriteJSON(fiber.Map{
						"event_type": "attachment_complete",
						"ID":         matchingMsg.ID.Hex(),
					})
				}
			}
		}

		c.Status(fiber.StatusCreated)
		return c.JSON(fiber.Map{
			"message": "Attachment created",
		})
	}
}

const maxRoomImageSize = 20 * 1024 * 1024 //20mb

func HandleUploadRoomImage(chatServer *ChatServer) func(*fiber.Ctx) error {
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

		if room.Author != c.Locals("uid").(primitive.ObjectID) {
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
		var blurImg image.Image
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

		img = resize.Resize(400, 0, img, resize.Lanczos2)
		blurImg = resize.Resize(6, 2, img, resize.Lanczos2)
		buf := &bytes.Buffer{}
		blurBuf := &bytes.Buffer{}
		if err := jpeg.Encode(buf, img, nil); err != nil {
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(fiber.Map{
				"message": "Internal error",
			})
		}
		if err := jpeg.Encode(blurBuf, blurImg, nil); err != nil {
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
		}

		imgBlurB64 := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(blurBuf.Bytes())

		db.RoomCollection.UpdateByID(c.Context(), roomId, bson.M{"img_blur": imgBlurB64})

		//send the updated chatroom image to all users through websocket api
		for conn := range chatServer.connections {
			if conn.Locals("uid").(primitive.ObjectID) != c.Locals("uid").(primitive.ObjectID) {
				conn.WriteJSON(fiber.Map{
					"ID":         roomId.Hex(),
					"img_url":    "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes()),
					"img_blur":   imgBlurB64,
					"event_type": "chatroom_update",
				})
			}
		}

		//clear the buffer. garbage collection does this automatically but this might be a little faster
		buf = nil
		blurBuf = nil

		c.Status(fiber.StatusOK)
		return c.JSON(fiber.Map{
			"message": "Updated image",
		})
	}
}

func HandleDeleteRoom(chatServer *ChatServer, protectedRids *map[primitive.ObjectID]struct{}) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
		var rids = *protectedRids

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

		_, ok := rids[oid]
		if ok {
			c.Status(fiber.StatusUnauthorized)
			return c.JSON(fiber.Map{
				"message": "You cannot delete test rooms.",
			})
		}

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
			conn.WriteJSON(fiber.Map{
				"ID":         oid.Hex(),
				"event_type": "chatroom_delete",
			})
		}

		c.Status(fiber.StatusOK)
		return c.JSON(fiber.Map{
			"message": "Room deleted",
		})
	}
}

func HandleJoinRoom(chatServer *ChatServer) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
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
			found.Decode(&room)
		}

		chatServer.registerRoomConn <- ChatRoomConnectionRegistration{id: c.Params("id"), uid: c.Locals("uid").(primitive.ObjectID).Hex()}

		c.Status(fiber.StatusOK)
		return c.JSON(room)
	}
}

func HandleLeaveRoom(chatServer *ChatServer) func(*fiber.Ctx) error {
	return func(c *fiber.Ctx) error {
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

		chatServer.unregisterRoomConn <- ChatRoomConnectionRegistration{id: c.Params("id"), uid: c.Locals("uid").(primitive.ObjectID).Hex()}

		c.Status(fiber.StatusOK)
		return c.JSON(fiber.Map{
			"message": "Left room",
		})
	}
}
