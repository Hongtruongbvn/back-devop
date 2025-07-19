package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Rental struct {
	ID       primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID   primitive.ObjectID `bson:"user_id" json:"user_id"`
	GameID   primitive.ObjectID `bson:"game_id" json:"game_id"`
	RentAt   time.Time          `bson:"rent_at" json:"rent_at"`
	ExpireAt time.Time          `bson:"expire_at" json:"expire_at"`
	Status   string             `bson:"status" json:"status"` // active, expired
}
