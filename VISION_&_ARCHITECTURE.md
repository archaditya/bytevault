# ByteVault Architecture Roadmap

## Project Vision

ByteVault is a high-performance file storage and processing platform.

Core goals:

1. Efficient large file uploads
2. Resumable multipart uploads
3. Real-time upload visibility
4. Asynchronous file processing
5. Scalable architecture from day one
6. Direct object-storage uploads (server never becomes file-transfer bottleneck)

---

# Version 1 (Storage Platform)

## Goal

Build a reliable cloud file storage platform.

No background processing.
No queues.
No WebSockets.

## Components

Frontend

* React / Next.js

Backend

* Go (net/http)

Database

* PostgreSQL

Object Storage

* Cloudflare R2

## File Upload Flow

1. User selects file.
2. Frontend requests upload session.
3. Backend creates upload record.
4. Backend generates presigned upload URL.
5. Frontend uploads file directly to R2.
6. Frontend notifies backend that upload completed.
7. Backend verifies object existence.
8. Backend creates file record.
9. File appears in dashboard.

## Database Schema

### files

* id
* user_id
* file_name
* file_size
* mime_type
* storage_key
* status
* created_at
* updated_at

### File Status

UPLOADING
READY
FAILED

## Important Rule

Files must never pass through backend servers.

Allowed:

Client → R2

Not Allowed:

Client → Backend → R2

---

# Version 2 (Resumable Upload System)

## Goal

Support large files and upload recovery.

## New Components

Redis

Used for:

* Upload progress
* Temporary upload state
* Active upload sessions

## Upload Method

Switch from single upload to multipart upload.

## Upload Flow

1. User selects file.

2. Frontend splits file into chunks.

3. Backend creates multipart upload session.

4. Backend returns:

   * upload_id
   * chunk_size
   * presigned URLs

5. Frontend uploads chunks in parallel.

6. After each chunk upload:

   * frontend updates progress
   * backend stores progress in Redis

7. User can leave upload page.

8. Dashboard can retrieve progress from Redis.

9. Upload can resume after interruption.

## New Database Table

### uploads

* id
* user_id
* file_name
* total_chunks
* uploaded_chunks
* upload_status
* created_at

## Upload Status

CREATED
UPLOADING
PAUSED
COMPLETED
FAILED

## Important Rule

Redis stores transient upload state.

PostgreSQL stores permanent metadata.

---

# Version 3 (Real-Time Upload Visibility)

## Goal

Provide live updates across all screens.

## New Components

WebSocket Server

## Flow

1. Upload starts.
2. Frontend uploads chunks.
3. Progress updates sent to backend.
4. Backend updates Redis.
5. Backend broadcasts progress through WebSocket.
6. Any connected dashboard instantly updates.

## Example

User starts upload.

User navigates to dashboard.

Dashboard still sees:

File.mp4

73% uploaded

without refreshing page.

## Important Rule

WebSocket is used for visibility only.

Actual upload data still goes directly to R2.

---

# Version 4 (File Processing Platform)

## Goal

Transform ByteVault from storage product into processing platform.

## New Components

Asynq

Worker Pool

## Processing Flow

Upload Complete

↓

Create Processing Job

↓

Worker Receives Job

↓

Run Processing Tasks

↓

Update File Status

↓

Broadcast Progress

## Processing Tasks

Metadata Extraction

Examples:

* image dimensions
* video duration
* PDF page count
* file hashes

Thumbnail Generation

Examples:

* image preview
* PDF preview
* video poster frame

Virus Scanning

Document Text Extraction

OCR

Search Indexing

## New File Statuses

UPLOADING

READY_FOR_PROCESSING

PROCESSING

READY

FAILED

## Processing Stages

metadata

thumbnail

virus_scan

ocr

indexing

complete

## Dashboard Example

Resume.pdf

Processing

Current Stage:
thumbnail

Progress:
35%

---

# Final Architecture

Frontend
│
▼

Go API (net/http)

```
│
├──────── PostgreSQL
│
├──────── Redis
│
├──────── WebSocket Hub
│
└──────── Asynq

             │
             ▼

         Workers

             │
             ▼

     Cloudflare R2
```

---

# Architectural Rules

Rule 1

Large files must always upload directly to R2.

Rule 2

Backend stores metadata, never raw files.

Rule 3

Temporary upload state belongs in Redis.

Rule 4

Permanent state belongs in PostgreSQL.

Rule 5

Heavy processing must never run inside HTTP request handlers.

Always use background workers.

Rule 6

WebSockets are only for realtime visibility.

They are not part of upload transport.

Rule 7

Every file must move through a lifecycle:

CREATED

↓

UPLOADING

↓

COMPLETED

↓

PROCESSING

↓

READY

or

FAILED

Rule 8

System should be horizontally scalable.

Any API instance should be replaceable without affecting uploads.