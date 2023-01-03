package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty"`
	Username  string             `bson:"username,maxlength=15" json:"username"`
	Password  string             `bson:"password" json:"-"`
	Base64pfp string             `bson:"-" json:"base64pfp,omitempty"`
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
	ID                primitive.ObjectID `bson:"_id,omitempty" json:"ID"` // omitempty to protect against zeroed _id insertion
	Content           string             `bson:"content,maxlength=200" json:"content"`
	Uid               string             `bson:"uid" json:"uid"`
	Timestamp         primitive.DateTime `bson:"timestamp" json:"timestamp"`
	HasAttachment     bool               `bson:"has_attachment" json:"has_attachment"`
	AttachmentPending bool               `bson:"attachment_pending" json:"attachment_pending"`
}

//socket message JSON from the client
type MessageEvent struct {
	Content       string `json:"content"`
	HasAttachment bool   `json:"has_attachment"`
}

type Room struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"ID"` // omitempty to protect against zeroed _id insertion
	Name      string             `bson:"name,maxlength=24" json:"name"`
	Author    primitive.ObjectID `bson:"author_id" json:"author_id"`
	CreatedAt primitive.DateTime `bson:"created_at" json:"created_at"`
	UpdatedAt primitive.DateTime `bson:"updated_at" json:"updated_at"`
	Messages  []Message          `bson:"messages" json:"messages"`
	ImgBlur   string             `bson:"img_blur" json:"img_blur,omitempty"`
}

type RoomImage struct {
	ID     primitive.ObjectID `bson:"_id, omitempty"` //should be the same as the rooms id
	Binary primitive.Binary   `bson:"binary"`
}

type Attachment struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"ID"` // ID should be the same as the message
	MimeType string             `bson:"mime_type" json:"mime_type"`
	Binary   primitive.Binary   `bson:"binary"`
}

//this is for the socket event when a user updates their profile
//i dont know why i only gave this socket event a model, the other ones need models too
//i should add them to here at some point to keep consistency
type UserUpdateEvent struct {
	UID       string
	base64pfp string
}
