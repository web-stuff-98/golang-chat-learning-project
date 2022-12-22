package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID       primitive.ObjectID `bson:"_id,omitempty"`
	Username string             `bson:"username"`
	Password string             `bson:"password" json:"-"`
}

type Pfp struct {
	ID     primitive.ObjectID `bson:"_id, omitempty"` //id should be the same id as the uid
	Binary primitive.Binary   `bson:"binary"`
}

type Session struct {
	UID       primitive.ObjectID `bson:"_uid"`
	ExpiresAt primitive.DateTime `bson:"exp"`
	SocketId  string             `bson:"socket_id"`
}

type Message struct {
	Content   string             `bson:"content" json:"content"`
	Uid       string             `bson:"uid" json:"uid"`
	Timestamp primitive.DateTime `bson:"timestamp" json:"timestamp"`
}

type Room struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"` // omitempty to protect against zeroed _id insertion
	Name      string             `bson:"name" json:"name"`
	Author    primitive.ObjectID `bson:"author_id" json:"author_id"`
	CreatedAt primitive.DateTime `bson:"created_at" json:"created_at"`
	UpdatedAt primitive.DateTime `bson:"updated_at" json:"updated_at"`
	Messages  []Message          `bson:"messages" json:"messages"`
}
