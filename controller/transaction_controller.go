package controllers

import (
	"context"
	"time"

	"go-mvc-demo/config"
	"go-mvc-demo/models"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func BuyGame(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	gameID := c.Param("id")

	userObjID, _ := primitive.ObjectIDFromHex(userID)
	gameObjID, _ := primitive.ObjectIDFromHex(gameID)

	var user models.User
	err := config.DB.Collection("users").FindOne(context.TODO(), bson.M{"_id": userObjID}).Decode(&user)
	if err != nil {
		c.JSON(404, gin.H{"error": "User not found"})
		return
	}

	var game models.Game
	err = config.DB.Collection("games").FindOne(context.TODO(), bson.M{"_id": gameObjID}).Decode(&game)
	if err != nil {
		c.JSON(404, gin.H{"error": "Game not found"})
		return
	}

	if user.CoinBalance < game.Price {
		c.JSON(400, gin.H{"error": "Insufficient coin balance"})
		return
	}

	// Trừ tiền và ghi nhận
	config.DB.Collection("users").UpdateOne(context.TODO(), bson.M{"_id": userObjID}, bson.M{
		"$inc": bson.M{"coin_balance": -game.Price},
	})

	purchase := models.Purchase{
		ID:         primitive.NewObjectID(),
		UserID:     userObjID,
		GameID:     gameObjID,
		PurchaseAt: time.Now(),
		Price:      game.Price,
	}

	_, err = config.DB.Collection("purchases").InsertOne(context.TODO(), purchase)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to record purchase"})
		return
	}

	c.JSON(200, gin.H{"message": "Purchase successful"})
}

type GameWithPurchaseInfo struct {
	ID          primitive.ObjectID `json:"id"`
	Title       string             `json:"title"`
	Image       string             `json:"image"`
	Price       int                `json:"price"`
	IsPurchased bool               `json:"is_purchased"`
	IsRented    bool               `json:"is_rented"`
	ExpireAt    *time.Time         `json:"expire_at,omitempty"`
}

func GetUserGames(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	userObjID, _ := primitive.ObjectIDFromHex(userID)

	// Get all purchases for the user
	purchaseCursor, err := config.DB.Collection("purchases").Find(context.TODO(), bson.M{"user_id": userObjID})
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch purchases"})
		return
	}
	defer purchaseCursor.Close(context.TODO())

	var purchases []models.Purchase
	if err = purchaseCursor.All(context.TODO(), &purchases); err != nil {
		c.JSON(500, gin.H{"error": "Failed to decode purchases"})
		return
	}

	// Get all rentals for the user (including expired ones)
	rentalCursor, err := config.DB.Collection("rentals").Find(context.TODO(), bson.M{"user_id": userObjID})
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch rentals"})
		return
	}
	defer rentalCursor.Close(context.TODO())

	var rentals []models.Rental
	if err = rentalCursor.All(context.TODO(), &rentals); err != nil {
		c.JSON(500, gin.H{"error": "Failed to decode rentals"})
		return
	}

	// Collect all unique game IDs
	gameIDs := make([]primitive.ObjectID, 0)
	gameMap := make(map[primitive.ObjectID]bool)

	for _, p := range purchases {
		if !gameMap[p.GameID] {
			gameMap[p.GameID] = true
			gameIDs = append(gameIDs, p.GameID)
		}
	}

	for _, r := range rentals {
		if !gameMap[r.GameID] {
			gameMap[r.GameID] = true
			gameIDs = append(gameIDs, r.GameID)
		}
	}

	// Get game details for all unique games
	var games []models.Game
	if len(gameIDs) > 0 {
		cursor, err := config.DB.Collection("games").Find(context.TODO(), bson.M{"_id": bson.M{"$in": gameIDs}})
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to fetch games"})
			return
		}
		defer cursor.Close(context.TODO())

		if err = cursor.All(context.TODO(), &games); err != nil {
			c.JSON(500, gin.H{"error": "Failed to decode games"})
			return
		}
	}

	// Create a map for quick game lookup
	gameInfoMap := make(map[primitive.ObjectID]models.Game)
	for _, g := range games {
		gameInfoMap[g.ID] = g
	}

	// Create response with combined information
	response := make([]GameWithPurchaseInfo, 0)

	// Process purchased games
	for _, p := range purchases {
		if game, exists := gameInfoMap[p.GameID]; exists {
			response = append(response, GameWithPurchaseInfo{
				ID:          game.ID,
				Title:       game.Name,
				Image:       game.ImageURL,
				Price:       game.Price,
				IsPurchased: true,
				IsRented:    false,
			})
		}
	}

	// Process rented games
	for _, r := range rentals {
		if game, exists := gameInfoMap[r.GameID]; exists {
			// Check if this game is already in response (purchased)
			found := false
			for i, res := range response {
				if res.ID == r.GameID {
					response[i].IsRented = true
					response[i].ExpireAt = &r.ExpireAt
					found = true
					break
				}
			}

			if !found {
				response = append(response, GameWithPurchaseInfo{
					ID:          game.ID,
					Title:       game.Name,
					Image:       game.ImageURL,
					Price:       game.Price,
					IsPurchased: false,
					IsRented:    true,
					ExpireAt:    &r.ExpireAt,
				})
			}
		}
	}

	c.JSON(200, gin.H{"games": response})
}
func RentGame(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	gameID := c.Param("id")

	userObjID, _ := primitive.ObjectIDFromHex(userID)
	gameObjID, _ := primitive.ObjectIDFromHex(gameID)

	var game models.Game
	err := config.DB.Collection("games").FindOne(context.TODO(), bson.M{"_id": gameObjID}).Decode(&game)
	if err != nil {
		c.JSON(404, gin.H{"error": "Game not found"})
		return
	}

	var user models.User
	config.DB.Collection("users").FindOne(context.TODO(), bson.M{"_id": userObjID}).Decode(&user)

	rentPrice := game.Price / 10
	if user.CoinBalance < rentPrice {
		c.JSON(400, gin.H{"error": "Insufficient coin balance"})
		return
	}

	config.DB.Collection("users").UpdateOne(context.TODO(), bson.M{"_id": userObjID}, bson.M{
		"$inc": bson.M{"coin_balance": -rentPrice},
	})

	now := time.Now()
	rental := models.Rental{
		ID:       primitive.NewObjectID(),
		UserID:   userObjID,
		GameID:   gameObjID,
		RentAt:   now,
		ExpireAt: now.Add(3 * 24 * time.Hour),
		Status:   "active",
	}

	_, err = config.DB.Collection("rentals").InsertOne(context.TODO(), rental)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to record rental"})
		return
	}

	c.JSON(200, gin.H{"message": "Game rented for 3 days"})
}

func CheckActiveRental(c *gin.Context) {
	userID := c.MustGet("user_id").(string)
	gameID := c.Param("id")

	userObjID, _ := primitive.ObjectIDFromHex(userID)
	gameObjID, _ := primitive.ObjectIDFromHex(gameID)

	var rental models.Rental
	err := config.DB.Collection("rentals").FindOne(context.TODO(), bson.M{
		"user_id": userObjID,
		"game_id": gameObjID,
		"status":  "active",
	}).Decode(&rental)

	if err != nil || time.Now().After(rental.ExpireAt) {
		c.JSON(200, gin.H{"active": false})
		return
	}

	c.JSON(200, gin.H{"active": true, "expire_at": rental.ExpireAt})
}
