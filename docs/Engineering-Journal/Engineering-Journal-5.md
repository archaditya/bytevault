# Engineering Journal #5

## Nginx Production Hardening

**Date:** 8 July 2026

---

# Objective

With Docker secured, the next step was hardening the public entry point of the application.

Since every external request reaches PushPort through Nginx, it became the first security boundary.

---

# Problems Identified

The previous configuration:

* leaked server information
* lacked security headers
* allowed hidden file probing to reach the application
* had no request throttling
* used default compression settings
* lacked optimized proxy configuration

---

# Implemented Improvements

---

## 1. Hide Server Version

Enabled:

```nginx
server_tokens off;
```

Prevents exposing exact Nginx version.

---

## 2. Improved Compression

Configured Gzip for:

* JSON
* JavaScript
* CSS
* XML
* SVG

Benefits

* Lower bandwidth
* Faster API responses

---

## 3. Upload Size

Configured:

```nginx
client_max_body_size 5G;
```

Preparing PushPort for future large file uploads.

---

## 4. Proxy Timeouts

Configured:

* proxy_connect_timeout
* proxy_send_timeout
* proxy_read_timeout

Required for:

* Large uploads
* Slow client connections
* Future multipart uploads

---

## 5. Security Headers

Added:

* X-Frame-Options
* X-Content-Type-Options
* Referrer-Policy
* Permissions-Policy

These headers improve browser-side security against common attacks.

---

## 6. Hidden File Protection

Added:

```nginx
location ~ /\.(?!well-known).* {
    deny all;
}
```

Now requests like

```
.env

.git

.env.production
```

are rejected by Nginx before reaching the Go application.

---

## 7. Rate Limiting

Configured:

```
10 Requests / Second

Burst 20
```

Purpose:

* Reduce automated scanning
* Slow down abusive clients
* Protect backend resources

---

### Verification

Sequential curl testing appeared successful because requests were processed one after another.

Proper concurrent testing using ApacheBench revealed:

```
200 Requests

↓

174 Rejected
```

confirming that rate limiting worked correctly.

---

### Interesting Observation

Default Nginx behavior returns:

```
503 Service Temporarily Unavailable
```

when requests exceed the configured rate.

For APIs, returning

```
429 Too Many Requests
```

is more semantically correct.

Future improvement:

```nginx
limit_req_status 429;
```

---

# Architecture

```
Internet

↓

Cloudflare

↓

Nginx

├── HTTPS

├── Reverse Proxy

├── Rate Limiting

├── Security Headers

├── Hidden File Protection

├── Compression

└── Timeout Handling

↓

Go Backend
```

---

# Lessons Learned

* Nginx is much more than a reverse proxy.
* Security should be handled as early as possible in the request lifecycle.
* Rate limiting must be tested with concurrent requests rather than sequential ones.
* API gateways should return HTTP 429 instead of generic HTTP 503 for throttling.
* Browser security headers provide additional protection without modifying application code.

---

# Future Improvements

* Fail2Ban
* SSH Hardening
* Content Security Policy (CSP)
* Custom error pages
* Access log optimization
* Prometheus metrics
* Grafana monitoring
* Automated PostgreSQL backups
