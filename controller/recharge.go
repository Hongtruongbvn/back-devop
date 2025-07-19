package controllers

import (
	"context"
	"go-mvc-demo/config"
	"go-mvc-demo/models"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func RechargeCoin(c *gin.Context) {
	userIDStr, _ := c.Get("user_id")
	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Amount int `json:"amount"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.Amount < 100 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid amount (min 100)"})
		return
	}

	recharge := models.Recharge{
		UserID:    userID,
		Amount:    req.Amount,
		Status:    "success",
		CreatedAt: time.Now(),
	}

	rechargeCollection := config.DB.Collection("recharges")
	userCollection := config.DB.Collection("users")

	_, err = rechargeCollection.InsertOne(context.TODO(), recharge)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save recharge"})
		return
	}

	_, err = userCollection.UpdateOne(context.TODO(), bson.M{"_id": userID}, bson.M{
		"$inc": bson.M{"coin_balance": req.Amount},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update coin balance"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Recharge successful"})
}

func GetRechargeHistory(c *gin.Context) {
	userIDStr, _ := c.Get("user_id")
	userID, err := primitive.ObjectIDFromHex(userIDStr.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	rechargeCollection := config.DB.Collection("recharges")

	cursor, err := rechargeCollection.Find(context.TODO(), bson.M{"user_id": userID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch history"})
		return
	}
	defer cursor.Close(context.TODO())

	var recharges []models.Recharge
	if err := cursor.All(context.TODO(), &recharges); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Decode failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"recharges": recharges})
}
