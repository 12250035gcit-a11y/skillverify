package handlers

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"freelance-platform/backend/database"
	"freelance-platform/backend/middleware"
	"freelance-platform/backend/models"
	"freelance-platform/backend/utils"
)

// GetJobs lists all open jobs, newest first.
// GET /jobs?page=1&limit=20
func GetJobs(w http.ResponseWriter, r *http.Request) {
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	if limit < 1 || limit > 100 {
		limit = 20
	}
	offset := (page - 1) * limit

	rows, err := database.DB.Query(
		`SELECT j.id, j.title, j.description, j.budget, j.budget_type, j.category, j.job_type,
		        j.experience, j.duration, j.location, j.skills, j.requirements, j.deadline,
		        j.employer_id, j.is_open, j.created_at, j.updated_at,
		        u.name AS employer_name, u.email AS employer_email
		 FROM jobs j
		 JOIN users u ON u.id = j.employer_id
		 WHERE j.is_open = TRUE
		 ORDER BY j.created_at DESC
		 LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error fetching jobs")
		return
	}
	defer rows.Close()

	type JobWithEmployer struct {
		models.Job
		EmployerName  string `json:"employer_name"`
		EmployerEmail string `json:"employer_email"`
	}

	jobs := []JobWithEmployer{}
	for rows.Next() {
		var job JobWithEmployer
		var budgetType, category, jobType, experience, duration, location, skills, requirements, deadline *string
		if err := rows.Scan(
			&job.ID, &job.Title, &job.Description,
			&job.Budget, &budgetType, &category, &jobType,
			&experience, &duration, &location, &skills, &requirements, &deadline,
			&job.EmployerID, &job.IsOpen,
			&job.CreatedAt, &job.UpdatedAt,
			&job.EmployerName, &job.EmployerEmail,
		); err != nil {
			continue
		}
		if budgetType   != nil { job.BudgetType   = *budgetType }
		if category     != nil { job.Category     = *category }
		if jobType      != nil { job.JobType      = *jobType }
		if experience   != nil { job.Experience   = *experience }
		if duration     != nil { job.Duration     = *duration }
		if location     != nil { job.Location     = *location }
		if skills       != nil { job.Skills       = *skills }
		if requirements != nil { job.Requirements = *requirements }
		if deadline     != nil { job.Deadline     = *deadline }
		jobs = append(jobs, job)
	}
	if err := rows.Err(); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error reading jobs")
		return
	}

	// Total count for pagination metadata
	var total int
	database.DB.QueryRow(`SELECT COUNT(*) FROM jobs WHERE is_open = TRUE`).Scan(&total)

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"jobs":  jobs,
		"total": total,
		"page":  page,
		"limit": limit,
	})
}

// CreateJob allows an employer to post a new job.
// POST /jobs  (requires employer or admin JWT)
func CreateJob(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r)
	if role != "employer" && role != "admin" {
		utils.ErrorResponse(w, http.StatusForbidden, "Only employers can post jobs")
		return
	}

	var req models.CreateJobRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	req.Title = strings.TrimSpace(req.Title)
	req.Description = strings.TrimSpace(req.Description)

	if req.Title == "" || req.Description == "" {
		utils.ErrorResponse(w, http.StatusBadRequest, "Title and description are required")
		return
	}
	if len(req.Title) > 255 {
		utils.ErrorResponse(w, http.StatusBadRequest, "Title must be 255 characters or fewer")
		return
	}
	if req.Budget != nil && *req.Budget <= 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "Budget must be a positive number")
		return
	}

	employerID := middleware.GetUserID(r)

	// Normalise optional enum fields — use NULL if empty so DB defaults apply
	var budgetType, category, jobType, experience, duration, location, skills, requirements, deadline interface{}
	if req.BudgetType   != "" { budgetType   = req.BudgetType }
	if req.Category     != "" { category     = req.Category }
	if req.JobType      != "" { jobType      = req.JobType }
	if req.Experience   != "" { experience   = req.Experience }
	if req.Duration     != "" { duration     = req.Duration }
	if req.Location     != "" { location     = req.Location }
	if req.Skills       != "" { skills       = req.Skills }
	if req.Requirements != "" { requirements = req.Requirements }
	if req.Deadline     != "" { deadline     = req.Deadline }

	var jobID int
	err := database.DB.QueryRow(
		`INSERT INTO jobs
		    (title, description, budget, budget_type, category, job_type,
		     experience, duration, location, skills, requirements, deadline, employer_id)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)
		 RETURNING id`,
		req.Title, req.Description, req.Budget, budgetType, category, jobType,
		experience, duration, location, skills, requirements, deadline, employerID,
	).Scan(&jobID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error creating job")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, map[string]interface{}{
		"message": "Job posted successfully",
		"id":      jobID,
	})
}

// Apply lets a freelancer apply for a job.
// POST /apply  (requires freelancer JWT)
func Apply(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r)
	if role != "freelancer" {
		utils.ErrorResponse(w, http.StatusForbidden, "Only freelancers can apply for jobs")
		return
	}

	var req models.ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	if req.JobID <= 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "A valid job_id is required")
		return
	}

	// Confirm the job exists and is open
	var isOpen bool
	err := database.DB.QueryRow(
		`SELECT is_open FROM jobs WHERE id = $1`, req.JobID,
	).Scan(&isOpen)
	if err == sql.ErrNoRows {
		utils.ErrorResponse(w, http.StatusNotFound, "Job not found")
		return
	}
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error checking job")
		return
	}
	if !isOpen {
		utils.ErrorResponse(w, http.StatusConflict, "This job is no longer accepting applications")
		return
	}

	freelancerID := middleware.GetUserID(r)
	coverNote := strings.TrimSpace(req.CoverNote)

	var appID int
	err = database.DB.QueryRow(
		`INSERT INTO applications (job_id, freelancer_id, cover_note)
		 VALUES ($1, $2, $3) RETURNING id`,
		req.JobID, freelancerID, nullableString(coverNote),
	).Scan(&appID)

	if err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "23505") ||
			strings.Contains(errStr, "unique") ||
			strings.Contains(errStr, "duplicate") {
			utils.ErrorResponse(w, http.StatusConflict, "You have already applied for this job")
			return
		}
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error submitting application")
		return
	}

	utils.JSONResponse(w, http.StatusCreated, map[string]interface{}{
		"message": "Application submitted successfully",
		"id":      appID,
	})
}

// GetMyApplications returns applications belonging to the logged-in freelancer.
// GET /my-applications  (requires freelancer JWT)
func GetMyApplications(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r)
	if role != "freelancer" {
		utils.ErrorResponse(w, http.StatusForbidden, "Freelancers only")
		return
	}

	freelancerID := middleware.GetUserID(r)
	rows, err := database.DB.Query(
		`SELECT a.id, a.job_id, a.freelancer_id, a.status, a.cover_note, a.created_at, a.updated_at,
		        j.title AS job_title, a.job_test_score,
		        (SELECT COUNT(*) FROM job_questions WHERE job_id = a.job_id) AS question_count
		 FROM applications a
		 JOIN jobs j ON j.id = a.job_id
		 WHERE a.freelancer_id = $1
		 ORDER BY a.created_at DESC`,
		freelancerID,
	)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error fetching applications")
		return
	}
	defer rows.Close()

	type AppWithTitle struct {
		models.Application
		JobTitle      string   `json:"job_title"`
		JobTestScore  *int     `json:"job_test_score"`
		QuestionCount int      `json:"question_count"`
	}

	apps := []AppWithTitle{}
	for rows.Next() {
		var a AppWithTitle
		var coverNote sql.NullString
		var testScore sql.NullInt64
		if err := rows.Scan(
			&a.ID, &a.JobID, &a.FreelancerID, &a.Status, &coverNote,
			&a.CreatedAt, &a.UpdatedAt, &a.JobTitle, &testScore, &a.QuestionCount,
		); err != nil {
			continue
		}
		a.CoverNote = coverNote.String
		if testScore.Valid {
			v := int(testScore.Int64)
			a.JobTestScore = &v
		}
		apps = append(apps, a)
	}

	utils.JSONResponse(w, http.StatusOK, apps)
}

// GetJobApplications lists all applications for a job the logged-in employer owns.
// GET /job-applications?job_id=123  (requires employer or admin JWT)
func GetJobApplications(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r)
	if role != "employer" && role != "admin" {
		utils.ErrorResponse(w, http.StatusForbidden, "Employers only")
		return
	}

	jobIDStr := r.URL.Query().Get("job_id")
	jobID, err := strconv.Atoi(jobIDStr)
	if err != nil || jobID <= 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "Valid job_id query parameter required")
		return
	}

	employerID := middleware.GetUserID(r)

	// Verify ownership (admin bypasses)
	if role != "admin" {
		var ownerID int
		err := database.DB.QueryRow(
			`SELECT employer_id FROM jobs WHERE id = $1`, jobID,
		).Scan(&ownerID)
		if err == sql.ErrNoRows {
			utils.ErrorResponse(w, http.StatusNotFound, "Job not found")
			return
		}
		if err != nil || ownerID != employerID {
			utils.ErrorResponse(w, http.StatusForbidden, "You do not own this job")
			return
		}
	}

	rows, err := database.DB.Query(
		`SELECT a.id, a.job_id, a.freelancer_id, a.status, a.cover_note, a.created_at, a.updated_at,
		        u.name AS freelancer_name, u.email AS freelancer_email,
		        u.is_verified, u.verification_score, a.job_test_score
		 FROM applications a
		 JOIN users u ON u.id = a.freelancer_id
		 WHERE a.job_id = $1
		 ORDER BY a.created_at DESC`,
		jobID,
	)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error fetching applications")
		return
	}
	defer rows.Close()

	type AppWithFreelancer struct {
		models.Application
		FreelancerName    string `json:"freelancer_name"`
		FreelancerEmail   string `json:"freelancer_email"`
		IsVerified        bool   `json:"is_verified"`
		VerificationScore int    `json:"verification_score"`
		JobTestScore      *int   `json:"job_test_score"`
	}

	apps := []AppWithFreelancer{}
	for rows.Next() {
		var a AppWithFreelancer
		var coverNote sql.NullString
		var testScore sql.NullInt64
		if err := rows.Scan(
			&a.ID, &a.JobID, &a.FreelancerID, &a.Status, &coverNote,
			&a.CreatedAt, &a.UpdatedAt,
			&a.FreelancerName, &a.FreelancerEmail, &a.IsVerified, &a.VerificationScore, &testScore,
		); err != nil {
			continue
		}
		a.CoverNote = coverNote.String
		if testScore.Valid {
			v := int(testScore.Int64)
			a.JobTestScore = &v
		}
		apps = append(apps, a)
	}

	utils.JSONResponse(w, http.StatusOK, apps)
}

// UpdateApplicationStatus lets an employer change an application's status.
// PUT /application-status?id=123  (requires employer or admin JWT)
func UpdateApplicationStatus(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r)
	if role != "employer" && role != "admin" {
		utils.ErrorResponse(w, http.StatusForbidden, "Employers only")
		return
	}

	appIDStr := r.URL.Query().Get("id")
	appID, err := strconv.Atoi(appIDStr)
	if err != nil || appID <= 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "Valid application id query parameter required")
		return
	}

	var req models.UpdateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	validStatuses := map[string]bool{
		"pending": true, "shortlisted": true, "rejected": true, "hired": true,
	}
	if !validStatuses[req.Status] {
		utils.ErrorResponse(w, http.StatusBadRequest, "Status must be one of: pending, shortlisted, rejected, hired")
		return
	}

	employerID := middleware.GetUserID(r)

	// Employers may only update applications on their own jobs
	result, err := database.DB.Exec(
		`UPDATE applications a
		 SET status = $1
		 FROM jobs j
		 WHERE a.id = $2
		   AND a.job_id = j.id
		   AND ($3 = 0 OR j.employer_id = $3)`,
		req.Status, appID, employerID,
	)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error updating application status")
		return
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		utils.ErrorResponse(w, http.StatusNotFound, "Application not found or access denied")
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]string{
		"message": "Application status updated to " + req.Status,
	})
}

// nullableString returns nil for empty strings (maps to SQL NULL).
func nullableString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// GetStats returns public platform statistics.
// GET /stats  (public)
func GetStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	var openJobs, freelancers, employers int
	database.DB.QueryRow(`SELECT COUNT(*) FROM jobs WHERE is_open = TRUE`).Scan(&openJobs)
	database.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'freelancer'`).Scan(&freelancers)
	database.DB.QueryRow(`SELECT COUNT(*) FROM users WHERE role = 'employer'`).Scan(&employers)
	utils.JSONResponse(w, http.StatusOK, map[string]int{
		"open_jobs":   openJobs,
		"freelancers": freelancers,
		"employers":   employers,
	})
}
