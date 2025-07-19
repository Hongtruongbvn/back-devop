package config

import (
	"context"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DB *mongo.Database

func ConnectDB() *mongo.Database {
	clientOptions := options.Client().ApplyURI("mongodb://14.225.217.120:27017/")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	DB = client.Database("game_library")
	log.Println("✅ Đã kết nối MongoDB!")

	return DB // ✅ trả về *mongo.Database
}
