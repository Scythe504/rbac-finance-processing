package utils

import (
	"os"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/scythe504/rbac-finance-processing/internal/database"
)

type JwtClaims struct {
	UserID string            `json:"user_id"`
	Role   database.RoleType `json:"role"`
	jwt.RegisteredClaims
}

// GenerateJWTToken creates a new JWT token for a user
func GenerateJWTToken(userID string, role database.RoleType) (string, error) {
	claims := JwtClaims{
		UserID: userID,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: &jwt.NumericDate{
				Time: time.Now().Add(30 * 24 * time.Hour),
			},
			Issuer: "rbac-finance-processing.scythe",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	secretKey := []byte(os.Getenv("JWT_SECRET"))
	return token.SignedString(secretKey)
}
