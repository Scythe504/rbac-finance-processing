package server

import (
	"context"
	"net/http"
	"os"
	"slices"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/gorilla/mux"
	"github.com/scythe504/zorvyn-rbac-finance/internal/database"
	"github.com/scythe504/zorvyn-rbac-finance/internal/utils"
)

type contextKey string

const (
	contextKeyUserID   contextKey = "userId"
	contextKeyUserRole contextKey = "role"
)

func (s *Server) authMiddleWare(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		tokenStrings := strings.Split(authHeader, " ")
		if authHeader == "" || tokenStrings[0] != "Bearer" {
			utils.WriteError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		jwtToken := tokenStrings[1]

		secretKey := []byte(os.Getenv("JWT_SECRET"))
		token, err := jwt.ParseWithClaims(jwtToken, &utils.JwtClaims{}, func(token *jwt.Token) (interface{}, error) {
			return secretKey, nil
		}, jwt.WithValidMethods([]string{
			jwt.SigningMethodHS256.Alg(),
		}))

		if err != nil {
			utils.WriteError(w, http.StatusUnauthorized, "Invalid Token")
			return
		}

		claims, ok := token.Claims.(*utils.JwtClaims)

		if !ok || !token.Valid {
			utils.WriteError(w, http.StatusUnauthorized, "Invalid Token")
			return
		}

		// Attach user ID to request context
		r = r.WithContext(context.WithValue(r.Context(), contextKeyUserID, claims.UserID))
		r = r.WithContext(context.WithValue(r.Context(), contextKeyUserRole, claims.Role))

		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireRole(roles ...database.RoleType) mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			userRole := r.Context().Value(contextKeyUserRole).(database.RoleType)
			
			if slices.Contains(roles, userRole) {
				next.ServeHTTP(w, r)
				return
			}
			
			utils.WriteError(w, http.StatusForbidden, "Forbidden")
		})
	}
}