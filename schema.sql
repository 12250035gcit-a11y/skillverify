-- =============================================================
--  SkillVerify Freelance Platform — PostgreSQL Schema
--  Run this against a fresh database to create all objects.
--  Safe to re-run on an existing database (idempotent).
-- =============================================================

-- ----------------------------------------------------------------
-- 0. Database (create externally if needed, then connect)
-- ----------------------------------------------------------------
-- CREATE DATABASE postgres;
-- \c postgres

-- ----------------------------------------------------------------
-- 1. ENUM types
-- ----------------------------------------------------------------
DO $$ BEGIN
    CREATE TYPE user_role AS ENUM ('freelancer', 'employer', 'admin');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE application_status AS ENUM ('pending', 'shortlisted', 'rejected', 'hired');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE job_type AS ENUM ('remote', 'hybrid', 'onsite');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE budget_type AS ENUM ('fixed', 'hourly', 'negotiable');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE experience_level AS ENUM ('entry', 'mid', 'senior', 'any');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE project_duration AS ENUM (
        'less_week', '1_4_weeks', '1_3_months', '3_6_months', 'ongoing'
    );
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- ----------------------------------------------------------------
-- 2. updated_at trigger function (shared by all tables)
-- ----------------------------------------------------------------
CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

-- ----------------------------------------------------------------
-- 3. users
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS users (
    id                 SERIAL          PRIMARY KEY,
    name               VARCHAR(100)    NOT NULL DEFAULT '',
    phone              VARCHAR(50)     NOT NULL DEFAULT '',
    email              VARCHAR(255)    NOT NULL UNIQUE,
    password           VARCHAR(512)    NOT NULL,
    role               user_role       NOT NULL DEFAULT 'freelancer',
    is_verified        BOOLEAN         NOT NULL DEFAULT FALSE,
    verification_score INT             NOT NULL DEFAULT 0
                           CHECK (verification_score BETWEEN 0 AND 100),
    created_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- Migrations for databases that predate name / phone columns
ALTER TABLE users ADD COLUMN IF NOT EXISTS name  VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE users ADD COLUMN IF NOT EXISTS phone VARCHAR(50)  NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_users_email ON users (LOWER(email));
CREATE INDEX IF NOT EXISTS idx_users_role  ON users (role);

COMMENT ON TABLE  users                    IS 'Platform accounts: freelancers, employers, admins';
COMMENT ON COLUMN users.name               IS 'Full display name provided at registration';
COMMENT ON COLUMN users.phone              IS 'Contact phone; required for account security';
COMMENT ON COLUMN users.password           IS 'Salted SHA-256: "<32-hex-salt>:<64-hex-hash>"';
COMMENT ON COLUMN users.verification_score IS '0–100; >=60 marks the user as verified';

DO $$ BEGIN
    CREATE TRIGGER trg_users_updated_at
        BEFORE UPDATE ON users FOR EACH ROW
        EXECUTE FUNCTION set_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- ----------------------------------------------------------------
-- 4. jobs
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS jobs (
    id           SERIAL           PRIMARY KEY,
    title        VARCHAR(255)     NOT NULL CHECK (TRIM(title) <> ''),
    description  TEXT             NOT NULL CHECK (TRIM(description) <> ''),

    budget       NUMERIC(12, 2)   CHECK (budget IS NULL OR budget > 0),
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
);

-- Migrations for databases that predate the extended job columns
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS budget_type  budget_type      NOT NULL DEFAULT 'fixed';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS category     VARCHAR(100);
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS job_type     job_type         NOT NULL DEFAULT 'remote';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS experience   experience_level NOT NULL DEFAULT 'any';
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS duration     project_duration;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS location     VARCHAR(255);
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS skills       TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS requirements TEXT;
ALTER TABLE jobs ADD COLUMN IF NOT EXISTS deadline     DATE;

CREATE INDEX IF NOT EXISTS idx_jobs_employer_id ON jobs (employer_id);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at  ON jobs (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_jobs_is_open     ON jobs (is_open) WHERE is_open = TRUE;
CREATE INDEX IF NOT EXISTS idx_jobs_category    ON jobs (category);
CREATE INDEX IF NOT EXISTS idx_jobs_job_type    ON jobs (job_type);

COMMENT ON TABLE  jobs              IS 'Job postings created by employers';
COMMENT ON COLUMN jobs.budget       IS 'Optional budget in USD; NULL = undisclosed';
COMMENT ON COLUMN jobs.budget_type  IS 'fixed | hourly | negotiable';
COMMENT ON COLUMN jobs.category     IS 'development | design | writing | marketing | data | video | finance | other';
COMMENT ON COLUMN jobs.skills       IS 'Comma-separated required skills, e.g. "React,Node.js,Figma"';
COMMENT ON COLUMN jobs.deadline     IS 'Last date to accept applications; NULL = open-ended';
COMMENT ON COLUMN jobs.is_open      IS 'FALSE once the employer closes the listing or hires someone';

DO $$ BEGIN
    CREATE TRIGGER trg_jobs_updated_at
        BEFORE UPDATE ON jobs FOR EACH ROW
        EXECUTE FUNCTION set_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- ----------------------------------------------------------------
-- 5. applications
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS applications (
    id            SERIAL             PRIMARY KEY,
    job_id        INT                NOT NULL REFERENCES jobs(id)  ON DELETE CASCADE,
    freelancer_id INT                NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status        application_status NOT NULL DEFAULT 'pending',
    cover_note    TEXT,
    job_test_score INT,
    created_at    TIMESTAMPTZ        NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ        NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_application UNIQUE (job_id, freelancer_id)
);

-- Migration for databases that predate the job_test_score column
ALTER TABLE applications ADD COLUMN IF NOT EXISTS job_test_score INT;

CREATE INDEX IF NOT EXISTS idx_applications_job_id        ON applications (job_id);
CREATE INDEX IF NOT EXISTS idx_applications_freelancer_id ON applications (freelancer_id);
CREATE INDEX IF NOT EXISTS idx_applications_status        ON applications (status);

COMMENT ON TABLE  applications                IS 'Freelancer applications to job postings';
COMMENT ON COLUMN applications.cover_note     IS 'Optional cover letter / pitch from the freelancer';
COMMENT ON COLUMN applications.job_test_score IS '0–100 score from the job-specific screening test; NULL if not yet taken';

DO $$ BEGIN
    CREATE TRIGGER trg_applications_updated_at
        BEFORE UPDATE ON applications FOR EACH ROW
        EXECUTE FUNCTION set_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- ----------------------------------------------------------------
-- 6. test_results  (skill-assessment audit log)
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS test_results (
    id          SERIAL      PRIMARY KEY,
    user_id     INT         NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    score       INT         NOT NULL CHECK (score BETWEEN 0 AND 100),
    is_verified BOOLEAN     NOT NULL,
    taken_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_test_results_user_id ON test_results (user_id);

COMMENT ON TABLE test_results IS 'Historical record of every skill-assessment attempt by a freelancer';

-- ----------------------------------------------------------------
-- 7. job_questions  (employer screening questions)
-- ----------------------------------------------------------------
CREATE TABLE IF NOT EXISTS job_questions (
    id          SERIAL      PRIMARY KEY,
    job_id      INT         NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    question    TEXT        NOT NULL CHECK (TRIM(question) <> ''),
    options     JSONB       NOT NULL DEFAULT '[]',
    correct_idx INT         NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_job_questions_job_id ON job_questions (job_id);

COMMENT ON TABLE  job_questions             IS 'Multiple-choice screening questions attached to a job by its employer';
COMMENT ON COLUMN job_questions.options     IS 'JSON array of answer strings, e.g. ["Yes","No","Maybe"]';
COMMENT ON COLUMN job_questions.correct_idx IS 'Zero-based index of the correct answer within options[]';

DO $$ BEGIN
    CREATE TRIGGER trg_job_questions_updated_at
        BEFORE UPDATE ON job_questions FOR EACH ROW
        EXECUTE FUNCTION set_updated_at();
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- ----------------------------------------------------------------
-- 8. Views
-- ----------------------------------------------------------------

-- Jobs with employer info and per-status application counts
CREATE OR REPLACE VIEW v_jobs_with_stats AS
SELECT
    j.id,
    j.title,
    j.description,
    j.budget,
    j.budget_type,
    j.category,
    j.job_type,
    j.experience,
    j.duration,
    j.location,
    j.skills,
    j.requirements,
    j.deadline,
    j.is_open,
    j.created_at,
    u.id    AS employer_id,
    u.name  AS employer_name,
    u.email AS employer_email,
    COUNT(a.id) FILTER (WHERE a.status = 'pending')     AS pending_count,
    COUNT(a.id) FILTER (WHERE a.status = 'shortlisted') AS shortlisted_count,
    COUNT(a.id) FILTER (WHERE a.status = 'hired')       AS hired_count,
    COUNT(a.id)                                          AS total_applications
FROM jobs j
JOIN  users u        ON u.id = j.employer_id
LEFT JOIN applications a ON a.job_id = j.id
GROUP BY j.id, u.id;

-- Freelancer profiles with application history and test scores
CREATE OR REPLACE VIEW v_freelancer_profiles AS
SELECT
    u.id,
    u.name,
    u.phone,
    u.email,
    u.is_verified,
    u.verification_score,
    u.created_at,
    (SELECT COUNT(*) FROM applications a WHERE a.freelancer_id = u.id)                        AS total_applications,
    (SELECT COUNT(*) FROM applications a WHERE a.freelancer_id = u.id AND a.status = 'hired') AS hired_count
FROM users u
WHERE u.role = 'freelancer';

-- ----------------------------------------------------------------
-- 9. Seed — default admin account
--    Password is generated by the Go backend at runtime.
--    To create the admin via the seeder script:
--      cd backend && go run /tmp/seed_admin.go
--    Or promote an existing user:
--      UPDATE users SET role = 'admin' WHERE email = 'admin@gcit.com';
-- ----------------------------------------------------------------
INSERT INTO users (name, phone, email, password, role)
VALUES (
    'Admin',
    '00000000',
    'admin@gcit.com',
    -- Placeholder hash — overwritten by the Go seeder which generates a real salted hash.
    -- Format: "<32-hex-char salt>:<64-hex-char SHA-256 hash>"
    'deadbeefdeadbeefdeadbeefdeadbeef:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa',
    'admin'
)
ON CONFLICT (email) DO NOTHING;

-- ----------------------------------------------------------------
-- 10. Useful queries for development / debugging
-- ----------------------------------------------------------------

-- List all tables in the public schema
-- SELECT tablename FROM pg_tables WHERE schemaname = 'public' ORDER BY tablename;

-- All users (id, name, email, role, verified status)
-- SELECT id, name, email, role, is_verified, verification_score FROM users ORDER BY created_at DESC;

-- Open jobs with employer name and application counts
-- SELECT * FROM v_jobs_with_stats WHERE is_open = TRUE ORDER BY created_at DESC;

-- Verified freelancers
-- SELECT id, name, email, verification_score FROM users WHERE role = 'freelancer' AND is_verified = TRUE;

-- Application pipeline for a specific job (replace <JOB_ID>)
-- SELECT a.id, u.name, u.email, a.status, a.job_test_score, a.created_at
-- FROM applications a JOIN users u ON u.id = a.freelancer_id
-- WHERE a.job_id = <JOB_ID>
-- ORDER BY a.job_test_score DESC NULLS LAST;

-- Shortlisted applicants who have not yet taken the screening test
-- SELECT a.id, u.name, u.email, j.title
-- FROM applications a
-- JOIN users u ON u.id = a.freelancer_id
-- JOIN jobs  j ON j.id = a.job_id
-- WHERE a.status = 'shortlisted' AND a.job_test_score IS NULL;

-- Reset a freelancer's platform verification so they can retake the skill test
-- UPDATE users SET is_verified = FALSE, verification_score = 0 WHERE id = <USER_ID>;

-- Promote any user to admin
-- UPDATE users SET role = 'admin' WHERE email = '<EMAIL>';
