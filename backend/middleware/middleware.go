package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"freelance-platform/backend/utils"
)

type contextKey string

const (
	UserIDKey   contextKey = "userID"
	UserRoleKey contextKey = "userRole"
)

// tokenTTL is how long a JWT stays valid.
const tokenTTL = 72 * time.Hour

func getJWTSecret() []byte {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		// Warn loudly in development; production MUST set JWT_SECRET.
		secret = "skillverify-secret-key-change-in-production"
	}
	return []byte(secret)
}

func b64Encode(data []byte) string {
	return base64.RawURLEncoding.EncodeToString(data)
}

func b64Decode(s string) ([]byte, error) {
	return base64.RawURLEncoding.DecodeString(s)
}

// GenerateToken creates a signed HS256 JWT with an expiry claim.
func GenerateToken(userID int, role string) (string, error) {
	header := b64Encode([]byte(`{"alg":"HS256","typ":"JWT"}`))

	claimsJSON, err := json.Marshal(map[string]interface{}{
		"user_id": userID,
		"role":    role,
		"exp":     time.Now().Add(tokenTTL).Unix(),
		"iat":     time.Now().Unix(),
	})
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}

	payload := b64Encode(claimsJSON)
	sigInput := header + "." + payload
	mac := hmac.New(sha256.New, getJWTSecret())
	mac.Write([]byte(sigInput))
	sig := b64Encode(mac.Sum(nil))

	return sigInput + "." + sig, nil
}

// verifyToken validates signature and expiry, returning the claims map.
func verifyToken(tokenString string) (map[string]interface{}, error) {
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	// Verify signature
	sigInput := parts[0] + "." + parts[1]
	mac := hmac.New(sha256.New, getJWTSecret())
	mac.Write([]byte(sigInput))
	expectedSig := b64Encode(mac.Sum(nil))
	if !hmac.Equal([]byte(expectedSig), []byte(parts[2])) {
		return nil, fmt.Errorf("invalid token signature")
	}

	// Decode payload
	claimsData, err := b64Decode(parts[1])
	if err != nil {
		return nil, fmt.Errorf("invalid token payload")
	}

	var claims map[string]interface{}
	if err := json.Unmarshal(claimsData, &claims); err != nil {
		return nil, fmt.Errorf("invalid token claims")
	}

	// Check expiry
	if exp, ok := claims["exp"].(float64); ok {
		if time.Now().Unix() > int64(exp) {
			return nil, fmt.Errorf("token has expired")
		}
	}

	return claims, nil
}

// AuthMiddleware validates the Bearer token and injects user info into context.
func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.ErrorResponse(w, http.StatusUnauthorized, "Authorization header required")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			utils.ErrorResponse(w, http.StatusUnauthorized, "Invalid authorization format — expected: Bearer <token>")
			return
		}

		claims, err := verifyToken(parts[1])
		if err != nil {
			utils.ErrorResponse(w, http.StatusUnauthorized, "Invalid or expired token")
			return
		}

		// Safe type assertions
		userIDFloat, ok := claims["user_id"].(float64)
		if !ok || userIDFloat <= 0 {
			utils.ErrorResponse(w, http.StatusUnauthorized, "Malformed token: missing user_id")
			return
		}
		role, ok := claims["role"].(string)
		if !ok || role == "" {
			utils.ErrorResponse(w, http.StatusUnauthorized, "Malformed token: missing role")
			return
		}

		ctx := context.WithValue(r.Context(), UserIDKey, int(userIDFloat))
		ctx = context.WithValue(ctx, UserRoleKey, role)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// GetUserID retrieves the authenticated user's ID from the request context.
// Returns 0 if not set (unauthenticated route).
func GetUserID(r *http.Request) int {
	if id, ok := r.Context().Value(UserIDKey).(int); ok {
		return id
	}
	return 0
}

// GetUserRole retrieves the authenticated user's role from the request context.
func GetUserRole(r *http.Request) string {
	if role, ok := r.Context().Value(UserRoleKey).(string); ok {
		return role
	}
	return ""
}
