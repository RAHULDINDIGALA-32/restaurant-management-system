package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Menu struct {
	ID         primitive.ObjectID `bson:"_id" json:"-"`
	Name       string             `bson:"name" json:"name" validate:"required,min=2,max=100"`
	Category   string             `bson:"category" json:"category" validate:"required"`
	Start_date *time.Time         `bson:"start_date,omitempty" json:"start_date"`
	End_date   *time.Time         `bson:"end_date,omitempty" json:"end_date"`
	Menu_id    string             `bson:"menu_id" json:"menu_id"`
	Created_at time.Time          `bson:"created_at" json:"created_at"`
	Updated_at time.Time          `bson:"updated_at" json:"updated_at"`
}
