package utils

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-gomail/gomail"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var jwtSecret = []byte("your_secret_key_here") // nên để trong biến môi trường

// Tạo token có thời hạn để reset password
func GenerateResetToken(email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"email": email,
		"exp":   time.Now().Add(10 * time.Minute).Unix(),
	})

	return token.SignedString(jwtSecret)
}

// Gửi mail chứa link reset password
func SendResetEmail(toEmail string, token string) error {
	resetLink := fmt.Sprintf("https://yourdomain.com/reset-password?token=%s", token)

	m := gomail.NewMessage()
	m.SetHeader("From", "your_email@example.com")
	m.SetHeader("To", toEmail)
	m.SetHeader("Subject", "Reset your password")
	m.SetBody("text/plain", fmt.Sprintf("Click the following link to reset your password:\n\n%s", resetLink))

	d := gomail.NewDialer("smtp.gmail.com", 587, "your_email@example.com", "your_email_password")

	return d.DialAndSend(m)
}
func VerifyResetToken(tokenString string) (string, error) {
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		return jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return "", errors.New("token không hợp lệ hoặc hết hạn")
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("không lấy được claims")
	}

	email, ok := claims["email"].(string)
	if !ok {
		return "", errors.New("email không hợp lệ trong token")
	}

	return email, nil
}
func HashPassword(password string) string {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// trong production nên xử lý lỗi nghiêm túc hơn
		panic(err)
	}
	return string(hashed)
}
func CheckPassword(plain, hashed string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(plain))
	return err == nil
}
