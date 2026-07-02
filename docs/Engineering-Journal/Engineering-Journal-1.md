# Engineering Journal #1

# Building ByteVault Backend from Scratch

**Duration:** June – July 2026

---

## Why I Started ByteVault

While learning backend development, I noticed that most file upload tutorials stop after storing a file locally or uploading it to cloud storage.

I wanted to build something closer to a real production system.

Instead of creating another CRUD project, I decided to build a file transfer platform that focuses on authentication, resumable uploads, cloud storage integration, permissions, and production deployment.

That project became **ByteVault**.

---

# Goal

Build a production-ready backend using Go that could later evolve into a complete SaaS product.

Some initial requirements were:

- User Authentication
- JWT based Authorization
- Folder Management
- File Metadata
- Cloud Storage Support
- Local Storage Support
- Cloudflare R2 Integration
- Role Based Access Control
- Activity Logging
- Docker Deployment
- Production Ready Structure

---

# Tech Stack

Backend

- Go
- Echo Framework

Database

- PostgreSQL

Authentication

- JWT
- Refresh Tokens
- bcrypt Password Hashing

Storage

- Local Storage
- Cloudflare R2
- Cloudinary (Pluggable)

Deployment

- Docker
- Docker Compose
- Nginx
- Ubuntu VPS

---

# Project Structure

Instead of keeping everything inside one folder, I wanted to understand how large backend projects are organized.

The project follows a layered architecture.

```
cmd/
    api/
    seed/

internal/

    config/

    database/

    handler/

    middleware/

    model/

    repository/

    service/

    server/

    storage/
```

Every layer has a single responsibility.

---

# Configuration

The first thing I learned was separating configuration from business logic.

Instead of hardcoding secrets and URLs, everything is loaded through configuration.

Some examples:

- Database URL
- JWT Secret
- Storage Provider
- R2 Credentials
- Server Port

This makes switching environments much easier.

---

# Database

Instead of manually creating tables every time, I learned database migrations.

At application startup:

1. Database connection is created.
2. SQL migrations run automatically.
3. Server starts only if migrations succeed.

I also learned Go's `embed` package.

Instead of shipping SQL files separately, migrations are embedded directly inside the compiled binary.

That was a new concept for me.

---

# Repository Pattern

I separated database queries from business logic.

Repositories only know how to communicate with PostgreSQL.

Examples:

- UserRepository
- SessionRepository
- RoleRepository
- FolderRepository
- FileRepository

This keeps SQL away from services.

---

# Service Layer

The service layer contains business logic.

Some examples:

Authentication

- Register
- Login
- Refresh Token

Files

- Upload Validation
- Storage Key Generation
- Quota Validation
- MIME Type Validation
- Metadata Management

Folders

- Folder Creation
- Nested Folder Support

This separation made the project much easier to understand.

---

# Authentication

One of the biggest learnings was implementing JWT authentication.

Features include:

- Registration
- Login
- Password Hashing
- JWT Access Token
- Refresh Token
- Session Tracking

Passwords are never stored directly.

Instead, bcrypt hashes them before saving.

---

# Authorization

Authentication answers:

"Who are you?"

Authorization answers:

"What are you allowed to do?"

I implemented Role Based Access Control.

Example roles:

- Admin
- User

Permissions are checked through middleware before protected routes execute.

---

# Middleware

I learned why middleware is useful.

Instead of checking authentication inside every handler, middleware does it once.

Current middleware includes:

- JWT Authentication
- Permission Checking

---

# Storage Abstraction

One feature I'm proud of is storage abstraction.

Instead of tightly coupling the application with Cloudflare R2, I created a storage provider interface.

Current providers:

- Local Storage
- Cloudflare R2
- Cloudinary

Changing providers only requires changing configuration.

No business logic changes.

---

# File Upload Flow

The upload flow looks like this:

User

↓

API

↓

Validation

↓

Storage Provider

↓

Database Metadata

Instead of only storing files, the application also stores metadata like:

- Filename
- Size
- MIME Type
- Provider
- Storage Key
- Upload Status

---

# Folder System

I also implemented folders.

A folder can contain:

- Files
- Other folders

This made the project feel much closer to Google Drive than a simple upload API.

---

# API Design

Routes are grouped logically.

Examples:

- Authentication
- Users
- Files
- Folders
- Admin
- Health

Protected routes automatically require authentication.

---

# Docker

Later in development I containerized the project.

The application now runs using Docker Compose.

Services include:

- Backend
- PostgreSQL

This eliminated "works on my machine" issues.

---

# Biggest Learnings

This project completely changed how I think about backend development.

Earlier I believed backend meant:

Receive Request

↓

Run SQL

↓

Return JSON

Now I understand that real backend systems involve much more.

- Architecture
- Separation of Concerns
- Authentication
- Authorization
- Storage
- Configuration
- Deployment
- Logging
- Database Design

---

# Challenges

Some issues I solved during development:

- Docker Networking
- PostgreSQL Container Setup
- Environment Variables
- JWT Middleware
- Cloudflare R2 Integration
- MIME Validation
- Docker Volumes
- Migration Execution
- Nginx Reverse Proxy

Each bug taught me something new.

---

# What I'm Most Proud Of

The biggest achievement isn't writing thousands of lines of Go code.

It's understanding how all the components work together.

A request now travels through multiple layers:

Client

↓

Router

↓

Middleware

↓

Handler

↓

Service

↓

Repository

↓

Database

or

↓

Storage Provider

That complete flow finally makes sense to me.

---

# What's Next

ByteVault V1 is now deployed on a production VPS.

The next version will focus on:

- Multipart Upload
- Resumable Uploads
- Better Dashboard
- Analytics
- Transfer Monitoring
- Better UX
- Performance Improvements

ByteVault started as a learning project.

My goal is to slowly evolve it into a real SaaS product.