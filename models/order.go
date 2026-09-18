package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Order struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Order_date *time.Time         `bson:"order_date,omitempty" json:"order_date,omitempty" validate:"required"`
	Order_id   string             `bson:"order_id" json:"order_id"`
	Table_id   *string            `bson:"table_id,omitempty" json:"table_id,omitempty" validate:"required"`
	Created_at time.Time          `bson:"created_at" json:"created_at"`
	Updated_at time.Time          `bson:"updated_at" json:"updated_at"`
}
