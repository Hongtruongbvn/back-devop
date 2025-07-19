package controllers

import (
	"context"
	"go-mvc-demo/config"
	"go-mvc-demo/models"
	"go-mvc-demo/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetUsers(c *gin.Context) {
	var users []models.User
	cursor, err := config.DB.Collection("users").Find(context.TODO(), bson.M{})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	defer cursor.Close(context.TODO())
	for cursor.Next(context.TODO()) {
		var user models.User
		cursor.Decode(&user)
		users = append(users, user)
	}
	c.JSON(http.StatusOK, users)
}

func GetUserByID(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var user models.User
	err = config.DB.Collection("users").FindOne(context.TODO(), bson.M{"_id": objID}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, user)
}

func CreateUser(c *gin.Context) {
	var input models.User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	input.ID = primitive.NewObjectID()
	input.CoinBalance = 1000
	input.Role = "user"

	_, err := config.DB.Collection("users").InsertOne(context.TODO(), input)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, input)
}

func UpdateUser(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var input models.User
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input"})
		return
	}

	update := bson.M{
		"$set": bson.M{
			"name":         input.Name,
			"email":        input.Email,
			"coin_balance": input.CoinBalance,
			"role":         input.Role,
			"updated_at":   time.Now(),
		},
	}

	_, err = config.DB.Collection("users").UpdateByID(context.TODO(), objID, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User updated"})
}

func DeleteUser(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	_, err = config.DB.Collection("users").DeleteOne(context.TODO(), bson.M{"_id": objID})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User deleted"})
}
func PromoteUserToAdmin(c *gin.Context) {
	idParam := c.Param("id")
	objID, err := primitive.ObjectIDFromHex(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	update := bson.M{
		"$set": bson.M{
			"role":       "admin",
			"updated_at": time.Now(),
		},
	}

	result, err := config.DB.Collection("users").UpdateByID(context.TODO(), objID, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if result.MatchedCount == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "User promoted to admin"})
}
func ForgotPasswordHandler(c *gin.Context) {
	// Lấy email từ JWT token đã được set bởi middleware
	emailVal, exists := c.Get("email")
	if !exists {
		c.JSON(400, gin.H{"error": "Không tìm thấy email trong token"})
		return
	}

	email, ok := emailVal.(string)
	if !ok || email == "" {
		c.JSON(400, gin.H{"error": "Email không hợp lệ trong token"})
		return
	}

	// Tạo token reset mật khẩu
	token, err := utils.GenerateResetToken(email)
	if err != nil {
		c.JSON(500, gin.H{"error": "Không tạo được token"})
		return
	}

	// Gửi email reset
	if err := utils.SendResetEmail(email, token); err != nil {
		c.JSON(500, gin.H{"error": "Gửi email thất bại"})
		return
	}

	c.JSON(200, gin.H{"message": "Đã gửi email hướng dẫn reset mật khẩu"})
}

func VerifyResetToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		email := claims["email"].(string)
		return email, nil
	}

	return "", err
}

var userCollection *mongo.Collection // khởi tạo trong init hoặc main

func ResetPasswordHandler(c *gin.Context) {
	var req struct {
		Email           string `json:"email"`
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu không hợp lệ"})
		return
	}

	// Tìm user theo email
	var user models.User
	err := userCollection.FindOne(context.TODO(), bson.M{"email": req.Email}).Decode(&user)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Email không tồn tại"})
		return
	}

	// Kiểm tra mật khẩu hiện tại có đúng không
	if !utils.CheckPassword(req.CurrentPassword, user.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Mật khẩu hiện tại không đúng"})
		return
	}

	// Hash và cập nhật mật khẩu mới
	hashedPassword := utils.HashPassword(req.NewPassword)
	update := bson.M{"$set": bson.M{"password": hashedPassword}}

	_, err = userCollection.UpdateOne(context.TODO(), bson.M{"email": req.Email}, update)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật mật khẩu"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Đổi mật khẩu thành công"})
}

func InitUserController(c *mongo.Collection) {
	userCollection = c
}
