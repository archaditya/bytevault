# PushPort Changelog

All notable changes to PushPort will be documented here.

The format is inspired by Keep a Changelog.

---

# Unreleased

### Added

- Upcoming features under active development.

### Changed

-

### Fixed

-

---

# v0.3.0 (Current Development)

## Added

### Authentication

- Google OAuth Login
- JWT Authentication
- Refresh Token Support
- Session Management

### Notifications

- Redis-based Notification Queue
- In-App Notification System
- Real-time Notification APIs

### Infrastructure

- Redis Integration
- Docker Health Checks
- Production CI/CD Pipeline
- Automated GitHub Actions Deployment

### Security

- Reverse Proxy through Nginx
- Security Headers
- Rate Limiting
- Internal-only Backend Port
- PostgreSQL isolated from public internet

### Storage

- Cloudflare R2 Integration
- Presigned Upload URLs
- Presigned Download URLs
- Storage Provider Abstraction

### UI

- Responsive Dashboard
- Mobile Navigation Improvements
- Google Login UI
- Notification UI

## Changed

- Backend deployment pipeline completely automated.
- Production Docker configuration improved.
- Backend now exposed only through Nginx.

## Fixed

- PostgreSQL exposure issue.
- Docker networking issues.
- Production deployment reliability.
- Reverse proxy configuration.
- Health check improvements.

---

# v0.2.0

## Added

- Folder hierarchy
- File upload APIs
- File delete APIs
- Rename APIs
- Local storage provider
- Cloudinary provider
- Cloudflare R2 provider
- Embedded SQL migrations

## Changed

- Refactored service architecture.
- Introduced storage abstraction layer.

## Fixed

- Database migration issues.
- Authentication middleware bugs.

---

# v0.1.0

## Initial Release

### Added

- User Registration
- User Login
- JWT Authentication
- PostgreSQL Integration
- Docker Support
- Echo Framework
- Basic Upload APIs
- Initial Folder APIs
