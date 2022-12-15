package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	Username string `bson:"username"`
	Password string `bson:"password" json:"-"`
}

type Session struct {
	UID       primitive.ObjectID `bson:"_uid"` //The users ID
	ExpiresAt primitive.DateTime `bson:"exp"`  //using NewDateTimeFromTime to generate this value
	SocketId  string             `bson:"socket_id"`
}

type Room struct {
	Name      string             `bson:"name" json:"name"`
	Author    primitive.ObjectID `bson:"author_id" json:"author_id"`
	CreatedAt primitive.DateTime `bson:"created_at" json:"created_at"`
	UpdatedAt primitive.DateTime `bson:"updated_at" json:"updated_at"`
}
