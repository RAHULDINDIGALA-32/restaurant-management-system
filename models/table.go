package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Table struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Number_of_guests *int               `bson:"number_of_guests" json:"number_of_guests" validate:"required,min=1"`
	Table_number     *int               `bson:"table_number" json:"table_number" validate:"required,min=1"`
	Created_at       time.Time          `bson:"created_at" json:"created_at"`
	Updated_at       time.Time          `bson:"updated_at" json:"updated_at"`
	Table_id         string             `bson:"table_id" json:"table_id"`
}
