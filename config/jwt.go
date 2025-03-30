package config

import (
	
	"fmt"
	"os"
	

	"github.com/golang-jwt/jwt/v4"
)

// ValidateJWT extracts the user ID from a JWT token
func ValidateJWT(tokenString string) (string, error) {
    secret := os.Getenv("JWT_SECRET")
    fmt.Println("Incoming Token:", tokenString)  // Debug
    fmt.Println("JWT Secret Used for Validation:", secret)  // Debug

    token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
        return []byte(secret), nil
    })

    if err != nil {
        fmt.Println("JWT Validation Error:", err)  // 🔴 Print exact error
        return "", err
    }

    if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
        return claims["email"].(string), nil
    }

    return "", fmt.Errorf("invalid token")
}
