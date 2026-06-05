package models

import "time"

type User struct {
	ID                int       `json:"id"`
	Name              string    `json:"name"`
	Phone             string    `json:"phone"`
	Email             string    `json:"email"`
	Password          string    `json:"-"`
	Role              string    `json:"role"`
	IsVerified        bool      `json:"is_verified"`
	VerificationScore int       `json:"verification_score"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type Job struct {
	ID           int       `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Budget       *float64  `json:"budget,omitempty"`
	BudgetType   string    `json:"budget_type,omitempty"`
	Category     string    `json:"category,omitempty"`
	JobType      string    `json:"job_type,omitempty"`
	Experience   string    `json:"experience,omitempty"`
	Duration     string    `json:"duration,omitempty"`
	Location     string    `json:"location,omitempty"`
	Skills       string    `json:"skills,omitempty"`
	Requirements string    `json:"requirements,omitempty"`
	Deadline     string    `json:"deadline,omitempty"`
	EmployerID   int       `json:"employer_id"`
	IsOpen       bool      `json:"is_open"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Application struct {
	ID           int       `json:"id"`
	JobID        int       `json:"job_id"`
	FreelancerID int       `json:"freelancer_id"`
	Status       string    `json:"status"`
	CoverNote    string    `json:"cover_note,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Question struct {
	ID            int      `json:"id"`
	Question      string   `json:"question"`
	Options       []string `json:"options"`
	CorrectAnswer int      `json:"-"`
}

// ── Request / Response DTOs ──────────────────────────────────────

type RegisterRequest struct {
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Email    string `json:"email"`
	Password string `json:"password"`
	Role     string `json:"role"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
	User  *User  `json:"user"`
}

type CreateJobRequest struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Budget       *float64 `json:"budget"`
	BudgetType   string   `json:"budget_type"`
	Category     string   `json:"category"`
	JobType      string   `json:"job_type"`
	Experience   string   `json:"experience"`
	Duration     string   `json:"duration"`
	Location     string   `json:"location"`
	Skills       string   `json:"skills"`
	Requirements string   `json:"requirements"`
	Deadline     string   `json:"deadline"`
}

type ApplyRequest struct {
	JobID     int    `json:"job_id"`
	CoverNote string `json:"cover_note"`
}

type UpdateApplicationRequest struct {
	Status string `json:"status"`
}

type TestAnswers struct {
	Answers map[int]int `json:"answers"`
}

type TestResult struct {
	Score      int    `json:"score"`
	IsVerified bool   `json:"is_verified"`
	Message    string `json:"message"`
}

type ErrorResponse struct {
	Message string `json:"message"`
}

// JobQuestion is a screening question set by an employer for a specific job.
type JobQuestion struct {
	ID          int       `json:"id"`
	JobID       int       `json:"job_id"`
	Question    string    `json:"question"`
	Options     []string  `json:"options"`
	CorrectIdx  int       `json:"-"` // never sent to applicants
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// JobQuestionInput is one question in a SetJobQuestions request.
type JobQuestionInput struct {
	Question   string   `json:"question"`
	Options    []string `json:"options"`
	CorrectIdx int      `json:"correct_idx"`
}

// SetJobQuestionsRequest replaces all questions for a job.
type SetJobQuestionsRequest struct {
	Questions []JobQuestionInput `json:"questions"`
}
