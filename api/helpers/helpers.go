package helpers

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/web-stuff-98/golang-chat-learning-project/db"
	"github.com/web-stuff-98/golang-chat-learning-project/db/models"

	"github.com/dgrijalva/jwt-go"
	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

/* ----------- HELPER/UTILITY FUNCTIONS ----------- */

// keepSocketId is true when refreshing the token, because otherwise the socket_id wont be preserved when the token refreshes
func GenerateToken(c *fiber.Ctx, uid primitive.ObjectID, expiresAt time.Time, keepSocketId bool) (string, error) {
	socketId := ""
	if keepSocketId {
		var session bson.M
		db.SessionCollection.FindOne(c.Context(), bson.M{"_uid": uid}).Decode(&session)
		if len(session) == 0 {
			return "", fmt.Errorf("Could not find original session.")
		}
		if session["socket_id"] == "" {
			return "", fmt.Errorf("socket_id is not available on session data.")
		}
		socketId = session["socket_id"].(string)
	}
	db.SessionCollection.DeleteMany(c.Context(), bson.M{"_uid": uid})
	inserted, err := db.SessionCollection.InsertOne(c.Context(), models.Session{
		UID:       uid,
		ExpiresAt: primitive.NewDateTimeFromTime(expiresAt),
		SocketId:  socketId,
	})
	if err != nil {
		return "", err
	}
	claims := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.StandardClaims{
		Issuer:    inserted.InsertedID.(primitive.ObjectID).Hex(), //the issuer is the session id
		ExpiresAt: expiresAt.Unix(),                               //not sure if this should be a unix timestamp
	})
	token, err := claims.SignedString([]byte(os.Getenv("SECRET")))
	return token, nil
}
func DecodeToken(c *fiber.Ctx) (*jwt.Token, error) {
	cookie := c.Cookies("session_token")
	token, err := jwt.ParseWithClaims(cookie, &jwt.StandardClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("SECRET")), nil
	})
	if err != nil {
		return nil, err
	}
	return token, nil
}
func DecodeTokenIssuer(c *fiber.Ctx) (string, error) {
	token, err := DecodeToken(c)
	if err != nil {
		return "", err
	}
	return token.Claims.(*jwt.StandardClaims).Issuer, nil
}
func GetSessionFromSID(c *fiber.Ctx, sid string) (bson.M, error) {
	oid, _ := primitive.ObjectIDFromHex(sid)
	var sessionData bson.M
	db.SessionCollection.FindOne(c.Context(), bson.M{"_id": oid}).Decode(&sessionData)
	if len(sessionData) == 0 {
		return nil, fmt.Errorf("Could not find session")
	}
	return sessionData, nil
}
func GetUserFromSID(c *fiber.Ctx, sid string) (bson.M, error) {
	session, err := GetSessionFromSID(c, sid)
	if err != nil {
		return nil, err
	}
	var userData bson.M
	db.UserCollection.FindOne(c.Context(), bson.M{"_id": session["_uid"]}).Decode(&userData)
	if len(userData) == 0 {
		return nil, fmt.Errorf("User does not exist")
	}

	return userData, nil
}
func DecodeTokenAndGetUID(c *fiber.Ctx) (primitive.ObjectID, error) {
	issuer, err := DecodeTokenIssuer(c)
	if err != nil {
		return primitive.NilObjectID, err
	}
	user, err := GetUserFromSID(c, issuer)
	if err != nil {
		return primitive.NilObjectID, err
	}
	return user["_id"].(primitive.ObjectID), nil
}
func AddSocketIdToSession(c *fiber.Ctx, socketId string) error {
	if c.Cookies("session_token") == "" {
		return fmt.Errorf("No cookie")
	}
	token, err := jwt.ParseWithClaims(c.Cookies("session_token"), &jwt.StandardClaims{}, func(t *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("SECRET")), nil
	})
	if err != nil {
		return err
	}
	oid, _ := primitive.ObjectIDFromHex(token.Claims.(*jwt.StandardClaims).Issuer)
	count, err := db.SessionCollection.CountDocuments(context.TODO(), bson.M{"_id": oid})
	if err != nil {
		return err
	}
	if count == 0 {
		return fmt.Errorf("Could not find session")
	}
	db.SessionCollection.UpdateOne(context.TODO(), bson.M{"_id": oid}, bson.D{{"$set", bson.D{{"socket_id", socketId}}}})
	log.Println("Added socket Id to session ", socketId)
	return nil
}
