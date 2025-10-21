# quizy

Quizy is a small Go service that exposes quiz questions stored in PostgreSQL via a JSON API. It uses the Go standard library and `github.com/lib/pq`.

## Features
- JSON HTTP endpoint with pagination metadata
- Configurable via the `DB_DSN` environment variable
- Sensible PostgreSQL connection pooling defaults

## Getting Started
Go 1.17+ and a reachable PostgreSQL instance are required.

1. Configure the environment variable `DB_DSN` (for example `postgres://user:pass@localhost:5432/quizy?sslmode=disable`). You can keep it in a local `.env` file when using a dotenv loader.
2. Download dependencies once with `go mod tidy`.
3. Run the server with `go run main.go`. The API listens on `:8080`.

### Example schema
```sql
CREATE TABLE questions (
  id SERIAL PRIMARY KEY,
  question TEXT NOT NULL,
  answer_a VARCHAR(255),
  answer_b VARCHAR(255),
  answer_c VARCHAR(255),
  correct VARCHAR(255),
  created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## API
Endpoint: `GET /api/quiz`

Query parameters:
- `limit` (optional) defaults to 10 and caps at 100
- `page` (optional) defaults to 1

Response example:
```json
{
  "data": [
    {
      "question": "What is 2 + 2?",
      "answers": ["3", "4", "5"],
      "correct": "4"
    }
  ],
  "meta": {
    "page": 1,
    "limit": 10,
    "total_pages": 5
  }
}
```

## Development
- `go test ./...` to confirm the project builds.
- Update this document whenever the API surface changes.
