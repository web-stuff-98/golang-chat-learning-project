package seed

import (
	"bytes"
	"context"
	"image"
	"image/jpeg"
	"math/rand"

	"github.com/nfnt/resize"
	"github.com/web-stuff-98/golang-chat-learning-project/api/helpers"
	"github.com/web-stuff-98/golang-chat-learning-project/db"
	"github.com/web-stuff-98/golang-chat-learning-project/db/models"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func GenerateSeed(numUsers uint8, numRooms uint8) (uids []primitive.ObjectID, rids []primitive.ObjectID, err error) {
	// Initialize empty slices for user IDs and room IDs
	uids = make([]primitive.ObjectID, 0, numUsers)
	rids = make([]primitive.ObjectID, 0, numRooms)

	// Generate users
	for i := uint8(0); i < numUsers; i++ {
		uid, err := generateUser(i)
		if err != nil {
			return nil, nil, err
		}
		uids = append(uids, uid)
	}

	// Generate rooms
	for i := uint8(0); i < numRooms; i++ {
		// Choose a random user ID from the list of generated user IDs
		uid := uids[rand.Intn(len(uids))]
		rid, err := generateRoom(i, uid)
		if err != nil {
			return nil, nil, err
		}
		rids = append(rids, rid)
	}

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
	img = resize.Resize(64, 0, img, resize.Lanczos2)
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, nil); err != nil {
		return primitive.NilObjectID, err
	}
	inserted, err := db.UserCollection.InsertOne(context.TODO(), models.User{
		Username: "TestAcc" + string(i+1),
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
	var decodeErr error
	defer r.Close()
	img, decodeErr = jpeg.Decode(r)
	if decodeErr != nil {
		return primitive.NilObjectID, decodeErr
	}
	img = resize.Resize(200, 0, img, resize.Lanczos2)
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, nil); err != nil {
		return primitive.NilObjectID, err
	}
	inserted, err := db.RoomCollection.InsertOne(context.TODO(), models.Room{
		Name:   "Room " + string(i+1),
		Author: uid,
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
