package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"freelance-platform/backend/database"
	"freelance-platform/backend/middleware"
	"freelance-platform/backend/models"
	"freelance-platform/backend/utils"
)

// Register creates a new user account.
// POST /register
func Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	req.Name  = strings.TrimSpace(req.Name)
	req.Phone = strings.TrimSpace(req.Phone)
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	if req.Name == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Full name is required")
		return
	}
	if len(req.Name) > 100 {
		utils.ErrorResponse(w, http.StatusBadRequest, "Name must be 100 characters or fewer")
		return
	}
	if !utils.IsValidPhone(req.Phone) {
		utils.ErrorResponse(w, http.StatusBadRequest, "A valid phone number is required (7–15 digits)")
		return
	}
	if !utils.IsValidEmail(req.Email) {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid email format")
		return
	}
	if !utils.IsStrongPassword(req.Password) {
		utils.ErrorResponse(w, http.StatusBadRequest, "Password must be at least 8 characters")
		return
	}

	// Only allow freelancer or employer self-registration
	req.Role = strings.ToLower(strings.TrimSpace(req.Role))
	if req.Role != "freelancer" && req.Role != "employer" {
		req.Role = "freelancer"
	}

	hashed, err := utils.HashPassword(req.Password)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error processing password")
		return
	}

	var id int
	err = database.DB.QueryRow(
		`INSERT INTO users (name, phone, email, password, role) VALUES ($1, $2, $3, $4, $5) RETURNING id`,
		req.Name, req.Phone, req.Email, hashed, req.Role,
	).Scan(&id)

	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "23505") ||
			strings.Contains(errStr, "unique") ||
			strings.Contains(errStr, "duplicate") {
			utils.ErrorResponse(w, http.StatusConflict, "An account with that email already exists")
			return
		}
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error creating account")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, map[string]interface{}{
		"message": "Registration successful",
		"id":      id,
	})
}

// Login authenticates a user and returns a JWT.
// POST /login
func Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	var user models.User
	err := database.DB.QueryRow(
		`SELECT id, name, phone, email, password, role, is_verified, verification_score, created_at, updated_at
		 FROM users WHERE email = $1`,
		req.Email,
	).Scan(
		&user.ID, &user.Name, &user.Phone, &user.Email, &user.Password,
		&user.Role, &user.IsVerified, &user.VerificationScore,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			utils.ErrorResponse(w, http.StatusUnauthorized, "Invalid email or password")
			return
		}
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error looking up account")
		return
	}

	if !utils.CheckPassword(req.Password, user.Password) {
		utils.ErrorResponse(w, http.StatusUnauthorized, "Invalid email or password")
		return
	}

	token, err := middleware.GenerateToken(user.ID, user.Role)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error generating token")
		return
	}

	user.Password = "" // never leak the hash
	utils.JSONResponse(w, http.StatusOK, models.LoginResponse{
		Token: token,
		User:  &user,
	})
}
