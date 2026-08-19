# Engineering Journal #3

# Deploying PushPort V1 to Production

Date: July 2026

---

## Objective

Deploy PushPort on a real production server instead of running it only on localhost.

---

# Infrastructure

Frontend

- Vercel

Backend

- Ubuntu VPS (OVH)

Database

- PostgreSQL (Docker)

Reverse Proxy

- Nginx

SSL

- Let's Encrypt

DNS

- Cloudflare

Storage

- Cloudflare R2

---

# Step 1

Purchased an OVH VPS.

Installed Ubuntu 24.04.

Connected using SSH.

```

ssh ubuntu@<server-ip>

```

---

# Step 2

Updated Ubuntu packages.

```

sudo apt update
sudo apt upgrade -y

```

Also learned that kernel updates require a reboot.

---

# Step 3

Installed Docker.

Learned:

- Docker Engine
- Docker Compose
- Docker Groups
- User Permissions

Added current user to docker group.

```

sudo usermod -aG docker ubuntu

```

---

# Step 4

Installed Git.

Cloned PushPort repository.

```

git clone ...

```

---

# Step 5

Created environment variables.

Configured:

- Database
- JWT
- Cloudflare R2
- Storage Provider

---

# Step 6

Started services.

```

docker compose up -d

```

Encountered multiple issues.

---

# Problems Solved

## PostgreSQL Version

Initially used PostgreSQL 18.

Docker volume compatibility caused startup failures.

Downgraded to PostgreSQL 17.

---

## Docker Networking

Backend couldn't resolve the database hostname.

Learned how Docker Compose networking works.

---

## Missing .env

Backend started but couldn't load configuration.

Created production environment file.

---

## Container Logs

Learned to debug using:

```

docker compose logs

```

instead of guessing.

---

# Step 7

Backend became healthy.

Health endpoint:

```

/api/v1/health

```

returned

```

{
"status":"healthy"
}

```

---

# Step 8

Installed Nginx.

Configured reverse proxy.

Traffic Flow:

Internet

↓

Nginx

↓

Go Backend

Instead of exposing port 8080 directly.

---

# Step 9

Configured Cloudflare DNS.

Created subdomains:

- PushPort.archadi.dev
- api-PushPort.archadi.dev

Learned about:

- A Records
- DNS Propagation
- Proxy Mode

---

# Step 10

Installed SSL.

Used Certbot.

```

sudo certbot --nginx

```

Generated HTTPS certificate.

Configured automatic renewal.

---

# Step 11

Connected frontend.

Hosted on Vercel.

Updated production environment variables.

Frontend now communicates securely with:

```

https://api.PushPort.archadi.dev

```

---

# Final Architecture

User

↓

Vercel

↓

Nginx

↓

Go Backend

↓

PostgreSQL

↓

Cloudflare R2

---

# Biggest Learnings

Deployment isn't only about writing code.

It involves:

- Linux
- SSH
- Docker
- Networking
- Reverse Proxies
- DNS
- SSL
- Cloud Infrastructure
- Debugging

---

# Production Challenges

Solved issues including:

- PostgreSQL startup failure
- Docker volumes
- Docker networking
- Environment variables
- Nginx configuration
- DNS propagation
- Cloudflare 521 errors
- HTTPS configuration

Every issue improved my understanding of production systems.

---

# Outcome

PushPort V1 is now publicly deployed.

This is my first self-hosted production backend running on my own VPS with Docker, Nginx, HTTPS, PostgreSQL, and Cloudflare.

The next milestone is PushPort V2.