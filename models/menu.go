package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Menu struct {
	ID         primitive.ObjectID `bson:"_id"`
	Name       string             `json:"name" validate:"required"`
	Categoty   string             `json:"category" validate:"required"`
	Start_date *time.Time         `json:"start_date"`
	End_date   *time.Time         `json:"end_date"`
	Menu_id    string             `json:"menu_id"`
	Created_at time.Time          `json:"created_at"`
	Updated_at time.Time          `json:"updated_at"`
}
