# API Service

Go REST API using Echo with SQLite storage and background monitoring jobs.

This service lives at `services/api` inside the monorepo. It owns persistence, monitoring jobs, authentication, and static web serving for production.

Run:

```bash
go run ./cmd/server
```

Environment:

- `ADDR`: listen address, default `:8080`
- `DB_PATH`: SQLite path, default `./connection-monitor.db`
- `STATIC_PATH`: web app static path, default `../../apps/web/dist`
- `APP_ENV`: informational environment value
- `ADMIN_USERNAME`: first admin username when the database has no users
- `ADMIN_PASSWORD`: first admin password when the database has no users
