package utils

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"regexp"
	"strings"
)

var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
var phoneDigits = regexp.MustCompile(`\D`)

// IsValidEmail returns true when email matches the standard pattern.
func IsValidEmail(email string) bool {
	return emailRegex.MatchString(strings.TrimSpace(email))
}

// IsValidPhone returns true when the number contains 7–15 digits (E.164 range).
func IsValidPhone(phone string) bool {
	digits := phoneDigits.ReplaceAllString(phone, "")
	return len(digits) >= 7 && len(digits) <= 15
}

// IsStrongPassword requires at least 8 characters.
// Extend this function if you need digit / symbol requirements.
func IsStrongPassword(password string) bool {
	return len(password) >= 8
}

// ClampInt constrains value to [min, max].
func ClampInt(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

// HashPassword creates a salted SHA-256 hash.
// Stored format: "<32-hex-char salt>:<64-hex-char hash>"
//
// NOTE: SHA-256 is used here to avoid external dependencies. For a
// production system with higher security requirements, replace this with
// bcrypt (golang.org/x/crypto/bcrypt) or Argon2.
func HashPassword(password string) (string, error) {
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return "", err
	}
	salt := hex.EncodeToString(saltBytes) // 32 hex chars
	hash := sha256.Sum256([]byte(salt + password))
	return salt + ":" + hex.EncodeToString(hash[:]), nil
}

// CheckPassword verifies a plaintext password against a stored hash string.
// Uses constant-time comparison to prevent timing attacks.
func CheckPassword(password, stored string) bool {
	colonIdx := strings.IndexByte(stored, ':')
	if colonIdx < 0 {
		return false
	}
	salt := stored[:colonIdx]
	expectedHash := stored[colonIdx+1:]

	hash := sha256.Sum256([]byte(salt + password))
	actualHash := hex.EncodeToString(hash[:])

	// subtle.ConstantTimeCompare prevents timing side-channel leaks
	return subtle.ConstantTimeCompare([]byte(actualHash), []byte(expectedHash)) == 1
}
