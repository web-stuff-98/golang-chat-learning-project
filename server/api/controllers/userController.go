package controllers

import (
	"bytes"
	"cookie-session/api/helpers"
	"cookie-session/api/validator"
	"cookie-session/db"
	"cookie-session/db/models"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"strings"
	"time"

	"github.com/nfnt/resize"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"golang.org/x/crypto/bcrypt"
)

// close websocket connection using sid
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

	var user models.User
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

	var pfp models.Pfp
	pfperr := db.PfpCollection.FindOne(c.Context(), bson.M{"_id": user.ID}).Decode(&pfp)

	if pfperr == nil {
		user.Base64pfp = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(pfp.Binary.Data)
	}

	hashErr := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(body.Password))
	if hashErr != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Incorrect credentials",
		})
	}

	expiresAt := time.Now().Add(120 * time.Second)
	token, err := helpers.GenerateToken(c, user.ID, expiresAt, false)

	c.Cookie(&fiber.Cookie{
		Name:    "session_token",
		Value:   token,
		Expires: expiresAt,
	})

	return c.JSON(user)
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
			"_id": user["_id"],
		})
	}
}

const maxPfpSize = 20 * 1024 * 1024 //20mb

func UpdatePfp(c *fiber.Ctx) error {
	file, err := c.FormFile("file")
	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": err,
		})
	}

	if file.Size > maxPfpSize {
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

	src, err := file.Open()
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Error opening file",
		})
	}
	defer src.Close()

	// Decode the image from the file
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

	img = resize.Resize(64, 64, img, resize.Lanczos2)
	buf := &bytes.Buffer{}
	if err := jpeg.Encode(buf, img, nil); err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Internal error",
		})
	}

	uid, err := helpers.DecodeTokenAndGetUID(c)
	if err != nil {
		c.Status(fiber.StatusNotFound)
		return c.JSON(fiber.Map{
			"message": "Your session could not be found",
		})
	}

	count, err := db.PfpCollection.CountDocuments(c.Context(), bson.M{"_id": uid})
	if count != 0 {
		db.PfpCollection.UpdateByID(c.Context(), uid, models.Pfp{
			Binary: primitive.Binary{Data: buf.Bytes()},
		})
	} else {
		db.PfpCollection.InsertOne(c.Context(), models.Pfp{
			ID:     uid,
			Binary: primitive.Binary{Data: buf.Bytes()},
		})
	}

	//clear the buffer. garbage collection does this automatically but this might be a little faster
	buf = nil

	c.Status(fiber.StatusOK)
	return c.JSON(fiber.Map{
		"message": "Updated pfp",
	})
}

func GetUser(c *fiber.Ctx) error {
	uid, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		c.Status(fiber.StatusBadRequest)
		return c.JSON(fiber.Map{
			"message": "Invalid ID",
		})
	}

	var user models.User
	count, err := db.UserCollection.CountDocuments(c.Context(), bson.M{"_id": uid})
	if err != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Internal error",
		})
	} else if count == 0 {
		c.Status(fiber.StatusNotFound)
		return c.JSON(fiber.Map{
			"message": "User not found",
		})
	}
	db.UserCollection.FindOne(c.Context(), bson.M{"_id": uid}).Decode(&user)

	var pfp models.Pfp
	pfpcount, pfperr := db.PfpCollection.CountDocuments(c.Context(), bson.M{"_id": uid})
	if pfperr != nil {
		c.Status(fiber.StatusInternalServerError)
		return c.JSON(fiber.Map{
			"message": "Internal error",
		})
	} else if pfpcount != 0 {
		db.PfpCollection.FindOne(c.Context(), bson.M{"_id": uid}).Decode(&pfp)
		user.Base64pfp = "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(pfp.Binary.Data)
	}

	c.Status(fiber.StatusOK)
	return c.JSON(user)
}
