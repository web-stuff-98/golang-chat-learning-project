package controllers

import (
	"cookie-session/api/helpers"
	"cookie-session/api/validator"
	"cookie-session/db"
	"cookie-session/db/models"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

func closeWsConn(c *fiber.Ctx, closeWsChan chan string, cookie string) error {
	if cookie == "" {
		return nil
	}
	issuer, err := helpers.DecodeTokenIssuer(c)
	if err != nil {
		return err
	}
	user, err := helpers.GetUserFromSID(c, issuer)
	if err != nil {
		return err
	}
	closeWsChan <- user["_id"].(primitive.ObjectID).Hex()
	return nil
}

func Register(c *fiber.Ctx) error {
	var body validator.Credentials
	if err := c.BodyParser(&body); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invalid request",
		})
	}
	count, err := db.UserCollection.CountDocuments(c.Context(), bson.M{"username": bson.M{"$regex": body.Username, "$options": "i"}})

	if err != nil {
		println(err)
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Internal error",
		})
	}
	if count != 0 {
		c.Status(400)
		return c.JSON(fiber.Map{
			"message": "There is a user by that name already",
		})
	}

	bytes, err := bcrypt.GenerateFromPassword([]byte(body.Password), 14)
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Internal error",
		})
	}

	inserted, err := db.UserCollection.InsertOne(c.Context(), models.User{
		Username: body.Username,
		Password: string(bytes),
	})

	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Could not create account. Needs better error handling here",
		})
	}

	expiresAt := time.Now().Add(120 * time.Second)
	token, err := helpers.GenerateToken(c, inserted.InsertedID.(primitive.ObjectID), expiresAt, false)

	c.Cookie(&fiber.Cookie{
		Name:    "session_token",
		Value:   token,
		Expires: expiresAt,
	})

	c.Status(201)
	return c.JSON(fiber.Map{
		"username": &body.Username,
		"_id":      inserted.InsertedID.(primitive.ObjectID).Hex(),
	})
}

func Login(c *fiber.Ctx) error {
	var body validator.Credentials
	if err := c.BodyParser(&body); err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invalid request",
		})
	}

	var user bson.M
	err := db.UserCollection.FindOne(c.Context(), bson.M{"username": bson.M{"$regex": body.Username, "$options": "i"}}).Decode(&user)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			c.Status(fiber.StatusBadRequest)
			return c.JSON(fiber.Map{
				"message": "Incorrect credentials",
			})
		}
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Internal error",
		})
	}

	hashErr := bcrypt.CompareHashAndPassword([]byte(user["password"].(string)), []byte(body.Password))
	if hashErr != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Incorrect credentials",
		})
	}

	expiresAt := time.Now().Add(120 * time.Second)
	token, err := helpers.GenerateToken(c, user["_id"].(primitive.ObjectID), expiresAt, false)

	c.Cookie(&fiber.Cookie{
		Name:    "session_token",
		Value:   token,
		Expires: expiresAt,
	})

	return c.JSON(fiber.Map{
		"username": &body.Username,
		"_id":      user["_id"],
	})
}

func Logout(closeWsChan chan string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Cookies("session_token", "") == "" {
			c.Status(fiber.StatusUnauthorized)
			return c.JSON(fiber.Map{
				"message": "You have no cookie",
			})
		}
		c.ClearCookie("session_token")
		err := closeWsConn(c, closeWsChan, c.Cookies("session_token"))
		if err != nil {
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(fiber.Map{
				"message": "Internal error",
			})
		}
		return c.SendStatus(fiber.StatusOK)
	}
}

func Welcome(c *fiber.Ctx) error {
	if c.Cookies("session_token", "") == "" {
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Unauthorized",
		})
	}

	issuer, err := helpers.DecodeTokenIssuer(c)
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Internal error",
		})
	}

	var session bson.M
	db.SessionCollection.FindOne(c.Context(), bson.M{"_id": issuer}).Decode(&session)
	fmt.Print(session)
	exp := session["exp"].(primitive.DateTime).Time()
	if time.Now().After(exp) {
		db.SessionCollection.DeleteOne(c.Context(), bson.M{"_id": issuer})
		c.Status(fiber.StatusUnauthorized)
		return c.JSON(fiber.Map{
			"message": "Your token has expired",
		})
	}

	var user bson.M
	db.UserCollection.FindOne(c.Context(), bson.M{"_id": session["_uid"]}).Decode(&user)

	return c.JSON(fiber.Map{
		"username": user["username"],
		"_id":      user["_id"],
	})
}

func Refresh(closeWsChan chan string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		if c.Cookies("session_token", "") == "" {
			c.Status(fiber.StatusUnauthorized)
			return c.JSON(fiber.Map{
				"message": "You are not logged in",
			})
		}

		issuer, err := helpers.DecodeTokenIssuer(c)
		if err != nil {
			c.Status(fiber.StatusUnauthorized)
			err := closeWsConn(c, closeWsChan, c.Cookies("session_token"))
			if err != nil {
				c.Status(fiber.StatusInternalServerError)
				return c.JSON(fiber.Map{
					"message": "Internal error",
				})
			}
			return c.JSON(fiber.Map{
				"message": "Unauthorized",
			})
		}

		session, err := helpers.GetSessionFromSID(c, issuer)
		if err != nil {
			c.Status(fiber.StatusUnauthorized)
			err := closeWsConn(c, closeWsChan, c.Cookies("session_token"))
			if err != nil {
				c.Status(fiber.StatusInternalServerError)
				return c.JSON(fiber.Map{
					"message": "Internal error",
				})
			}
			return c.JSON(fiber.Map{
				"message": "Unauthorized",
			})
		}

		exp := session["exp"].(primitive.DateTime).Time()
		if time.Now().After(exp) {
			db.SessionCollection.DeleteOne(c.Context(), bson.M{"_id": issuer})
			c.Status(fiber.StatusUnauthorized)
			err := closeWsConn(c, closeWsChan, c.Cookies("session_token"))
			if err != nil {
				c.Status(fiber.StatusInternalServerError)
				return c.JSON(fiber.Map{
					"message": "Internal error",
				})
			}
			return c.JSON(fiber.Map{
				"message": "Your token has expired",
			})
		}

		user, err := helpers.GetUserFromSID(c, issuer)
		if err != nil {
			c.Status(fiber.StatusNotFound)
			err := closeWsConn(c, closeWsChan, c.Cookies("session_token"))
			if err != nil {
				c.Status(fiber.StatusInternalServerError)
				return c.JSON(fiber.Map{
					"message": "Internal error",
				})
			}
			return c.JSON(fiber.Map{
				"message": "Your account does not exist",
			})
		}

		expiresAt := time.Now().Add(120 * time.Second)
		token, err := helpers.GenerateToken(c, user["_id"].(primitive.ObjectID), expiresAt, true)
		if err != nil {
			fmt.Println(err)
			c.Status(fiber.StatusInternalServerError)
			return c.JSON(fiber.Map{
				"message": "There was an error refreshing your token",
			})
		}

		c.Cookie(&fiber.Cookie{
			Name:    "session_token",
			Value:   token,
			Expires: expiresAt,
		})

		return c.JSON(fiber.Map{
			"username": user["username"],
			"_id":      user["_id"],
		})
	}
}
