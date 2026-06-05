package routes

import (
	"net/http"

	"freelance-platform/backend/handlers"
	"freelance-platform/backend/middleware"
)

// corsMiddleware adds CORS headers and handles preflight OPTIONS requests.
// In production, replace "*" with your actual frontend origin.
func corsMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	}
}

// auth wraps a handler with CORS + JWT authentication.
func auth(next http.HandlerFunc) http.HandlerFunc {
	return corsMiddleware(middleware.AuthMiddleware(next))
}

// SetupRoutes registers all routes on a new ServeMux and returns it.
// Using a custom mux (instead of http.DefaultServeMux) avoids global state
// and makes the server easier to test.
func SetupRoutes() *http.ServeMux {
	mux := http.NewServeMux()

	// ── Static frontend ──────────────────────────────────────────────────
	fs := http.FileServer(http.Dir("./frontend"))
	mux.Handle("/", fs)

	// ── Auth (public) ────────────────────────────────────────────────────
	mux.HandleFunc("/register", corsMiddleware(handlers.Register))
	mux.HandleFunc("/login", corsMiddleware(handlers.Login))

	// ── Jobs ─────────────────────────────────────────────────────────────
	// GET  /jobs  → public listing (with pagination)
	// POST /jobs  → employer creates a job (protected)
	mux.HandleFunc("/jobs", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetJobs(w, r)
		case http.MethodPost:
			middleware.AuthMiddleware(handlers.CreateJob)(w, r)
		default:
			w.Header().Set("Allow", "GET, POST, OPTIONS")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	// ── Applications ─────────────────────────────────────────────────────
	// POST /apply                     → freelancer applies (protected)
	// GET  /my-applications           → freelancer's own applications (protected)
	// GET  /job-applications?job_id=N → employer sees applicants (protected)
	// PUT  /application-status?id=N   → employer updates status (protected)
	mux.HandleFunc("/apply", auth(handlers.Apply))
	mux.HandleFunc("/my-applications", auth(handlers.GetMyApplications))
	mux.HandleFunc("/job-applications", auth(handlers.GetJobApplications))
	mux.HandleFunc("/application-status", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			w.Header().Set("Allow", "PUT, OPTIONS")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		middleware.AuthMiddleware(handlers.UpdateApplicationStatus)(w, r)
	}))

	// ── Public stats ─────────────────────────────────────────────────────
	mux.HandleFunc("/stats", corsMiddleware(handlers.GetStats))

	// ── Skill assessment ─────────────────────────────────────────────────
	mux.HandleFunc("/test-questions", auth(handlers.GetTestQuestions))
	mux.HandleFunc("/take-test", auth(handlers.TakeTest))
	mux.HandleFunc("/take-job-test", auth(handlers.TakeJobTest))

	// ── Employer tools ────────────────────────────────────────────────────
	// GET  /my-jobs                      → employer's own jobs + application counts
	// GET  /job-questions?job_id=N       → screening questions for a job (public)
	// POST /job-questions?job_id=N       → employer sets/replaces questions (protected)
	mux.HandleFunc("/my-jobs", auth(handlers.GetMyJobs))
	mux.HandleFunc("/job-questions-employer", auth(handlers.GetJobQuestionsEmployer))
	mux.HandleFunc("/job-questions", corsMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			handlers.GetJobQuestions(w, r)
		case http.MethodPost:
			middleware.AuthMiddleware(handlers.SetJobQuestions)(w, r)
		default:
			w.Header().Set("Allow", "GET, POST, OPTIONS")
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))

	// ── Admin ────────────────────────────────────────────────────
	mux.HandleFunc("/admin/stats",       auth(handlers.GetAdminStats))
	mux.HandleFunc("/admin/users",       auth(handlers.GetAllUsers))
	mux.HandleFunc("/admin/verify-user", auth(handlers.SetUserVerification))
	mux.HandleFunc("/admin/jobs",        auth(handlers.GetAdminJobs))

	return mux
}
