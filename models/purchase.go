package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Purchase struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID     primitive.ObjectID `bson:"user_id" json:"user_id"`
	GameID     primitive.ObjectID `bson:"game_id" json:"game_id"`
	PurchaseAt time.Time          `bson:"purchase_at" json:"purchase_at"`
	Price      int                `bson:"price" json:"price"`
}
