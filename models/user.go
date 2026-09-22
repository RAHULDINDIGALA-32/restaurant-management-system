package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	FirstName *string            `bson:"first_name" json:"first_name" validate:"required,min=2,max=100"`
	LastName  *string            `bson:"last_name" json:"last_name" validate:"required,min=2,max=100"`
	Password  *string            `bson:"password" json:"-" validate:"required,min=6"`
	Email     *string            `bson:"email" json:"email" validate:"required,email"`
	Avatar    *string            `bson:"avatar,omitempty" json:"avatar,omitempty"`
	Phone     *string            `bson:"phone" json:"phone" validate:"required"`
	UserID    string             `bson:"user_id" json:"user_id"`
	Role      string             `bson:"role" json:"role"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time          `bson:"updated_at" json:"updated_at"`
}
