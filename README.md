# SkillVerify — Smart Skills Verification & Freelance Platform

A full-stack web application built with Go (stdlib only), HTML/CSS/JS frontend, and PostgreSQL.

## Features

- **User Registration & Login** — JWT-based auth (72-hour expiry), roles: Freelancer / Employer / Admin
- **Skill Verification** — 10-question MCQ test; ≥60% earns a Verified badge; full attempt history logged
- **Job Marketplace** — Employers post jobs (with optional budget); Freelancers browse with pagination
- **Application Tracking** — Status pipeline: `pending → shortlisted → rejected / hired`
- **Employer Dashboard** — View all applicants per job with verification badge visibility
- **Zero External Go Dependencies** — only `github.com/lib/pq` (PostgreSQL driver); vendored

---

## Quick Start (Docker — recommended)

```bash
docker-compose up --build
```

Open http://localhost:8080 in your browser.

---

## Manual Setup

### 1. Prerequisites

- Go 1.21+
- PostgreSQL 13+

### 2. Create the database

```sql
CREATE DATABASE freelance_platform;
```

### 3. Run the SQL schema (recommended)

The Go app auto-creates tables on startup, but running `schema.sql` first
adds indexes, views, and comments that the auto-migration does not:

```bash
psql -U postgres -d freelance_platform -f schema.sql
```

### 4. Configure environment

```bash
cp .env.example .env
# Edit .env with your PostgreSQL credentials and a strong JWT_SECRET
```

### 5. Build and run

```bash
go build -mod=vendor -o server .
./server
```

Or with `go run`:

```bash
go run main.go
```

App available at http://localhost:8080.

---

## API Endpoints

| Method | Endpoint                        | Auth           | Description                                 |
|--------|---------------------------------|----------------|---------------------------------------------|
| POST   | `/register`                     | —              | Register (email, password, role)            |
| POST   | `/login`                        | —              | Login → JWT token + user object             |
| GET    | `/jobs?page=1&limit=20`         | —              | List open jobs (paginated)                  |
| POST   | `/jobs`                         | Employer/Admin | Post a new job (title, description, budget) |
| POST   | `/apply`                        | Freelancer     | Apply for a job (job_id, cover_note)        |
| GET    | `/my-applications`              | Freelancer     | My submitted applications                   |
| GET    | `/job-applications?job_id=N`    | Employer/Admin | Applicants for a specific job               |
| PUT    | `/application-status?id=N`      | Employer/Admin | Update application status                   |
| GET    | `/test-questions`               | Any user       | Get 10 skill assessment questions           |
| POST   | `/take-test`                    | Freelancer     | Submit answers → score + verification       |

All protected routes require `Authorization: Bearer <token>` header.

---

## What Was Fixed

### Security
- **JWT expiry** — tokens now expire after 72 hours (`exp` + `iat` claims)
- **Constant-time password comparison** — uses `crypto/subtle` to prevent timing attacks
- **Safe type assertions** — middleware panics on malformed tokens are gone
- **Stronger password policy** — minimum raised from 6 to 8 characters
- **Case-insensitive email** — normalised to lowercase on register & login

### Database / SQL
- **ENUM types** — `user_role` and `application_status` replace free-text columns
- **`updated_at` trigger** — auto-maintained on `users`, `jobs`, `applications`
- **`budget` column** — nullable `NUMERIC(12,2)` on `jobs`
- **`cover_note` column** — optional text on `applications`
- **`is_open` column** — jobs can be closed after hiring
- **`test_results` table** — full audit history of every test attempt
- **Indexes** — on `email`, `employer_id`, `freelancer_id`, `created_at`, `status`
- **CHECK constraints** — verify `verification_score ∈ [0,100]`, `budget > 0`
- **Views** — `v_jobs_with_stats`, `v_freelancer_profiles` for analytics

### API
- **Pagination** — `GET /jobs` accepts `?page=N&limit=N`; returns `total` count
- **New endpoints** — `/my-applications`, `/job-applications`, `/application-status`
- **Job validation** — checks job exists and `is_open = TRUE` before accepting application
- **Employer ownership check** — employers can only see / update their own jobs' applications
- **PostgreSQL error codes** — uses error code `23505` for reliable duplicate detection

### Architecture
- **Custom `ServeMux`** — replaces `http.DefaultServeMux` for testability
- **Graceful shutdown** — SIGINT/SIGTERM handled with 10-second drain timeout
- **Connection pool tuning** — `MaxOpenConns`, `MaxIdleConns`, `ConnMaxLifetime`
- **Built-in `.env` loader** — no external library needed
- **`DB_SSLMODE` env var** — easy switch to `require` for hosted PostgreSQL (Render)

---

## Project Structure

```
.
├── main.go                    # Entry point — server init + graceful shutdown
├── schema.sql                 # Full SQL schema (indexes, views, seed data)
├── go.mod
├── vendor/                    # Vendored lib/pq
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── backend/
│   ├── database/database.go   # DB connection, pool config, auto-migration
│   ├── models/models.go       # Structs + DTOs (Budget, CoverNote, IsOpen …)
│   ├── middleware/middleware.go# JWT with expiry; safe context helpers
│   ├── handlers/
│   │   ├── auth.go            # Register (lowercase email), Login
│   │   ├── jobs.go            # GetJobs (paginated), CreateJob, Apply,
│   │   │                      #   GetMyApplications, GetJobApplications,
│   │   │                      #   UpdateApplicationStatus
│   │   └── test.go            # GetTestQuestions, TakeTest (audit log)
│   ├── routes/routes.go       # Custom ServeMux, CORS, auth wiring
│   └── utils/
│       ├── utils.go           # Email/password validation, HashPassword,
│       │                      #   CheckPassword (constant-time)
│       └── response.go        # JSON response helpers
└── frontend/
    ├── index.html
    ├── login.html
    ├── register.html
    ├── dashboard.html
    ├── jobs.html
    ├── test.html
    ├── app.js
    └── styles.css
```

## Security Notes for Production

1. **Set `JWT_SECRET`** to a random 32+ byte value (`openssl rand -hex 32`)
2. **Set `DB_SSLMODE=require`** when using hosted PostgreSQL (Render, Supabase, etc.)
3. **Restrict CORS** — replace `*` in `routes.go` with your frontend's actual origin
4. **Consider bcrypt** — swap `utils.HashPassword` with `golang.org/x/crypto/bcrypt` for stronger hashing
5. **Rate-limit `/login`** and `/register` — add middleware or a reverse-proxy rule
