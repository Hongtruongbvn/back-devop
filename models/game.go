package models

import "go.mongodb.org/mongo-driver/bson/primitive"

type Game struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RawgID      int                `bson:"rawg_id" json:"rawg_id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description" json:"description"`
	ImageURL    string             `bson:"image_url" json:"image_url"`
	Genres      []string           `bson:"genres" json:"genres"`
	Platforms   []string           `bson:"platforms" json:"platforms"`
	Rating      float64            `bson:"rating" json:"rating"`
	Price       int                `bson:"price" json:"price"`
}
