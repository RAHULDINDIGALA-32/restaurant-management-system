package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Session struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    string             `bson:"user_id" json:"user_id"`
	TokenHash string             `bson:"token_hash" json:"-"`
	ExpiresAt time.Time          `bson:"expires_at" json:"expires_at"`
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
	RevokedAt *time.Time         `bson:"revoked_at,omitempty" json:"revoked_at,omitempty"`
	UserAgent string             `bson:"user_agent,omitempty" json:"-"`
	IPAddress string             `bson:"ip_address,omitempty" json:"-"`
}
