# Focuz API

Backend API for a note-taking application with workspaces, topics, and activities.

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

### Topics
- `GET /topic-types` - get topic types
- `POST /topics` - create a topic
- `PATCH /topics/{id}` - update a topic
- `PATCH /topics/{id}/delete` - soft delete a topic
- `PATCH /topics/{id}/restore` - restore a topic
- `GET /spaces/{spaceId}/topics` - get topics in a workspace

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
go test ./...
```

### Local development
```bash
# Start only the database and MinIO
docker-compose up db minio -d

# Run the API locally
go run main.go
```

## API Documentation

Swagger UI is available at: http://localhost:8081

The documentation is automatically updated when the `openapi.yaml` file changes. 