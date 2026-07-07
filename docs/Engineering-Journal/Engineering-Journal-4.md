# Engineering Journal #4

## Production Docker & CI/CD Hardening

**Date:** 7 July 2026

---

# Objective

The backend was already deployed and functional, but the deployment architecture still resembled a development setup.

The goal of this sprint was to make Docker, CI/CD, and deployment infrastructure production-ready before implementing more product features.

---

# Problems Identified

### 1. Development Docker Image

The Docker image contained:

* Go compiler
* Build cache
* Source code
* Dependencies

This resulted in:

* Large image size
* Slower deployments
* Larger attack surface

---

### 2. Backend Exposed Directly

Docker Compose exposed:

```yaml
ports:
  - "8080:8080"
```

Meaning:

```
Internet
    │
    ├── Nginx
    │
    └── Backend
```

Anyone could bypass Nginx and access the API directly.

---

### 3. No Container Health Monitoring

Docker only knew whether:

* the process existed

It had no idea whether:

* PostgreSQL was connected
* API was serving requests
* Application had started correctly

---

### 4. CI/CD Issues

Workflow contained unnecessary restart:

```
docker compose up --build

↓

docker compose restart backend
```

This introduced unnecessary downtime.

---

### 5. Unlimited Docker Logs

Containers were configured with default logging.

Long-running production containers would eventually consume disk space.

---

# Implemented Improvements

---

## 1. Multi-stage Docker Build

Replaced the previous Dockerfile with a production multi-stage build.

Builder Stage

```
Go Compiler

↓

Compile Binary
```

Runtime Stage

```
Only Binary

+

CA Certificates

+

Minimal Alpine Image
```

### Benefits

* Smaller image
* Faster deployments
* Reduced attack surface
* Cleaner production container

---

## 2. Docker Health Check

Added Docker Health Check.

Instead of only checking whether the process existed, Docker now verifies:

```
GET /api/v1/health
```

Docker now reports:

```
healthy

or

unhealthy
```

instead of simply

```
Up
```

---

## Interesting Debugging

Initially the health check continuously failed.

Reason:

BusyBox wget behaves differently than GNU wget.

Health Check

```
wget --spider
```

always returned Exit Code 6 despite the endpoint being healthy.

After debugging inside the running container:

```
wget -O-
```

worked correctly.

Lesson:

Never assume Linux utilities behave identically across distributions.

---

## 3. Restart Policies

Every service now uses

```yaml
restart: unless-stopped
```

Benefits

* Survives Docker daemon restart
* Automatic recovery after VPS reboot
* Production-friendly behavior

---

## 4. Docker Log Rotation

Added:

```yaml
logging:
  driver: json-file
  options:
    max-size: "10m"
    max-file: "5"
```

Previously:

```
Logs

↓

Grow Forever

↓

Disk Full
```

Now:

```
10 MB

↓

Rotate

↓

Maximum 5 Files
```

---

## 5. Redis Added

Integrated Redis into Docker Compose.

Configuration:

* Persistent volume
* Restart policy
* Internal Docker networking

Redis is now available for:

* Notification System
* Future Queues
* Caching
* Background Workers

---

## 6. Backend Localhost Binding

Changed

```yaml
8080:8080
```

to

```yaml
127.0.0.1:8080:8080
```

Architecture changed from

```
Internet

↓

Backend
```

to

```
Internet

↓

Nginx

↓

Backend
```

This removed direct public access to the backend.

---

## 7. Firewall Cleanup

Removed

```
8080
```

from UFW.

Current exposed ports:

```
22

80

443
```

Only Nginx is publicly accessible.

---

## 8. CI/CD Improvements

Deployment flow became:

```
Git Push

↓

GitHub Actions

↓

SSH

↓

Git Pull

↓

Docker Build

↓

Docker Up

↓

Health Check

↓

Cleanup

↓

Deployment Complete
```

Removed unnecessary:

```
docker compose restart backend
```

---

# Architecture Before

```
Internet

├── Backend

└── Nginx
```

---

# Architecture After

```
Internet

↓

Cloudflare

↓

Nginx

↓

127.0.0.1:8080

↓

Docker

↓

Go Backend

├── PostgreSQL

└── Redis
```

---

# Lessons Learned

* Multi-stage Docker builds significantly reduce production image size.
* Health checks verify application health, not just process existence.
* Binding services to localhost prevents accidental public exposure.
* Docker log rotation is essential for long-running servers.
* CI/CD should avoid unnecessary restarts.
* Production containers should always restart automatically after failures.

---

# Future Improvements

* Zero-downtime deployments
* Database migration step
* Rollback strategy
* Prometheus health monitoring
* Docker image vulnerability scanning