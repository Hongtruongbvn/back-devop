package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Recharge struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"user_id"`
	Amount    int                `bson:"amount" json:"amount"`
	Status    string             `bson:"status" json:"status"` // pending, success, failed
	CreatedAt time.Time          `bson:"created_at" json:"created_at"`
}
