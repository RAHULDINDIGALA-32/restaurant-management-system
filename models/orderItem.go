package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type OrderItem struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id,omitempty"`
	Quantity      *string            `bson:"quantity" json:"quantity" validate:"required,oneof=S M L"`
	Unit_price    *float64           `bson:"unit_price" json:"unit_price" validate:"required"`
	Food_id       *string            `bson:"food_id" json:"food_id" validate:"required"`
	Order_item_id string             `bson:"order_item_id" json:"order_item_id"`
	Order_id      string             `bson:"order_id" json:"order_id" validate:"required"`
	Created_at    time.Time          `bson:"created_at" json:"created_at"`
	Updated_at    time.Time          `bson:"updated_at" json:"updated_at"`
}
