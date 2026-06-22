# ByteVault — Project Progress & Context

> **Last Updated:** 2026-06-23
> **Current Milestone:** 1 — Core File System
> **Language Learning:** Go (from scratch)
> **GitHub:** github.com/archaditya/bytevault

---

## 🛠️ Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go v1.25.7 |
| HTTP Framework | Echo v4 |
| Database | PostgreSQL 18 |
| DB Driver | pgx/pgxpool (no ORM) |
| Logging | Zerolog |
| Config | Koanf (.env + env vars) |
| Migrations | Embedded SQL via Go `embed` package (auto-run on startup) |
| Auth | JWT (golang-jwt/v5) + bcrypt |
| Frontend | Next.js 16 + Tailwind CSS v4 |
| Module Path | `github.com/archaditya/bytevault` |

---

## 📁 Current Project Structure

```
ByteVault/
├── cmd/api/
│   ├── main.go                         ← Entry point + embed migrations
│   └── migrations/                     ← SQL files (embedded into binary)
│       ├── tern.conf
│       ├── 001_create_users_table.sql
│       ├── 002_create_auth_providers_table.sql
│       ├── 003_create_sessions_table.sql
│       ├── 004_add_rbac_devices_activity.sql
│       └── 005_add_rbac_devices_activity.sql
├── internal/
│   ├── config/config.go                ← Koanf config (Server, DB, App, JWT)
│   ├── logger/logger.go                ← Zerolog (console dev, JSON prod)
│   ├── database/postgres.go            ← pgxpool + RunMigrations()
│   ├── model/
│   │   ├── user.go                     ← User struct (with RoleName + Permissions)
│   │   ├── role.go                     ← Role struct (JSONB permissions)
│   │   ├── user_device.go             ← UserDevice struct (FCM tokens)
│   │   └── activity_log.go            ← ActivityLog struct
│   ├── repository/
│   │   ├── user_repository.go          ← CRUD + ListAll + GetStats
│   │   ├── session_repository.go       ← CRUD for sessions
│   │   ├── role_repository.go          ← Role CRUD + AssignRoleToUser
│   │   ├── device_repository.go        ← FCM device token CRUD
│   │   └── activity_repository.go      ← Activity log queries
│   ├── service/
│   │   └── auth_service.go             ← Register, Login, JWT (with role+perms), bcrypt
│   ├── handler/
│   │   ├── health.go                   ← GET /health
│   │   └── auth_handler.go             ← Register, Login, Refresh, Logout
│   ├── middleware/
│   │   ├── auth.go                     ← JWT auth middleware (extracts role+perms)
│   │   └── permission.go               ← RequirePermission middleware
│   └── server/
│       ├── server.go                   ← Echo setup + middleware
│       ├── routes.go                   ← Central route wiring
│       ├── health_routes.go            ← /health
│       ├── auth_routes.go              ← /auth/*
│       ├── user_routes.go              ← /me, /me/devices, /me/sessions
│       └── admin_routes.go             ← /admin/* (permission-gated)
├── web/                                ← Next.js 16 + Tailwind v4 Frontend
│   ├── src/
│   │   ├── app/
│   │   │   ├── layout.js              ← Root layout + AuthProvider
│   │   │   ├── page.js                ← Landing page (cyber-vault aesthetic)
│   │   │   ├── login/page.js          ← Login (glassmorphism)
│   │   │   ├── register/page.js       ← Register
│   │   │   ├── dashboard/
│   │   │   │   ├── layout.js          ← User sidebar layout
│   │   │   │   ├── page.js            ← User dashboard (stats + quick actions)
│   │   │   │   ├── profile/page.js    ← User profile
│   │   │   │   ├── devices/page.js    ← FCM devices list
│   │   │   │   └── sessions/page.js   ← Active sessions
│   │   │   └── admin/
│   │   │       ├── layout.js          ← Admin sidebar (red accent)
│   │   │       ├── page.js            ← Admin dashboard (live stats)
│   │   │       ├── users/page.js      ← User management table
│   │   │       ├── roles/page.js      ← Role + permission management
│   │   │       ├── activity/page.js   ← Activity logs
│   │   │       └── sessions/page.js   ← Session monitor
│   │   ├── components/auth/Guards.js  ← AuthGuard + RoleGuard
│   │   ├── contexts/AuthContext.js    ← Auth state + token management
│   │   └── lib/api/
│   │       ├── http.js                ← Fetch wrapper + token refresh
│   │       ├── auth.js                ← Auth API calls
│   │       ├── admin.js               ← Admin API calls
│   │       └── index.js               ← Barrel export
│   └── .env.local
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
| 0 | Setup | ✅ Complete | Server + health endpoint + Zerolog + Koanf |
| 0.5 | User & Auth + RBAC | ✅ Complete | DB + Auth + JWT + RBAC + Frontend |
| 1 | Core File System | 🟡 Starting | File upload/download, Cloudinary |
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

### Milestone 0.5 (2026-04-29 → 2026-06-23)
- PostgreSQL connection with pgxpool (connection pooling)
- Auto-migrations on startup using Go `embed` package
- Migration files (users, auth_providers, sessions, rbac+devices+activity)
- User model with JSON tags, nullable fields, and role info
- User repository (Create, FindByEmail, FindByID, SoftDelete, ListAll, GetStats)
- Session repository (Create, FindByTokenHash, Delete)
- Role repository (FindByName, FindByID, ListAll, AssignRoleToUser, GetUserRole)
- Device repository (Upsert, FindByUserID, Deactivate)
- Activity repository (Log, ListAll, ListByUserID)
- Auth service (Register, Login, Refresh, Logout, ValidateAccessToken)
- JWT access tokens (15min) + refresh tokens (7 days, SHA-256 hashed)
- JWT now includes role + permissions (JSONB from roles table)
- bcrypt password hashing (cost factor 14)
- Auth middleware (Bearer token validation + role/permissions extraction)
- Permission middleware (RequirePermission for granular access control)
- Route architecture: health_routes, auth_routes, user_routes, admin_routes
- **RBAC system:** Separate roles table with JSONB permissions
- **Default roles:** user + super_admin (seeded via migration)
- **Activity logging:** register, login events tracked with IP + user-agent
- **FCM device tracking:** user_devices table for push notifications
- **Frontend (Next.js 16 + Tailwind v4):**
  - Landing page (dark cyber-vault aesthetic, terminal preview, features)
  - Login/Register pages (glassmorphism, glow inputs, gradient buttons)
  - User dashboard (stats, quick actions, system status)
  - User profile, devices, sessions pages
  - Admin dashboard (live stats from API)
  - Admin user management (paginated table)
  - Admin role management (permission badges)
  - Admin activity logs (color-coded action badges)
  - Admin session monitor
  - AuthGuard + RoleGuard components
  - API client with auto token refresh (same pattern as spire-engage)

---

## 🔗 API Endpoints

### Public
| Method | Path | Purpose |
|--------|------|---------|
| GET | /api/v1/health | Health check |
| POST | /api/v1/auth/register | Register (auto-assigns "user" role) |
| POST | /api/v1/auth/login | Login (returns JWT with role+perms) |
| POST | /api/v1/auth/refresh | Refresh tokens |
| POST | /api/v1/auth/logout | Logout |

### Protected (JWT required)
| Method | Path | Purpose |
|--------|------|---------|
| GET | /api/v1/me | Get current user + role |
| POST | /api/v1/me/devices | Register FCM device |
| GET | /api/v1/me/devices | List user's devices |
| DELETE | /api/v1/me/devices/:id | Remove device |

### Admin (JWT + permission required)
| Method | Path | Permission | Purpose |
|--------|------|-----------|---------|
| GET | /api/v1/admin/stats | admin:users | System stats |
| GET | /api/v1/admin/users | admin:users | List all users (paginated) |
| GET | /api/v1/admin/roles | admin:roles | List all roles |
| GET | /api/v1/admin/activity | admin:activity | Activity logs (paginated) |

---

## 🗄️ Database Schema

### users
| Column | Type | Notes |
|--------|------|-------|
| id | UUID (PK) | gen_random_uuid() |
| email | VARCHAR(255) | NOT NULL (not UNIQUE — soft delete support) |
| password | TEXT | nullable (OAuth users) |
| first_name, last_name | VARCHAR(255) | |
| avatar_url | TEXT | |
| is_verified | BOOLEAN | default false |
| status | VARCHAR(255) | |
| created_by, updated_by, deleted_by | UUID | |
| created_at, updated_at | TIMESTAMPTZ | |
| deleted_at | TIMESTAMPTZ | nullable (soft delete) |

### roles
| Column | Type | Notes |
|--------|------|-------|
| id | UUID (PK) | |
| name | VARCHAR(50) | UNIQUE |
| description | TEXT | |
| permissions | JSONB | {"user:read": true, "admin:users": false, ...} |
| is_system_role | BOOLEAN | |

### user_roles
| Column | Type | Notes |
|--------|------|-------|
| user_id | UUID (FK→users) | CASCADE |
| role_id | UUID (FK→roles) | CASCADE |
| UNIQUE | (user_id, role_id) | |

### user_devices
| Column | Type | Notes |
|--------|------|-------|
| id | UUID (PK) | |
| user_id | UUID (FK→users) | CASCADE |
| fcm_token | TEXT | UNIQUE |
| device_type | VARCHAR(20) | web/android/ios |
| device_id | VARCHAR(255) | |
| is_active | BOOLEAN | |

### activity_logs
| Column | Type | Notes |
|--------|------|-------|
| id | UUID (PK) | |
| user_id | UUID (FK→users) | SET NULL on delete |
| action | VARCHAR(100) | e.g., "user.register" |
| resource_type | VARCHAR(50) | |
| metadata | JSONB | Extra data |
| ip_address, user_agent | | |

### sessions, auth_providers
Same as before (see earlier versions).

---

## 📝 Go Concepts Learned

| Concept | Milestone | Explanation |
|---------|-----------|-------------|
| package, import, func main() | M0 | Basic Go program structure |
| struct, struct tags | M0 | Custom types + JSON/Koanf annotations |
| method receivers | M0 | Functions that belong to a struct |
| pointers (* and &) | M0 | References vs copies |
| error handling (if err != nil) | M0 | Go's explicit error pattern |
| dependency injection | M0 | Pass deps in, don't use globals |
| internal/ packages | M0 | Private packages within module |
| context.Context | M0.5 | Timer + cancel button for requests |
| defer | M0.5 | "Run this when function exits" |
| connection pools | M0.5 | Reuse DB connections |
| //go:embed | M0.5 | Bake files into binary at compile time |
| fs.FS | M0.5 | Filesystem interface (real or embedded) |
| *string (nullable) | M0.5 | Pointer types for NULL DB values |
| json:"-" | M0.5 | Exclude field from JSON |
| $1, $2 (parameterized SQL) | M0.5 | Prevent SQL injection |
| QueryRow().Scan() | M0.5 | Read one DB row into variables |
| rows.Next() + defer rows.Close() | M0.5 | Iterate multiple DB rows |
| errors.Is() | M0.5 | Check specific error type |
| bcrypt | M0.5 | Password hashing |
| JWT (signed tokens) | M0.5 | Stateless auth tokens |
| crypto/rand + SHA-256 | M0.5 | Secure random token generation |
| middleware pattern | M0.5 | Intercept requests before handlers |
| map[string]bool | M0.5 | Go maps for permission checks |
| json.Marshal/Unmarshal | M0.5 | Convert between JSON bytes ↔ Go maps |
| JSONB in PostgreSQL | M0.5 | Binary JSON for flexible schemas |

---

## 🏗️ Architecture Pattern

```
Request → Echo Middleware → Auth MW → Permission MW → Handler → Service → Repository → PostgreSQL
                                                                    ↓
                                                                JWT / bcrypt
                                                                    ↓
                                                              Activity Logger
```

---

## 🔗 Key File References

| Purpose | Path |
|---------|------|
| Entry point | cmd/api/main.go |
| Config | internal/config/config.go |
| Logger | internal/logger/logger.go |
| DB + Migrations | internal/database/postgres.go |
| User model | internal/model/user.go |
| Role model | internal/model/role.go |
| Device model | internal/model/user_device.go |
| Activity model | internal/model/activity_log.go |
| User queries | internal/repository/user_repository.go |
| Session queries | internal/repository/session_repository.go |
| Role queries | internal/repository/role_repository.go |
| Device queries | internal/repository/device_repository.go |
| Activity queries | internal/repository/activity_repository.go |
| Auth logic | internal/service/auth_service.go |
| Auth endpoints | internal/handler/auth_handler.go |
| Health endpoint | internal/handler/health.go |
| JWT middleware | internal/middleware/auth.go |
| Permission middleware | internal/middleware/permission.go |
| Route wiring | internal/server/routes.go |
| Admin routes | internal/server/admin_routes.go |
| User routes | internal/server/user_routes.go |
| Server setup | internal/server/server.go |
| Frontend root | web/src/app/layout.js |
| Landing page | web/src/app/page.js |
| Auth context | web/src/contexts/AuthContext.js |
| API client | web/src/lib/api/http.js |

---

## ⚠️ Known Issues / TODO

- [ ] Email OTP login (not implemented yet)
- [ ] OAuth Google/Apple (not implemented yet)
- [ ] Validator package not yet integrated for request validation
- [ ] `auth_providers` table exists but not used in auth flow yet
- [ ] Firebase Admin SDK not yet integrated for actual push notifications
- [x] GitHub username changed from `adityakkpk` → `archaditya` (all imports fixed)
