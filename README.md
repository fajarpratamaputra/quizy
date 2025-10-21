# quizy

Quizy exposes quiz questions stored in PostgreSQL via a JSON API. The reusable service lives under `pkg/quiz`, so the code can run locally (`go run .`) or as a Vercel serverless function.

## Project Layout
- `main.go` / `cmd/server` - local HTTP server entrypoints
- `api/quiz.go` - Vercel entrypoint (`package handler` with exported `Handler`)
- `pkg/quiz` - shared service (DB access, pagination, response shaping)

## Requirements
- Go 1.17+
- PostgreSQL (Supabase connection strings work out of the box)
- `DB_DSN` environment variable containing the connection URI, e.g. `postgres://user:pass@host:5432/db?sslmode=require`

## Local Development
1. Copy your DSN into `.env` (`DB_DSN=postgres://...`).
2. Start the server: `go run .` (or `go run ./cmd/server`). Server listens on `:8080` or `${PORT}`.
3. Hit `http://localhost:8080/api/quiz?limit=10&page=1` to verify.

## Deploying to Vercel
1. Move the project into a Vercel workspace.
2. Set the `DB_DSN` environment variable in the Vercel dashboard (Project Settings -> Environment Variables).
3. Deploy normally (`vercel --prod`). The function is available at `/api/quiz`.
4. For local parity, `vercel dev` will execute `api/quiz.go` and respect your `.env` / Vercel env vars.

## Database Schema Example
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

## Development Tips
- `go test ./...` ensures the module builds.
- Update this doc when the API or deployment flow changes.
