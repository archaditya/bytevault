# ByteVault — Project Progress & Context

> **Last Updated:** 2026-04-29
> **Current Milestone:** 0.5 — User & Auth System (Part 1: DB + Migrations)
> **Language Learning:** Go (from scratch)

---

## 🛠️ Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go v1.25.7 |
| HTTP Framework | Echo v4 |
| Database | PostgreSQL |
| DB Driver | pgx/pgxpool (no ORM) |
| Logging | Zerolog |
| Config | Koanf |
| Migrations | Tern |
| Module Path | `github.com/archaditya/bytevault` |

---

## 📁 Current Project Structure

```
ByteVault/
├── cmd/api/main.go
├── internal/
│   ├── config/config.go
│   ├── logger/logger.go
│   ├── handler/health.go
│   ├── server/server.go
│   ├── server/routes.go
│   └── database/              ← NEW in Part 1
│       └── postgres.go
├── migrations/                 ← NEW in Part 1
│   ├── tern.conf
│   ├── 001_create_users_table.sql
│   ├── 002_create_auth_providers_table.sql
│   └── 003_create_sessions_table.sql
├── .env
├── .env.example
├── .gitignore
├── MakeFile
├── go.mod
└── go.sum
```

---

## 📍 Milestone Tracker

| # | Milestone | Status | Notes |
|---|-----------|--------|-------|
| 0 | Setup | ✅ Complete | Server + health endpoint |
| 0.5 | User & Auth | 🟡 In Progress | Part 1: DB + Migrations |
| 1 | Core File System | 🔴 Not Started | |
| 2 | Transfer Engine | 🔴 Not Started | |
| 3 | Queue System | 🔴 Not Started | |
| 4 | System Optimization | 🔴 Not Started | |
| 5 | AI Service | 🔴 Not Started | |
| 6 | Infrastructure | 🔴 Not Started | |
| 7 | Business Layer | 🔴 Not Started | |

---

## ✅ Completed Work

### Milestone 0 (2026-04-28)
- Go module initialized
- Config with Koanf (.env + env vars)
- Zerolog logger (dev: console, prod: JSON)
- Echo server with middleware (Recover, CORS, RequestID, RequestLogger)
- Health endpoint: GET /api/v1/health
- Makefile

---

## 📝 Go Concepts Learned

| Concept | Milestone |
|---------|-----------|
| package, import, func main() | M0 |
| struct, struct tags | M0 |
| method receivers | M0 |
| pointers (* and &) | M0 |
| error handling (if err != nil) | M0 |
| dependency injection | M0 |
| internal/ packages | M0 |
| context.Context | M0.5 Part 1 |
| defer | M0.5 Part 1 |
| connection pools | M0.5 Part 1 |

---

## 🗄️ Database Schema

### users
| Column | Type | Notes |
|--------|------|-------|
| id | UUID (PK) | gen_random_uuid() |
| email | VARCHAR(255) | UNIQUE |
| password_hash | TEXT | nullable (OAuth users) |
| full_name | VARCHAR(255) | |
| avatar_url | TEXT | |
| is_active | BOOLEAN | default true |
| is_verified | BOOLEAN | default false |
| created_at | TIMESTAMPTZ | |
| updated_at | TIMESTAMPTZ | |

### auth_providers
| Column | Type | Notes |
|--------|------|-------|
| id | UUID (PK) | |
| user_id | UUID (FK→users) | CASCADE delete |
| provider | VARCHAR(50) | email/google/apple |
| provider_user_id | VARCHAR(255) | OAuth ID |
| created_at | TIMESTAMPTZ | |
| UNIQUE | (user_id, provider) | |

### sessions
| Column | Type | Notes |
|--------|------|-------|
| id | UUID (PK) | |
| user_id | UUID (FK→users) | CASCADE delete |
| refresh_token_hash | TEXT | SHA-256 hash |
| user_agent | TEXT | |
| ip_address | VARCHAR(45) | |
| expires_at | TIMESTAMPTZ | |
| created_at | TIMESTAMPTZ | |
| last_used_at | TIMESTAMPTZ | |

---

## 🔗 Key Files

| Purpose | Path |
|---------|------|
| Entry point | cmd/api/main.go |
| Config | internal/config/config.go |
| Logger | internal/logger/logger.go |
| Server | internal/server/server.go |
| Routes | internal/server/routes.go |
| Health handler | internal/handler/health.go |
| DB connection | internal/database/postgres.go |
| Migrations | migrations/*.sql |
