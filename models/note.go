package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Note struct {
	ID         primitive.ObjectID `bson:"_id"`
	Title      string             `json:"title"`
	Text       string             `json:"text"`
	Note_id    string             `json:"note_id"`
	Created_at time.Time          `json:"creatd _at"`
	Updated_at time.Time          `json:"updated_at"`
}
