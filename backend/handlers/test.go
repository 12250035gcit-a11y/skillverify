package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"freelance-platform/backend/database"
	"freelance-platform/backend/middleware"
	"freelance-platform/backend/models"
	"freelance-platform/backend/utils"
)

// skillQuestions is the built-in 10-question skill assessment.
var skillQuestions = []models.Question{
	{ID: 1, Question: "Which data structure uses LIFO (Last In, First Out)?",
		Options: []string{"Queue", "Stack", "Tree", "Graph"}, CorrectAnswer: 1},
	{ID: 2, Question: "What does HTML stand for?",
		Options: []string{"Hyper Text Markup Language", "High Tech Modern Language", "Hyper Transfer Markup Language", "Home Tool Markup Language"}, CorrectAnswer: 0},
	{ID: 3, Question: "Which of the following is NOT a JavaScript data type?",
		Options: []string{"String", "Boolean", "Float", "Undefined"}, CorrectAnswer: 2},
	{ID: 4, Question: "What HTTP status code means 'Not Found'?",
		Options: []string{"200", "301", "404", "500"}, CorrectAnswer: 2},
	{ID: 5, Question: "Which SQL keyword is used to retrieve data from a table?",
		Options: []string{"GET", "FETCH", "SELECT", "RETRIEVE"}, CorrectAnswer: 2},
	{ID: 6, Question: "What does CSS stand for?",
		Options: []string{"Computer Style Sheets", "Cascading Style Sheets", "Creative Style System", "Colorful Style Sheets"}, CorrectAnswer: 1},
	{ID: 7, Question: "Which OOP principle hides internal implementation details?",
		Options: []string{"Inheritance", "Polymorphism", "Encapsulation", "Abstraction"}, CorrectAnswer: 2},
	{ID: 8, Question: "What is the time complexity of binary search?",
		Options: []string{"O(n)", "O(n²)", "O(log n)", "O(1)"}, CorrectAnswer: 2},
	{ID: 9, Question: "Which HTTP method is used to update an existing resource?",
		Options: []string{"GET", "POST", "PUT", "DELETE"}, CorrectAnswer: 2},
	{ID: 10, Question: "What does API stand for?",
		Options: []string{"Application Programming Interface", "Automated Program Integration", "Application Process Interface", "Advanced Programming Index"}, CorrectAnswer: 0},
}

// GetTestQuestions returns the questions without correct answers.
// GET /test-questions  (protected)
func GetTestQuestions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	type PublicQuestion struct {
		ID       int      `json:"id"`
		Question string   `json:"question"`
		Options  []string `json:"options"`
	}

	public := make([]PublicQuestion, len(skillQuestions))
	for i, q := range skillQuestions {
		public[i] = PublicQuestion{ID: q.ID, Question: q.Question, Options: q.Options}
	}

	utils.JSONResponse(w, http.StatusOK, public)
}

// TakeTest evaluates submitted answers, updates the user's verification status,
// and writes an audit record to test_results.
// POST /take-test  (protected — freelancers only)
func TakeTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}

	role := middleware.GetUserRole(r)
	if role != "freelancer" {
		utils.ErrorResponse(w, http.StatusForbidden, "Only freelancers take the skill test")
		return
	}

	var submission models.TestAnswers
	if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}
	if len(submission.Answers) == 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "No answers provided")
		return
	}

	correct := 0
	for _, q := range skillQuestions {
		if answer, ok := submission.Answers[q.ID]; ok && answer == q.CorrectAnswer {
			correct++
		}
	}

	total := len(skillQuestions)
	score := utils.ClampInt((correct*100)/total, 0, 100)
	isVerified := score >= 60

	userID := middleware.GetUserID(r)

	// Update user record
	_, err := database.DB.Exec(
		`UPDATE users SET is_verified = $1, verification_score = $2 WHERE id = $3`,
		isVerified, score, userID,
	)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error saving test result")
		return
	}

	// Persist audit log entry (ignore error — non-critical)
	database.DB.Exec(
		`INSERT INTO test_results (user_id, score, is_verified) VALUES ($1, $2, $3)`,
		userID, score, isVerified,
	)

	message := "You did not pass. A score of 60% or higher is required for verification. Please try again!"
	if isVerified {
		message = "Congratulations! You are now a verified freelancer."
	}

	utils.JSONResponse(w, http.StatusOK, models.TestResult{
		Score:      score,
		IsVerified: isVerified,
		Message:    message,
	})
}

// GetJobQuestions returns the screening questions for a job (without correct answers).
// GET /job-questions?job_id=N  (public — anyone can view to answer on application)
func GetJobQuestions(w http.ResponseWriter, r *http.Request) {
	jobIDStr := r.URL.Query().Get("job_id")
	jobID, err := strconv.Atoi(jobIDStr)
	if err != nil || jobID <= 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "Valid job_id query parameter required")
		return
	}

	rows, err := database.DB.Query(
		`SELECT id, job_id, question, options FROM job_questions WHERE job_id = $1 ORDER BY id`, jobID,
	)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error fetching questions")
		return
	}
	defer rows.Close()

	type PublicJobQuestion struct {
		ID       int      `json:"id"`
		JobID    int      `json:"job_id"`
		Question string   `json:"question"`
		Options  []string `json:"options"`
	}

	questions := []PublicJobQuestion{}
	for rows.Next() {
		var q PublicJobQuestion
		var optionsJSON []byte
		if err := rows.Scan(&q.ID, &q.JobID, &q.Question, &optionsJSON); err != nil {
			continue
		}
		json.Unmarshal(optionsJSON, &q.Options)
		questions = append(questions, q)
	}
	utils.JSONResponse(w, http.StatusOK, questions)
}

// SetJobQuestions lets an employer replace all screening questions for one of their jobs.
// POST /job-questions?job_id=N  (requires employer or admin JWT)
func SetJobQuestions(w http.ResponseWriter, r *http.Request) {
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

	// Verify ownership
	if role != "admin" {
		var ownerID int
		err := database.DB.QueryRow(`SELECT employer_id FROM jobs WHERE id = $1`, jobID).Scan(&ownerID)
		if err != nil {
			utils.ErrorResponse(w, http.StatusNotFound, "Job not found")
			return
		}
		if ownerID != employerID {
			utils.ErrorResponse(w, http.StatusForbidden, "You do not own this job")
			return
		}
	}

	var req models.SetJobQuestionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}

	if len(req.Questions) > 20 {
		utils.ErrorResponse(w, http.StatusBadRequest, "Maximum 20 questions allowed per job")
		return
	}

	// Replace all existing questions in a transaction
	tx, err := database.DB.Begin()
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Transaction error")
		return
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM job_questions WHERE job_id = $1`, jobID); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error clearing questions")
		return
	}

	for _, q := range req.Questions {
		q.Question = strings.TrimSpace(q.Question)
		if q.Question == "" || len(q.Options) < 2 {
			utils.ErrorResponse(w, http.StatusBadRequest, "Each question needs text and at least 2 options")
			return
		}
		optionsJSON, _ := json.Marshal(q.Options)
		if _, err := tx.Exec(
			`INSERT INTO job_questions (job_id, question, options, correct_idx) VALUES ($1,$2,$3,$4)`,
			jobID, q.Question, optionsJSON, q.CorrectIdx,
		); err != nil {
			utils.ErrorResponse(w, http.StatusInternalServerError, "Error saving question")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error committing questions")
		return
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"message": "Questions saved",
		"count":   len(req.Questions),
	})
}

// GetJobQuestionsEmployer returns screening questions with correct answers for the job owner.
// GET /job-questions-employer?job_id=N  (requires employer or admin JWT)
func GetJobQuestionsEmployer(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
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
	if role != "admin" {
		var ownerID int
		if err := database.DB.QueryRow(`SELECT employer_id FROM jobs WHERE id = $1`, jobID).Scan(&ownerID); err != nil {
			utils.ErrorResponse(w, http.StatusNotFound, "Job not found")
			return
		}
		if ownerID != employerID {
			utils.ErrorResponse(w, http.StatusForbidden, "You do not own this job")
			return
		}
	}

	rows, err := database.DB.Query(
		`SELECT id, job_id, question, options, correct_idx FROM job_questions WHERE job_id = $1 ORDER BY id`, jobID,
	)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error fetching questions")
		return
	}
	defer rows.Close()

	type EmployerJobQuestion struct {
		ID         int      `json:"id"`
		JobID      int      `json:"job_id"`
		Question   string   `json:"question"`
		Options    []string `json:"options"`
		CorrectIdx int      `json:"correct_idx"`
	}

	questions := []EmployerJobQuestion{}
	for rows.Next() {
		var q EmployerJobQuestion
		var optionsJSON []byte
		if err := rows.Scan(&q.ID, &q.JobID, &q.Question, &optionsJSON, &q.CorrectIdx); err != nil {
			continue
		}
		json.Unmarshal(optionsJSON, &q.Options)
		questions = append(questions, q)
	}
	utils.JSONResponse(w, http.StatusOK, questions)
}

// GetMyJobs returns all jobs posted by the logged-in employer.
// GET /my-jobs  (requires employer or admin JWT)
func GetMyJobs(w http.ResponseWriter, r *http.Request) {
	role := middleware.GetUserRole(r)
	if role != "employer" && role != "admin" {
		utils.ErrorResponse(w, http.StatusForbidden, "Employers only")
		return
	}

	employerID := middleware.GetUserID(r)
	rows, err := database.DB.Query(
		`SELECT j.id, j.title, j.is_open, j.created_at,
		        COUNT(a.id) AS application_count
		 FROM jobs j
		 LEFT JOIN applications a ON a.job_id = j.id
		 WHERE j.employer_id = $1
		 GROUP BY j.id
		 ORDER BY j.created_at DESC`,
		employerID,
	)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error fetching jobs")
		return
	}
	defer rows.Close()

	type JobSummary struct {
		ID               int    `json:"id"`
		Title            string `json:"title"`
		IsOpen           bool   `json:"is_open"`
		CreatedAt        string `json:"created_at"`
		ApplicationCount int    `json:"application_count"`
	}

	jobs := []JobSummary{}
	for rows.Next() {
		var j JobSummary
		if err := rows.Scan(&j.ID, &j.Title, &j.IsOpen, &j.CreatedAt, &j.ApplicationCount); err != nil {
			continue
		}
		jobs = append(jobs, j)
	}
	utils.JSONResponse(w, http.StatusOK, jobs)
}

// TakeJobTest grades a freelancer's answers for a specific job's screening questions.
// POST /take-job-test?job_id=N  (protected — freelancers only)
func TakeJobTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		utils.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
		return
	}
	role := middleware.GetUserRole(r)
	if role != "freelancer" {
		utils.ErrorResponse(w, http.StatusForbidden, "Freelancers only")
		return
	}

	jobIDStr := r.URL.Query().Get("job_id")
	jobID, err := strconv.Atoi(jobIDStr)
	if err != nil || jobID <= 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "Valid job_id query parameter required")
		return
	}

	userID := middleware.GetUserID(r)

	// Only shortlisted applicants may take the test
	var appID int
	err = database.DB.QueryRow(
		`SELECT id FROM applications WHERE job_id = $1 AND freelancer_id = $2 AND status = 'shortlisted'`,
		jobID, userID,
	).Scan(&appID)
	if err != nil {
		utils.ErrorResponse(w, http.StatusForbidden, "You must be shortlisted for this job before taking its test")
		return
	}

	// Fetch correct answers from the database
	rows, err := database.DB.Query(
		`SELECT id, correct_idx FROM job_questions WHERE job_id = $1 ORDER BY id`, jobID,
	)
	if err != nil {
		utils.ErrorResponse(w, http.StatusInternalServerError, "Error fetching questions")
		return
	}
	defer rows.Close()

	correctAnswers := map[int]int{}
	for rows.Next() {
		var id, ci int
		if rows.Scan(&id, &ci) == nil {
			correctAnswers[id] = ci
		}
	}
	if len(correctAnswers) == 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "This job has no screening questions")
		return
	}

	var submission models.TestAnswers
	if err := json.NewDecoder(r.Body).Decode(&submission); err != nil {
		utils.ErrorResponse(w, http.StatusBadRequest, "Invalid JSON request body")
		return
	}
	if len(submission.Answers) == 0 {
		utils.ErrorResponse(w, http.StatusBadRequest, "No answers provided")
		return
	}

	correct := 0
	for id, ca := range correctAnswers {
		if ans, ok := submission.Answers[id]; ok && ans == ca {
			correct++
		}
	}
	total := len(correctAnswers)
	score := utils.ClampInt((correct*100)/total, 0, 100)
	passed := score >= 60

	// Persist score on the application record
	database.DB.Exec(
		`UPDATE applications SET job_test_score = $1 WHERE id = $2`, score, appID,
	)

	message := "Your score has been saved. The employer will review your application."
	if passed {
		message = "Great score! The employer will see your result when reviewing applicants."
	}

	utils.JSONResponse(w, http.StatusOK, map[string]interface{}{
		"score":   score,
		"passed":  passed,
		"correct": correct,
		"total":   total,
		"message": message,
	})
}
