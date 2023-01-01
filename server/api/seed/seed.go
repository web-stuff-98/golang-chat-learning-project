package seed

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"log"
	"math/rand"

	"github.com/nfnt/resize"
	"github.com/web-stuff-98/golang-chat-learning-project/api/helpers"
	"github.com/web-stuff-98/golang-chat-learning-project/db"
	"github.com/web-stuff-98/golang-chat-learning-project/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GenerateSeed(numUsers uint8, numRooms uint8) (uids map[primitive.ObjectID]struct{}, rids map[primitive.ObjectID]struct{}, err error) {
	// Drop DB
	db.DB.Drop(context.TODO())

	log.Println("Generating seed...")

	// Initialize empty maps for user IDs and room IDs
	uids = make(map[primitive.ObjectID]struct{})
	rids = make(map[primitive.ObjectID]struct{})

	// Generate users
	for i := uint8(0); i < numUsers; i++ {
		uid, err := generateUser(i)
		if err != nil {
			return nil, nil, err
		}
		uids[uid] = struct{}{}
	}

	// Generate rooms
	for i := uint8(0); i < numRooms; i++ {
		// Choose a random user ID from the map of generated user IDs
		uid := randomKey(uids)
		rid, err := generateRoom(i, uid)
		if err != nil {
			return nil, nil, err
		}
		rids[rid] = struct{}{}
	}

	log.Println("Seed generated.")

	return uids, rids, nil
}

func generateUser(i uint8) (uid primitive.ObjectID, err error) {
	r := helpers.DownloadRandomImage(true)
	var img image.Image
	var decodeErr error
	defer r.Close()
	img, decodeErr = jpeg.Decode(r)
	if decodeErr != nil {
		return primitive.NilObjectID, decodeErr
	}
	img = resize.Resize(36, 0, img, resize.Lanczos2)
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, nil); err != nil {
		return primitive.NilObjectID, err
	}
	inserted, err := db.UserCollection.InsertOne(context.TODO(), models.User{
		Username: fmt.Sprintf("TestAcc%d", i+1),
		Password: "$2a$12$VyvB4n4y8eq6mX8of9A3OOv/FRSzxSe54sk6ptifiT82RMtGpPI4a",
	})
	if err != nil {
		return primitive.NilObjectID, err
	}
	if db.PfpCollection.InsertOne(context.TODO(), models.Pfp{
		ID:     inserted.InsertedID.(primitive.ObjectID),
		Binary: primitive.Binary{Data: buf.Bytes()},
	}); err != nil {
		return primitive.NilObjectID, err
	}
	buf = nil
	return inserted.InsertedID.(primitive.ObjectID), nil
}

func generateRoom(i uint8, uid primitive.ObjectID) (rid primitive.ObjectID, err error) {
	r := helpers.DownloadRandomImage(false)
	var img image.Image
	var imgBlur image.Image
	var decodeErr error
	defer r.Close()
	img, decodeErr = jpeg.Decode(r)
	if decodeErr != nil {
		return primitive.NilObjectID, decodeErr
	}
	img = resize.Resize(220, 0, img, resize.Lanczos2)
	imgBlur = resize.Resize(6, 1, img, resize.Lanczos2)
	buf := &bytes.Buffer{}
	blurBuf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, nil); err != nil {
		return primitive.NilObjectID, err
	}
	if err := jpeg.Encode(blurBuf, imgBlur, nil); err != nil {
		return primitive.NilObjectID, err
	}
	inserted, err := db.RoomCollection.InsertOne(context.TODO(), models.Room{
		Name:     fmt.Sprintf("Room %d", i+1),
		Author:   uid,
		Messages: []models.Message{},
		ImgBlur:  "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(blurBuf.Bytes()),
	})
	if err != nil {
		return primitive.NilObjectID, err
	}
	if db.RoomImageCollection.InsertOne(context.TODO(), models.RoomImage{
		ID:     inserted.InsertedID.(primitive.ObjectID),
		Binary: primitive.Binary{Data: buf.Bytes()},
	}); err != nil {
		return primitive.NilObjectID, err
	}
	return inserted.InsertedID.(primitive.ObjectID), nil
}

func randomKey(m map[primitive.ObjectID]struct{}) primitive.ObjectID {
	keys := make([]primitive.ObjectID, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys[rand.Intn(len(keys))]
}
