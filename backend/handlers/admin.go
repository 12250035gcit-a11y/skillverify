package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"

	"freelance-platform/backend/database"
	"freelance-platform/backend/middleware"
	"freelance-platform/backend/utils"
)

func adminOnly(w http.ResponseWriter, r *http.Request) bool {
	if middleware.GetUserRole(r) != "admin" {
		utils.ErrorResponse(w, http.StatusForbidden, "Admin access required")
		return false
	}
	return true
}

// GetAllUsers lists every registered user.
// GET /admin/users
func GetAllUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	if !adminOnly(w, r) {
		return
	}

	rows, err := database.DB.Query(
		`SELECT id, name, email, phone, role, is_verified, verification_score, created_at
		 FROM users ORDER BY created_at DESC`,
	)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error fetching users")
		return
	}
	defer rows.Close()

	type UserRow struct {
		ID                int    `json:"id"`
		Name              string `json:"name"`
		Email             string `json:"email"`
		Phone             string `json:"phone"`
		Role              string `json:"role"`
		IsVerified        bool   `json:"is_verified"`
		VerificationScore int    `json:"verification_score"`
		CreatedAt         string `json:"created_at"`
	}

	users := []UserRow{}
	for rows.Next() {
		var u UserRow
		var name, phone sql.NullString
		if err := rows.Scan(&u.ID, &name, &u.Email, &phone, &u.Role,
			&u.IsVerified, &u.VerificationScore, &u.CreatedAt); err != nil {
			continue
		}
		u.Name = name.String
		u.Phone = phone.String
		users = append(users, u)
	}
	utils.JSONResponse(w, http.StatusOK, users)
}

// SetUserVerification lets an admin manually verify or unverify a freelancer.
// PUT /admin/verify-user?id=N   body: {"verified":true}
func SetUserVerification(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	if !adminOnly(w, r) {
		return
	}

	userIDStr := r.URL.Query().Get("id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID <= 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "Valid user id required")
		return
	}

	var req struct {
		Verified bool `json:"verified"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON")
		return
	}

	score := 0
	if req.Verified {
		score = 100
	}

	result, err := database.DB.Exec(
		`UPDATE users SET is_verified = $1, verification_score = $2
		 WHERE id = $3 AND role = 'freelancer'`,
		req.Verified, score, userID,
	)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error updating user")
		return
	}
	if n, _ := result.RowsAffected(); n == 0 {
		utils.ErrorResponse(w, http.StatusNotFound, "Freelancer not found")
		return
	}

	msg := "User verification removed"
	if req.Verified {
		msg = "User verified successfully"
	}
	utils.JSONResponse(w, http.StatusOK, map[string]string{"message": msg})
}

// GetAdminJobs lists every job on the platform including closed ones.
// GET /admin/jobs
func GetAdminJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	if !adminOnly(w, r) {
		return
	}

	rows, err := database.DB.Query(
		`SELECT j.id, j.title, j.is_open, j.created_at,
		        u.name AS employer_name, u.email AS employer_email,
		        COUNT(a.id) AS application_count
		 FROM jobs j
		 JOIN users u ON u.id = j.employer_id
		 LEFT JOIN applications a ON a.job_id = j.id
		 GROUP BY j.id, u.name, u.email
		 ORDER BY j.created_at DESC`,
	)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error fetching jobs")
		return
	}
	defer rows.Close()

	type AdminJob struct {
		ID               int    `json:"id"`
		Title            string `json:"title"`
		IsOpen           bool   `json:"is_open"`
		CreatedAt        string `json:"created_at"`
		EmployerName     string `json:"employer_name"`
		EmployerEmail    string `json:"employer_email"`
		ApplicationCount int    `json:"application_count"`
	}

	jobs := []AdminJob{}
	for rows.Next() {
		var j AdminJob
		var empName sql.NullString
		if err := rows.Scan(&j.ID, &j.Title, &j.IsOpen, &j.CreatedAt,
			&empName, &j.EmployerEmail, &j.ApplicationCount); err != nil {
			continue
		}
		j.EmployerName = empName.String
		jobs = append(jobs, j)
	}
	utils.JSONResponse(w, http.StatusOK, jobs)
}

// GetAdminStats returns extended platform stats for the admin dashboard.
// GET /admin/stats
func GetAdminStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	if !adminOnly(w, r) {
		return
	}

	var totalUsers, freelancers, employers, verified, openJobs, totalJobs, totalApps int
	database.DB.QueryRow(`SELECT COUNT(*) FROM users`).Scan(&totalUsers)
	database.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role='freelancer'`).Scan(&freelancers)
	database.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role='employer'`).Scan(&employers)
	database.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE is_verified=TRUE`).Scan(&verified)
	database.DB.QueryRow(`SELECT COUNT(*) FROM jobs WHERE is_open=TRUE`).Scan(&openJobs)
	database.DB.QueryRow(`SELECT COUNT(*) FROM jobs`).Scan(&totalJobs)
	database.DB.QueryRow(`SELECT COUNT(*) FROM applications`).Scan(&totalApps)

	utils.JSONResponse(w, http.StatusOK, map[string]int{
		"total_users":  totalUsers,
		"freelancers":  freelancers,
		"employers":    employers,
		"verified":     verified,
		"open_jobs":    openJobs,
		"total_jobs":   totalJobs,
		"applications": totalApps,
	})
}
