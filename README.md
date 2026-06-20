# Connection Monitor

Connection Monitor is a local-first network health dashboard for a Synology NAS or any Docker host on your LAN. It tracks latency, DNS resolution health, speed test results, outages, and basic connection status over time.

This is a monorepo. The API service is Go with Echo REST APIs and SQLite. The web app is React, TypeScript, Vite, and Ant Design. The final Docker image serves the static web app from the Go service.

## Repository Layout

```text
apps/
  web/            React web app
services/
  api/            Go REST API and monitoring jobs
data/             Local runtime SQLite data, ignored by git
Dockerfile        Production container build
docker-compose.yml
```

Future clients such as iOS and Android should live under `apps/ios` and `apps/android`.

## What It Does

- Monitors TCP latency to configurable targets.
- Monitors DNS resolution for configurable domains.
- Runs manual and scheduled speed tests through a configurable CLI command.
- Stores history in a local SQLite database under `/data`.
- Detects outages when all latency targets fail for a configurable number of checks.
- Provides dashboard, speed test, latency, DNS, outage, and settings pages.
- Requires local login with users stored in SQLite.

## What It Does Not Do

- It does not expose itself publicly by default.
- It does not require Synology Web Station or QuickConnect.
- It does not expose public access or configure reverse proxies for you.
- It does not replace full observability tooling such as Prometheus.

## Run Locally

Backend:

```bash
cd services/api
go mod download
DB_PATH=./connection-monitor.db STATIC_PATH=../../apps/web/dist go run ./cmd/server
```

Frontend dev server:

```bash
cd apps/web
npm install
npm run dev
```

Open `http://localhost:5173`. Vite proxies `/api` to `http://localhost:8080`.

## Tests

```bash
cd services/api
go test ./...

cd ../../apps/web
npm test
npm run build
```

Backend tests use real temporary SQLite databases and `httptest`. They do not use mock frameworks.

## CI/CD

GitHub Actions workflows live in `.github/workflows`.

### Pull Request And Main Validation

`Validate` runs on pull requests to `main` and pushes to `main`.

It validates the monorepo in separate jobs:

- `API`: sets up Go from `services/api/go.mod`, checks `gofmt`, checks `go mod tidy`, runs `go vet ./...`, and runs `go test ./...`.
- `Web`: sets up Node 24, installs with `npm ci`, runs frontend tests, and builds the Vite app.
- `Container`: validates `docker-compose.yml` and builds the production Docker image without pushing it.

The workflow uses per-project dependency caches and cancels older in-progress validation runs for the same branch.

### Releases

`Release` runs on pushes to `main` and uses Release Please.

Use Conventional Commit messages such as:

```text
feat(api): add outage classifier
fix(web): show settings save errors
chore: update docker metadata
```

Release Please opens or updates a release PR. When that PR is merged, it creates:

- `CHANGELOG.md`
- a git tag such as `v0.2.0`
- a GitHub Release

The starting release version is tracked in `.release-please-manifest.json`.

### Container Publishing

`Publish Container` runs when a GitHub Release is published. It builds the production image and pushes it to GitHub Container Registry:

```text
ghcr.io/OWNER/REPOSITORY
```

Release builds publish:

- the release tag, for example `v0.2.0`
- `latest` for non-prerelease releases

The workflow can also be run manually with a custom image tag.

### Dependency Updates

Dependabot is configured for:

- GitHub Actions
- Go modules in `services/api`
- npm packages in `apps/web`
- Docker base images at the repository root

### Required GitHub Settings

Recommended repository settings:

- Protect `main`.
- Require the `Validate` workflow before merging.
- Use squash or rebase merges to keep a readable conventional commit history.
- Allow GitHub Actions to create pull requests so Release Please can manage release PRs.
- Keep package write permissions enabled for GitHub Actions if publishing to GHCR.

## Docker Compose

Set the initial admin credentials before the first run. They are only used when the database has no users yet:

```bash
cp .env.example .env
```

Edit `.env`, then start the app:

```bash
docker compose up -d --build
```

The app listens on:

```text
http://SYNOLOGY_IP:8810
```

Default local compose volume:

```text
./data:/data
```

For Synology, set `DATA_DIR` before starting Compose or create a `.env` file:

```bash
DATA_DIR=/volume1/docker/connection-monitor/data docker compose up -d --build
```

Synology compose volume:

```text
/volume1/docker/connection-monitor/data:/data
```

The SQLite database is stored at:

```text
/volume1/docker/connection-monitor/data/connection-monitor.db
```

Default `.env.example` credentials are `admin` / `changeme123`. Change them before deployment, or change the password from the Users page after first login.

## Synology Container Manager

1. Create `/volume1/docker/connection-monitor/data`.
2. Copy this repository to your NAS or build from a Git source supported by your workflow.
3. Create a `.env` file next to `docker-compose.yml` with `DATA_DIR=/volume1/docker/connection-monitor/data`, `ADMIN_USERNAME`, and `ADMIN_PASSWORD`.
4. In Container Manager, create a project using `docker-compose.yml`.
5. Build and start the service.
6. Open `http://SYNOLOGY_IP:8810` from your LAN.

No Web Station, reverse proxy, QuickConnect, or public port forwarding is required.

## Speed Test Command

The default command is:

```text
speedtest --accept-license --accept-gdpr --format=json
```

The Docker image installs the official Ookla CLI by default. The app expects Ookla JSON output or simple JSON with fields like `download_mbps`, `upload_mbps`, `ping_ms`, and `jitter_ms`. If the command is missing or fails, the failed result is stored and shown in the UI with the error.

To build without bundling the Ookla CLI:

```bash
docker build --build-arg INSTALL_SPEEDTEST=false -t connection-monitor .
```

## Users And Login

On first startup, the app creates one admin user from:

```text
ADMIN_USERNAME
ADMIN_PASSWORD
```

After users exist in SQLite, changing those environment variables does not overwrite existing users. Use the Users page to add users, rename users, change passwords, or delete users.

## Keep It LAN-Only

- Do not add router port forwarding for port `8810`.
- Do not expose it through QuickConnect.
- Bind Compose only on your LAN host. For stricter binding, change ports to `"192.168.1.10:8810:8080"` using your NAS LAN IP.
- Use a VPN if you need remote access.

## Back Up SQLite

Stop the container or use SQLite online backup tooling, then copy:

```text
/volume1/docker/connection-monitor/data/connection-monitor.db
```

For a simple offline backup:

```bash
docker compose stop
cp /volume1/docker/connection-monitor/data/connection-monitor.db /volume1/docker/connection-monitor/data/connection-monitor.db.bak
docker compose start
```

## Troubleshooting

- `database is locked`: the app uses WAL and a single DB writer connection. Check that no external process is holding the database open.
- Speed test failures: verify the configured command exists inside the container and emits JSON.
- Empty dashboard: monitoring starts after the server starts; wait for the first latency/DNS interval or run a speed test manually.
- DNS failures: confirm the container has DNS access and your NAS DNS settings are valid.
- Cannot open UI: confirm Container Manager maps host `8810` to container `8080` and that your NAS firewall allows LAN access.
