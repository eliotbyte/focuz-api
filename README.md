# Focuz API

Backend API for a note-taking application with workspaces, notes, charts, and activities.

## Quick Start

1. Create a `.env` file in the project root:
```env
POSTGRES_USER=focuz_user
POSTGRES_PASSWORD=focuz_password
POSTGRES_DB=focuz_db
JWT_SECRET=your_secret_key_here
MINIO_EXTERNAL_ENDPOINT=http://localhost:9000
```

2. Start the project:
```bash
docker-compose up -d
```

3. The API will be available at: http://localhost:8080
4. API documentation (Swagger UI): http://localhost:8081

## Project Structure

- `main.go` - application entry point
- `handlers/` - HTTP handlers
- `models/` - data models
- `repository/` - data access layer
- `migrations/` - database migrations
- `middleware/` - middleware (CORS, authentication)
- `initializers/` - service initializers
- `types/` - data types
- `tests/` - tests

## API Endpoints

### Authentication
- `POST /register` - user registration
- `POST /login` - user login

### Workspaces (Spaces)
- `GET /spaces` - get available workspaces
- `POST /spaces` - create a workspace
- `PATCH /spaces/{id}` - update a workspace
- `PATCH /spaces/{id}/delete` - soft delete a workspace
- `PATCH /spaces/{id}/restore` - restore a workspace
- `GET /spaces/{id}/users` - get users in a workspace
- `POST /spaces/{id}/invite` - invite a user
- `DELETE /spaces/{id}/users/{userId}` - remove a user from a workspace

### Notes
- `GET /notes` - get notes
- `POST /notes` - create a note
- `GET /notes/{id}` - get a note by ID
- `PATCH /notes/{id}/delete` - soft delete a note
- `PATCH /notes/{id}/restore` - restore a note
- `GET /tags/autocomplete` - tag autocomplete

### Activities
- `GET /activities` - get activity analysis
- `POST /activities` - create an activity
- `PATCH /activities/{id}` - update an activity
- `PATCH /activities/{id}/delete` - soft delete an activity
- `PATCH /activities/{id}/restore` - restore an activity

### Activity Types
- `GET /spaces/{spaceId}/activity-types` - get activity types
- `POST /spaces/{spaceId}/activity-types` - create an activity type
- `PATCH /spaces/{spaceId}/activity-types/{typeId}/delete` - soft delete an activity type
- `PATCH /spaces/{spaceId}/activity-types/{typeId}/restore` - restore an activity type

### Charts
- `GET /charts` - get charts
- `POST /charts` - create a chart
- `PATCH /charts/{id}` - update a chart
- `PATCH /charts/{id}/delete` - soft delete a chart
- `PATCH /charts/{id}/restore` - restore a chart
- `GET /chart-types` - get chart types
- `GET /period-types` - get period types

### Attachments
- `POST /upload` - upload a file
- `GET /files/{id}` - get a file

### Sync (Offline)

- `GET /sync?since=<RFC3339>&spaceId?=<id>` — pull changes since timestamp. Returns notes, tags, filters, charts, activities, spaces changed after `since`. Use for polling or after WS/SSE events.
- `POST /sync` — push local changes. Body contains arrays: `notes`, `tags`, `filters`, `charts`, `activities`. Server applies with last-write-wins by `modified_at` and returns `mappings` (clientId -> serverId) and `conflicts`.

Example pull:
```bash
curl -H "Authorization: Bearer <TOKEN>" \
     "http://localhost:8080/sync?since=2024-01-01T00:00:00Z"
```

Example push:
```bash
curl -X POST -H "Authorization: Bearer <TOKEN>" -H "Content-Type: application/json" \
  -d '{
    "notes": [
      {
        "clientId": "tmp-123",
        "space_id": 1,
        "user_id": 1,
        "text": "Hello",
        "tags": ["important"],
        "created_at": "2025-01-01T10:00:00Z",
        "modified_at": "2025-01-01T10:00:00Z"
      }
    ],
    "tags": [], "filters": [], "charts": [], "activities": []
  }' \
  http://localhost:8080/sync
```

### Utilities by Space

- `GET /spaces/{spaceId}/tags` — list tags in space.
- `GET /spaces/{spaceId}/filters` — list filters in space (alias of `GET /filters?spaceId=...`).

## Filters

Saved note filters with nested grouping and JSON parameters.

- Create: POST `/filters` { spaceId, parentId?, name, params<object/json> }
- List: GET `/filters?spaceId=...&page=1&pageSize=20`
- Update: PATCH `/filters/{id}` { name?, parentId?, params? }
- Delete: PATCH `/filters/{id}/delete`
- Restore: PATCH `/filters/{id}/restore`

## Technologies

- **Go 1.24** - main language
- **Gin** - web framework
- **PostgreSQL** - database
- **pgroonga** - full-text search
- **MinIO** - object storage
- **JWT** - authentication
- **Docker** - containerization

## Development

### Run tests
```bash
# Quick test run (PowerShell)
.\run-tests.ps1
```

### Local development
```bash
# Start only the database and MinIO
# IMPORTANT: the DB image includes PGroonga extension required by migrations
# If you run Postgres yourself, ensure PGroonga is installed, or migrations will fail
# Recommended:
 docker-compose up db minio -d

# Run the API locally
 go run main.go
```

### Environment notes
- `ALLOWED_ORIGINS`: comma-separated origins allowed in production for CORS and WebSocket (e.g. `https://app.example.com,https://staging.example.com`).
- `TRUSTED_PROXIES`: comma-separated proxy CIDRs or IPs for correct client IP; defaults to `127.0.0.1, ::1` when unset.
- `RATE_LIMIT_RPS`, `RATE_LIMIT_BURST`, `RATE_LIMIT_WHITELIST`, `RATE_LIMIT_ENABLED`: tune/disable rate limiting.
- `MINIO_EXTERNAL_ENDPOINT`: external hostname:port for presigned URLs; if empty, internal endpoint is used.

## API Documentation

Swagger UI is available at: http://localhost:8081

The documentation is automatically updated when the `openapi.yaml` file changes. 