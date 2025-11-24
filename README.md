# LevelTalk

LevelTalk is a small Go 1.22+ web app that helps language learners generate CEFR-aligned dialogs using htmx-driven forms. Dialogs are persisted in PostgreSQL together with placeholder audio URLs produced via a TTS stub so the UI can render `<audio>` players.

## Features

- Clean Go module layout (`cmd`, `internal`, `migrations`).
- Server-side rendered UI with htmx-enhanced forms (no SPA).
- Stubbed LLM + TTS clients with clear TODOs for real integrations.
- PostgreSQL persistence layer with repository, migrations, and tests.
- Dockerfile + docker-compose for local development.

## Requirements

- Go 1.23 or newer
- Docker + Docker Compose v2
- Access to a PostgreSQL instance (Docker compose provides one)

## Configuration

The app reads environment variables via `internal/config`:

| Variable | Description | Required | Example |
| --- | --- | --- | --- |
| `DB_DSN` | PostgreSQL connection string | ✅ | `postgres://leveltalk:leveltalk@localhost:5432/leveltalk?sslmode=disable` |
| `PORT` | HTTP port (default `8080`) | ❌ | `8080` |
| `LLM_API_KEY` | placeholder for future LLM integration | ❌ | `sk-...` |
| `ELEVENLABS_API_KEY` | placeholder for ElevenLabs | ❌ | `elevenlabs-...` |

## Running locally (without Docker)

```bash
export DB_DSN="postgres://leveltalk:leveltalk@localhost:5432/leveltalk?sslmode=disable"
go run ./cmd/server
```

The server automatically runs SQL migrations on startup. Visit <http://localhost:8080>.

## Database migrations

All SQL files in `migrations/` are embedded and executed at startup via `storage.RunMigrations`. For manual execution you can run the server with `DB_DSN` pointing to the desired database—no additional tooling is required right now.

## Tests

```bash
go test ./...
```

Tests currently cover:

- LLM stub guarantees input words are present.
- Dialog repository create/search logic using `sqlmock`.

## Docker workflow

1. Start everything:

   ```bash
   docker compose up --build
   ```

2. Access the app at <http://localhost:8080>.

`docker-compose.yml` provisions two services:

- `app`: builds the Go binary and exposes port 8080, automatically applying migrations on startup.
- `db`: PostgreSQL 16 with a named volume (`pgdata`) for persistence.

To tear everything down (including volumes):

```bash
docker compose down -v
```


