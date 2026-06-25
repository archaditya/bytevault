# ByteVault — Project Progress Tracker

> Last updated: **2026-06-26**

---

## Overall Progress Snapshot

| Version | Scope | Backend | Frontend | Wired Together |
|---------|-------|---------|----------|----------------|
| V1 — Storage Platform | ✅ Defined | ✅ 100% | ✅ 100% | ✅ 100% |
| V2 — Resumable Uploads | ✅ Defined | ❌ 0% | 🟡 ~30% (UI shell) | ❌ 0% |
| V3 — Real-Time Visibility | ✅ Defined | ❌ 0% | 🟡 ~20% (UI shell) | ❌ 0% |
| V4 — File Processing | ✅ Defined | ❌ 0% | ❌ 0% | ❌ 0% |

---

## Version 1 — Storage Platform

### Backend Checklist

| Layer | Item | Status | Notes |
|-------|------|--------|-------|
| **Config** | `config.go` — env loading (koanf) | ✅ Done | Upgraded key transformer to bind flat struct tags |
| **Database** | PostgreSQL connection pool | ✅ Done | pgxpool with health checks |
| **Database** | Migration system (embedded SQL) | ✅ Done | `schema_migrations` tracking |
| **Migration** | `users` table | ✅ Done | 001 |
| **Migration** | `sessions` table | ✅ Done | 003 |
| **Migration** | `roles`, `user_roles`, `user_devices`, `activity_logs` | ✅ Done | 005 |
| **Migration** | `files` table | ✅ Done | 006 |
| **Model** | `User`, `File`, `Role`, `ActivityLog`, `UserDevice` | ✅ Done | |
| **Repository** | `UserRepository` (CRUD, ListAll, GetStats) | ✅ Done | |
| **Repository** | `SessionRepository` (CRUD, FindByHash) | ✅ Done | |
| **Repository** | `RoleRepository` (FindByName, AssignRole, GetUserRole) | ✅ Done | |
| **Repository** | `ActivityRepository` (Log, ListAll, ListByUserID) | ✅ Done | |
| **Repository** | `DeviceRepository` (Upsert, FindByUserID, Deactivate) | ✅ Done | |
| **Repository** | `FileRepository` (Create, FindByID, List, UpdatePublic, SoftDelete) | ✅ Done | Filters out non-READY files |
| **Service** | `AuthService` (Register, Login, Refresh, Logout, ValidateToken) | ✅ Done | JWT + bcrypt + session rotation |
| **Service** | `FileService` (Upload, Download, DownloadPublic, List, ToggleShare, Delete) | ✅ Done | |
| **Handler** | `AuthHandler` (Register, Login, Refresh, Logout) | ✅ Done | Standardized JSON responses |
| **Handler** | `FileHandler` (Upload, Download, DownloadPublic, List, ToggleShare, Delete) | ✅ Done | Standardized JSON responses |
| **Handler** | `HealthHandler` | ✅ Done | |
| **Middleware** | JWT Auth middleware | ✅ Done | Extracts user_id, role, permissions |
| **Middleware** | Permission-based access control | ✅ Done | |
| **Routes** | Health, Auth, User, Admin, File routes | ✅ Done | |
| **Storage** | `StorageProvider` interface | ✅ Done | Upload, Download, Delete, Presigned URLs |
| **Storage** | Local storage implementation | ✅ Done | |
| **Storage** | Cloudinary storage implementation | ✅ Done | |
| **Storage** | Cloudflare R2 storage implementation | ✅ Done | AWS SDK v2, presigned URLs |
| **Storage** | Pluggable provider selection | ✅ Done | Config-driven switch |
| **Schema** | `files` table has `status` column | ✅ Done | Added in 007 migration |
| **API** | Presigned upload URL endpoint | ✅ Done | `POST /api/v1/files/upload-session` |
| **API** | Upload completion callback | ✅ Done | `POST /api/v1/files/:id/complete` |
| **Flow** | Direct client→R2 upload flow | ✅ Done | Presigned PUT direct-to-cloud upload completed |

### Frontend Checklist

| Layer | Item | Status | Notes |
|-------|------|--------|-------|
| **Framework** | Next.js 15 + TypeScript | ✅ Done | App Router |
| **Styling** | Next + custom Tailwind design system | ✅ Done | Dark theme, custom tokens |
| **UI Library** | Radix UI primitives | ✅ Done | Dialog, Dropdown, Tabs, etc. |
| **Layout** | AppShell (Sidebar + Navbar) | ✅ Done | |
| **State** | Zustand stores | ✅ Done | |
| **Data Layer** | React Query hooks | ✅ Done | Swapped mocks for direct hooks |
| **Landing Page** | Landing | ✅ Done | |
| **Dashboard** | Stats, Charts, Recent Transfers, Usage Widget | ✅ Done | Mock data only |
| **Files** | Explorer (grid/list), Toolbar, Cards, Detail page | ✅ Done | Fully integrated with backend APIs |
| **Transfers** | List, Cards, Speed Graph, Timeline, Detail page | ✅ Done | Mock data only |
| **Storage** | Provider cards, Usage widget | ✅ Done | Mock data only |
| **Shared Links** | Share link cards | ✅ Done | Mock data only |
| **Analytics** | Charts, Provider comparison | ✅ Done | Mock data only |
| **Settings** | Profile, Security, Notifications, API Keys, Storage Prefs | ✅ Done | Mock data only |
| **Profile** | Profile page | ✅ Done | Mock data only |
| **Types** | `FileRecord` structure | ✅ Done | Aligned with status codes |
| **Auth** | Login/Register pages | ✅ Done | Integrated with backend services |
| **Auth** | Token storage (localStorage/cookies) | ✅ Done | Managed in `api-client.ts` |
| **Auth** | Auth guards / route protection | ✅ Done | Protected via `RouteGuard` |
| **API Client** | HTTP client with auth interceptor | ✅ Done | Unwraps JSON `.data` envelope automatically |
| **Services** | Real API calls (replacing mock) | ✅ Done | `web/services/files.service.ts` fully wired |
| **File Upload** | Actual upload UI with progress | ✅ Done | Direct cloud upload flow wired up |

### Integration Checklist (Frontend ↔ Backend)

| Item | Status | Notes |
|------|--------|-------|
| API base URL / proxy config | ✅ Done | Configured in `next.config.js` |
| Auth flow (register → store tokens → refresh) | ✅ Done | Fully wired |
| File listing from API | ✅ Done | Fully wired |
| File upload to API (or presigned URL) | ✅ Done | Direct PUT presigned uploads |
| File download from API | ✅ Done | Fully wired |
| File sharing toggle | ✅ Done | Fully wired |
| File deletion | ✅ Done | Fully wired |
| Dashboard stats from API | ❌ Not wired | |
| CORS configuration | ✅ Done | Cloudflare R2 bucket CORS configured for localhost:3000 |

---

## Architectural Rule Compliance

| Rule | Status | Detail |
|------|--------|--------|
| **Rule 1**: Files upload directly to R2 | ✅ Compliant | Raw bytes bypass backend CPU/RAM and upload directly to R2 |
| **Rule 2**: Backend stores metadata, never raw files | ✅ Compliant | Backend manages database records and metadata only |
| **Rule 3**: Temp state in Redis | N/A | V2 scope |
| **Rule 4**: Permanent state in PostgreSQL | ✅ Compliant | |
| **Rule 5**: No heavy processing in handlers | ✅ Compliant | |
| **Rule 6**: WebSockets for visibility only | N/A | V3 scope |
| **Rule 7**: File lifecycle statuses | ✅ Compliant | `status` track added (UPLOADING -> READY) |
| **Rule 8**: Horizontally scalable | ✅ Compliant | |
