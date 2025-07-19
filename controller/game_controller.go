package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"go-mvc-demo/config"
	"go-mvc-demo/models"
	"io"
	"log"
	"math"
	"math/rand"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const RAWG_API_KEY = "adcb1c5b37944e2d9b1a9e730d721f44" // <-- Thay bằng API key thật của bạn
func CreateGame(c *gin.Context) {
	err := c.Request.ParseMultipartForm(10 << 20)
	if err != nil {
		c.JSON(400, gin.H{"error": "Failed to parse form"})
		return
	}

	name := c.PostForm("name")
	description := c.PostForm("description")
	rawgID, _ := strconv.Atoi(c.PostForm("rawg_id"))
	rating, _ := strconv.ParseFloat(c.PostForm("rating"), 64)
	price, _ := strconv.Atoi(c.PostForm("price"))

	genres := strings.Split(c.PostForm("genres"), ",")
	platforms := strings.Split(c.PostForm("platforms"), ",")

	for i := range genres {
		genres[i] = strings.TrimSpace(genres[i])
	}
	for i := range platforms {
		platforms[i] = strings.TrimSpace(platforms[i])
	}

	file, header, err := c.Request.FormFile("image")
	if err != nil {
		c.JSON(400, gin.H{"error": "Image upload failed"})
		return
	}
	defer file.Close()

	os.MkdirAll("uploads", os.ModePerm)
	filename := fmt.Sprintf("uploads/%d_%s", time.Now().UnixNano(), header.Filename)
	out, err := os.Create(filename)
	if err != nil {
		c.JSON(500, gin.H{"error": "Cannot save image"})
		return
	}
	defer out.Close()
	io.Copy(out, file)

	game := models.Game{
		ID:          primitive.NewObjectID(),
		RawgID:      rawgID,
		Name:        name,
		Description: description,
		ImageURL:    "/" + filename,
		Genres:      genres,
		Platforms:   platforms,
		Rating:      rating,
		Price:       price,
	}

	_, err = config.DB.Collection("games").InsertOne(context.TODO(), game)
	if err != nil {
		log.Println("Error inserting game:", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	log.Println("Game created successfully:", game.Name)
	c.JSON(201, game)
}

func GetGames(c *gin.Context) {
	cacheKey := "games:all"
	val, err := config.RedisClient.Get(config.Ctx, cacheKey).Result()

	if err == nil {
		// cache hit
		var games []models.Game
		if err := json.Unmarshal([]byte(val), &games); err == nil {
			c.JSON(200, games)
			return
		}
	}

	// cache miss
	cursor, err := config.DB.Collection("games").Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(500, gin.H{"error": "Error retrieving games"})
		return
	}
	defer cursor.Close(context.TODO())

	var games []models.Game
	if err := cursor.All(context.TODO(), &games); err != nil {
		c.JSON(500, gin.H{"error": "Failed to decode games"})
		return
	}

	// Lưu vào cache 5 phút
	data, _ := json.Marshal(games)
	config.RedisClient.Set(config.Ctx, cacheKey, data, time.Minute*5)

	c.JSON(200, games)
}

func GetGameByID(c *gin.Context) {
	idParam := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid ID"})
		return
	}

	var game models.Game
	err = config.DB.Collection("games").FindOne(context.TODO(), bson.M{"_id": objectID}).Decode(&game)
	if err != nil {
		c.JSON(404, gin.H{"error": "Game not found"})
		return
	}
	c.JSON(200, game)
}

func DeleteGame(c *gin.Context) {
	idParam := c.Param("id")
	objectID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(400, gin.H{"error": "Invalid ID"})
		return
	}

	_, err = config.DB.Collection("games").DeleteOne(context.TODO(), bson.M{"_id": objectID})
	if err != nil {
		c.JSON(500, gin.H{"error": "Delete failed"})
		return
	}
	c.JSON(200, gin.H{"message": "Game deleted successfully"})
}

func FetchAndSaveGames(c *gin.Context) {
	totalPages := 250 // 250 pages * 40 games = 10,000 games
	pageSize := 40
	importedCount := 0

	for page := 1; page <= totalPages; page++ {
		url := fmt.Sprintf("https://api.rawg.io/api/games?key=%s&page_size=%d&page=%d", RAWG_API_KEY, pageSize, page)
		resp, err := http.Get(url)
		if err != nil {
			c.JSON(500, gin.H{"error": fmt.Sprintf("Failed to fetch from RAWG at page %d", page)})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to read response body"})
			return
		}

		var result map[string]interface{}
		err = json.Unmarshal(body, &result)
		if err != nil {
			c.JSON(500, gin.H{"error": "Failed to parse JSON"})
			return
		}

		results, ok := result["results"].([]interface{})
		if !ok {
			c.JSON(500, gin.H{"error": "Invalid data format"})
			return
		}

		for _, item := range results {
			gameMap := item.(map[string]interface{})

			rawgID := int(gameMap["id"].(float64))
			name := gameMap["name"].(string)

			image := ""
			if gameMap["background_image"] != nil {
				image = gameMap["background_image"].(string)
			}

			rating := 0.0
			if gameMap["rating"] != nil {
				rating = gameMap["rating"].(float64)
			}

			genresRaw := gameMap["genres"].([]interface{})
			var genres []string
			for _, g := range genresRaw {
				genres = append(genres, g.(map[string]interface{})["name"].(string))
			}

			platformsRaw := gameMap["platforms"].([]interface{})
			var platforms []string
			for _, p := range platformsRaw {
				pMap := p.(map[string]interface{})
				platform := pMap["platform"].(map[string]interface{})
				platforms = append(platforms, platform["name"].(string))
			}

			description := "No description available"

			price := 100 + rand.Intn(900)

			filter := bson.M{"rawg_id": rawgID}
			count, err := config.DB.Collection("games").CountDocuments(context.TODO(), filter)
			if err != nil {
				c.JSON(500, gin.H{"error": "Database error"})
				return
			}
			if count == 0 {
				game := models.Game{
					ID:          primitive.NewObjectID(),
					RawgID:      rawgID,
					Name:        name,
					Description: description,
					ImageURL:    image,
					Genres:      genres,
					Platforms:   platforms,
					Rating:      rating,
					Price:       price,
				}
				_, err := config.DB.Collection("games").InsertOne(context.TODO(), game)
				if err != nil {
					c.JSON(500, gin.H{"error": "Failed to insert into DB"})
					return
				}
				importedCount++
			}
		}
	}

	c.JSON(200, gin.H{"message": fmt.Sprintf("%d games imported", importedCount)})
}
func FetchGamesByPage(c *gin.Context) {
	pageStr := c.Query("page")
	if pageStr == "" {
		pageStr = "1"
	}
	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		c.JSON(400, gin.H{"error": "Invalid page"})
		return
	}

	pageSize := 100
	skip := (page - 1) * pageSize

	cursor, err := config.DB.Collection("games").Find(
		context.TODO(),
		bson.M{},
		options.Find().SetLimit(int64(pageSize)).SetSkip(int64(skip)),
	)
	if err != nil {
		c.JSON(500, gin.H{"error": "Error retrieving games"})
		return
	}
	defer cursor.Close(context.TODO())

	var games []models.Game
	if err := cursor.All(context.TODO(), &games); err != nil {
		c.JSON(500, gin.H{"error": "Failed to decode games"})
		return
	}

	totalGames, err := config.DB.Collection("games").CountDocuments(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(500, gin.H{"error": "Error counting games"})
		return
	}

	totalPages := int(math.Ceil(float64(totalGames) / float64(pageSize)))

	c.JSON(200, gin.H{
		"games":      games,
		"page":       page,
		"totalPages": totalPages,
		"totalGames": totalGames,
	})
}
func GetPurchasedGames(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDStr.(string)
	cacheKey := fmt.Sprintf("user:%s:purchases", userID)

	// Try cache first
	val, err := config.RedisClient.Get(config.Ctx, cacheKey).Result()
	if err == nil {
		var cached []gin.H
		if err := json.Unmarshal([]byte(val), &cached); err == nil {
			c.JSON(http.StatusOK, gin.H{"purchases": cached})
			return
		}
	}

	// Cache miss: proceed DB fetch
	objID, _ := primitive.ObjectIDFromHex(userID)
	cursor, err := config.DB.Collection("purchases").Find(context.TODO(), bson.M{"user_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch purchases"})
		return
	}
	defer cursor.Close(context.TODO())

	var purchases []models.Purchase
	if err := cursor.All(context.TODO(), &purchases); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode purchases"})
		return
	}

	var gameIDs []primitive.ObjectID
	for _, p := range purchases {
		gameIDs = append(gameIDs, p.GameID)
	}

	gameCursor, err := config.DB.Collection("games").Find(context.TODO(), bson.M{"_id": bson.M{"$in": gameIDs}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch games"})
		return
	}
	defer gameCursor.Close(context.TODO())

	gameMap := make(map[primitive.ObjectID]models.Game)
	for gameCursor.Next(context.TODO()) {
		var game models.Game
		if err := gameCursor.Decode(&game); err == nil {
			gameMap[game.ID] = game
		}
	}

	var result []gin.H
	for _, p := range purchases {
		game := gameMap[p.GameID]
		result = append(result, gin.H{
			"id":          p.ID.Hex(),
			"price":       p.Price,
			"purchase_at": p.PurchaseAt,
			"game": gin.H{
				"id":        game.ID.Hex(),
				"name":      game.Name,
				"image_url": game.ImageURL,
				"price":     game.Price,
			},
		})
	}

	// Save cache 3 phút
	if data, err := json.Marshal(result); err == nil {
		config.RedisClient.Set(config.Ctx, cacheKey, data, time.Minute*3)
	}

	c.JSON(http.StatusOK, gin.H{"purchases": result})
}

func GetRentedGames(c *gin.Context) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	userID := userIDStr.(string)
	cacheKey := fmt.Sprintf("user:%s:rentals", userID)

	val, err := config.RedisClient.Get(config.Ctx, cacheKey).Result()
	if err == nil {
		var cached []gin.H
		if err := json.Unmarshal([]byte(val), &cached); err == nil {
			c.JSON(http.StatusOK, gin.H{"rentals": cached})
			return
		}
	}

	objID, _ := primitive.ObjectIDFromHex(userID)
	cursor, err := config.DB.Collection("rentals").Find(context.TODO(), bson.M{"user_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch rentals"})
		return
	}
	defer cursor.Close(context.TODO())

	var rentals []models.Rental
	if err := cursor.All(context.TODO(), &rentals); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode rentals"})
		return
	}

	var gameIDs []primitive.ObjectID
	for _, r := range rentals {
		gameIDs = append(gameIDs, r.GameID)
	}

	gameCursor, err := config.DB.Collection("games").Find(context.TODO(), bson.M{"_id": bson.M{"$in": gameIDs}})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch games"})
		return
	}
	defer gameCursor.Close(context.TODO())

	gameMap := make(map[primitive.ObjectID]models.Game)
	for gameCursor.Next(context.TODO()) {
		var game models.Game
		if err := gameCursor.Decode(&game); err == nil {
			gameMap[game.ID] = game
		}
	}

	var results []gin.H
	for _, rental := range rentals {
		game := gameMap[rental.GameID]
		results = append(results, gin.H{
			"id":        rental.ID.Hex(),
			"rent_at":   rental.RentAt,
			"expire_at": rental.ExpireAt,
			"status":    rental.Status,
			"game": gin.H{
				"id":        game.ID.Hex(),
				"name":      game.Name,
				"image_url": game.ImageURL,
				"price":     game.Price,
			},
		})
	}

	// Cache 3 phút
	if data, err := json.Marshal(results); err == nil {
		config.RedisClient.Set(config.Ctx, cacheKey, data, time.Minute*3)
	}

	c.JSON(http.StatusOK, gin.H{"rentals": results})
}
