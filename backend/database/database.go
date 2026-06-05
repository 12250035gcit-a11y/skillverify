package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/lib/pq"
)

// DB is the shared database connection pool.
var DB *sql.DB

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}

// InitDB opens the database connection and runs auto-migrations.
func InitDB() {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "postgres")
	password := getEnv("DB_PASSWORD", "password")
	dbname := getEnv("DB_NAME", "freelance_platform")
	sslmode := getEnv("DB_SSLMODE", "disable")

	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		host, port, user, password, dbname, sslmode,
	)

	var err error
	DB, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	// Connection pool tuning
	DB.SetMaxOpenConns(25)
	DB.SetMaxIdleConns(10)
	DB.SetConnMaxLifetime(5 * time.Minute)

	if err = DB.Ping(); err != nil {
		log.Fatalf("Error connecting to database: %v\nCheck that PostgreSQL is running and credentials are correct.", err)
	}

	log.Println("Database connected successfully")
	createTables()
}

// createTables runs idempotent DDL on startup.
func createTables() {
	queries := []string{
		// ── ENUM types (safe to re-run) ─────────────────────────────────
		`DO $$ BEGIN
			CREATE TYPE user_role AS ENUM ('freelancer','employer','admin');
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

		`DO $$ BEGIN
			CREATE TYPE application_status AS ENUM ('pending','shortlisted','rejected','hired');
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

		`DO $$ BEGIN
			CREATE TYPE job_type AS ENUM ('remote','hybrid','onsite');
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

		`DO $$ BEGIN
			CREATE TYPE budget_type AS ENUM ('fixed','hourly','negotiable');
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

		`DO $$ BEGIN
			CREATE TYPE experience_level AS ENUM ('entry','mid','senior','any');
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

		`DO $$ BEGIN
			CREATE TYPE project_duration AS ENUM ('less_week','1_4_weeks','1_3_months','3_6_months','ongoing');
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

		// ── updated_at trigger function ──────────────────────────────────
		`CREATE OR REPLACE FUNCTION set_updated_at()
		RETURNS TRIGGER LANGUAGE plpgsql AS $$
		BEGIN NEW.updated_at = NOW(); RETURN NEW; END; $$`,

		// ── users ────────────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS users (
			id                 SERIAL       PRIMARY KEY,
			name               VARCHAR(100) NOT NULL DEFAULT '',
			phone              VARCHAR(50)  NOT NULL DEFAULT '',
			email              VARCHAR(255) NOT NULL UNIQUE,
			password           VARCHAR(512) NOT NULL,
			role               user_role    NOT NULL DEFAULT 'freelancer',
			is_verified        BOOLEAN      NOT NULL DEFAULT FALSE,
			verification_score INT          NOT NULL DEFAULT 0
			                       CHECK (verification_score BETWEEN 0 AND 100),
			created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
			updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW()
		)`,

		// migrate existing tables that predate name/phone columns
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS name  VARCHAR(100) NOT NULL DEFAULT ''`,
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(50)  NOT NULL DEFAULT ''`,

		`CREATE INDEX IF NOT EXISTS idx_users_email ON users (LOWER(email))`,

		`DO $$ BEGIN
			CREATE TRIGGER trg_users_updated_at
				BEFORE UPDATE ON users FOR EACH ROW
				EXECUTE FUNCTION set_updated_at();
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

		// ── jobs ─────────────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS jobs (
			id           SERIAL           PRIMARY KEY,
			title        VARCHAR(255)     NOT NULL CHECK (TRIM(title) <> ''),
			description  TEXT             NOT NULL CHECK (TRIM(description) <> ''),
			budget       NUMERIC(12,2)    CHECK (budget IS NULL OR budget > 0),
			budget_type  budget_type      NOT NULL DEFAULT 'fixed',
			category     VARCHAR(100),
			job_type     job_type         NOT NULL DEFAULT 'remote',
			experience   experience_level NOT NULL DEFAULT 'any',
			duration     project_duration,
			location     VARCHAR(255),
			skills       TEXT,
			requirements TEXT,
			deadline     DATE,
			employer_id  INT              NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			is_open      BOOLEAN          NOT NULL DEFAULT TRUE,
			created_at   TIMESTAMPTZ      NOT NULL DEFAULT NOW(),
			updated_at   TIMESTAMPTZ      NOT NULL DEFAULT NOW()
		)`,

		// migrate existing jobs tables that predate the extended columns
		`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS budget_type  budget_type      NOT NULL DEFAULT 'fixed'`,
		`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS category     VARCHAR(100)`,
		`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS job_type     job_type         NOT NULL DEFAULT 'remote'`,
		`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS experience   experience_level NOT NULL DEFAULT 'any'`,
		`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS duration     project_duration`,
		`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS location     VARCHAR(255)`,
		`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS skills       TEXT`,
		`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS requirements TEXT`,
		`ALTER TABLE jobs ADD COLUMN IF NOT EXISTS deadline     DATE`,

		`CREATE INDEX IF NOT EXISTS idx_jobs_employer_id ON jobs (employer_id)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_created_at  ON jobs (created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_is_open     ON jobs (is_open) WHERE is_open = TRUE`,
		`CREATE INDEX IF NOT EXISTS idx_jobs_category    ON jobs (category)`,

		`DO $$ BEGIN
			CREATE TRIGGER trg_jobs_updated_at
				BEFORE UPDATE ON jobs FOR EACH ROW
				EXECUTE FUNCTION set_updated_at();
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

		// ── applications ─────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS applications (
			id            SERIAL             PRIMARY KEY,
			job_id        INT                NOT NULL REFERENCES jobs(id)  ON DELETE CASCADE,
			freelancer_id INT                NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			status        application_status NOT NULL DEFAULT 'pending',
			cover_note    TEXT,
			created_at    TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
			updated_at    TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
			CONSTRAINT uq_application UNIQUE (job_id, freelancer_id)
		)`,

		`ALTER TABLE applications ADD COLUMN IF NOT EXISTS job_test_score INT`,

		`CREATE INDEX IF NOT EXISTS idx_applications_job_id        ON applications (job_id)`,
		`CREATE INDEX IF NOT EXISTS idx_applications_freelancer_id ON applications (freelancer_id)`,

		`DO $$ BEGIN
			CREATE TRIGGER trg_applications_updated_at
				BEFORE UPDATE ON applications FOR EACH ROW
				EXECUTE FUNCTION set_updated_at();
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,

		// ── test_results (audit log) ──────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS test_results (
			id          SERIAL      PRIMARY KEY,
			user_id     INT         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			score       INT         NOT NULL CHECK (score BETWEEN 0 AND 100),
			is_verified BOOLEAN     NOT NULL,
			taken_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		`CREATE INDEX IF NOT EXISTS idx_test_results_user_id ON test_results (user_id)`,

		// ── job_questions ─────────────────────────────────────────────────
		`CREATE TABLE IF NOT EXISTS job_questions (
			id          SERIAL      PRIMARY KEY,
			job_id      INT         NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
			question    TEXT        NOT NULL CHECK (TRIM(question) <> ''),
			options     JSONB       NOT NULL DEFAULT '[]',
			correct_idx INT         NOT NULL DEFAULT 0,
			created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,

		`CREATE INDEX IF NOT EXISTS idx_job_questions_job_id ON job_questions (job_id)`,

		`DO $$ BEGIN
			CREATE TRIGGER trg_job_questions_updated_at
				BEFORE UPDATE ON job_questions FOR EACH ROW
				EXECUTE FUNCTION set_updated_at();
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	}

	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			log.Fatalf("Migration error:\n%s\n\nError: %v", q, err)
		}
	}
	log.Println("Database tables ready")
}
