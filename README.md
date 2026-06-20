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

Connection Monitor is designed to run as one container on Synology Container Manager. The Go API serves the React build, so you do not need Web Station, a separate nginx container, or a second frontend service.

### Recommended NAS Layout

Use one parent directory for the app and one persistent directory for SQLite data:

```text
/volume1/docker/connection-monitor/
  app/       Repository checkout or uploaded source bundle
  data/      SQLite database and WAL files
```

The `data` directory is the only runtime state you must preserve. Do not store the database inside the source checkout because application updates may replace that directory.

### What To Ship

For a source-based install, ship the repository root with:

- `Dockerfile`
- `docker-compose.yml`
- `.env.example`
- `apps/web`
- `services/api`
- `README.md`

Do not ship generated or local runtime directories:

- `.git`, unless you intentionally deploy with Git
- `data`
- `.cache`
- `apps/web/node_modules`
- `apps/web/dist`
- local `.env` files containing real credentials

Create a clean source archive from your workstation with:

```bash
tar \
  --exclude='.git' \
  --exclude='data' \
  --exclude='.cache' \
  --exclude='apps/web/node_modules' \
  --exclude='apps/web/dist' \
  -czf connection-monitor-source.tar.gz .
```

Upload that archive to the NAS and extract it under:

```text
/volume1/docker/connection-monitor/app
```

### Environment File

Create `/volume1/docker/connection-monitor/app/.env` before the first start:

```bash
DATA_DIR=/volume1/docker/connection-monitor/data
ADMIN_USERNAME=admin
ADMIN_PASSWORD=replace-this-password
```

`ADMIN_USERNAME` and `ADMIN_PASSWORD` are only used to create the first admin user when the database has no users. After first boot, manage users and passwords from the Users page.

### Install With Container Manager UI

1. Open File Station.
2. Create `/volume1/docker/connection-monitor/app`.
3. Create `/volume1/docker/connection-monitor/data`.
4. Upload or clone the project into `/volume1/docker/connection-monitor/app`.
5. Create `/volume1/docker/connection-monitor/app/.env` with `DATA_DIR`, `ADMIN_USERNAME`, and `ADMIN_PASSWORD`.
6. Open Container Manager.
7. Go to `Project`.
8. Choose `Create`.
9. Set the project name to `connection-monitor`.
10. Set the project path to `/volume1/docker/connection-monitor/app`.
11. Select the existing `docker-compose.yml`.
12. Build and start the project.
13. Open `http://SYNOLOGY_IP:8810` from a LAN device.

The first build compiles the Go service, builds the React app, and installs the official Ookla Speedtest CLI in the final image. The NAS needs internet access during that build.

### Install With SSH

If SSH is enabled on the NAS, the same deployment can be done from a shell:

```bash
ssh admin@SYNOLOGY_IP
sudo -i
mkdir -p /volume1/docker/connection-monitor/app /volume1/docker/connection-monitor/data
cd /volume1/docker/connection-monitor/app
git clone https://github.com/OWNER/REPOSITORY.git .
cp .env.example .env
vi .env
docker compose up -d --build
docker compose logs -f
```

Some Synology installations provide Compose as `docker-compose` instead of `docker compose`:

```bash
docker-compose up -d --build
docker-compose logs -f
```

### Install From A GitHub Release Image

The CI/CD pipeline publishes release images to GitHub Container Registry after a GitHub Release is created:

```text
ghcr.io/OWNER/REPOSITORY:v0.2.0
```

To deploy a prebuilt image instead of building on the NAS, replace the `build: .` line in `docker-compose.yml` with an image reference:

```yaml
services:
  connection-monitor:
    image: ghcr.io/OWNER/REPOSITORY:v0.2.0
    container_name: connection-monitor
    ports:
      - "8810:8080"
    volumes:
      - ${DATA_DIR:-./data}:/data
    environment:
      - TZ=${TZ:-Europe/Madrid}
      - DB_PATH=/data/connection-monitor.db
      - APP_ENV=production
      - STATIC_PATH=/app/apps/web/dist
      - ADMIN_USERNAME=${ADMIN_USERNAME:-admin}
      - ADMIN_PASSWORD=${ADMIN_PASSWORD:-changeme123}
    restart: unless-stopped
```

Use immutable release tags such as `v0.2.0` for NAS deployments. `latest` is convenient for testing but makes rollbacks less predictable.

### Offline Image Transfer

If the NAS cannot build or pull images, build the image on another machine and transfer it manually.

On your workstation:

```bash
docker build -t connection-monitor:local .
docker save connection-monitor:local -o connection-monitor-image.tar
```

Upload `connection-monitor-image.tar` to the NAS, then load it:

```bash
ssh admin@SYNOLOGY_IP
sudo -i
docker load -i /path/to/connection-monitor-image.tar
```

Use `image: connection-monitor:local` in `docker-compose.yml` instead of `build: .`.

### Updating On Synology

Back up the database before updating:

```bash
docker compose stop
cp /volume1/docker/connection-monitor/data/connection-monitor.db /volume1/docker/connection-monitor/data/connection-monitor.db.bak
docker compose start
```

For a source checkout:

```bash
cd /volume1/docker/connection-monitor/app
git pull
docker compose up -d --build
```

For a release image, update the image tag in `docker-compose.yml`, then recreate the service:

```bash
docker compose pull
docker compose up -d
```

Migrations run automatically at startup. Keep the database backup until you confirm the new version starts cleanly.

### Permissions

The container runs the app as a non-root user. The mounted data directory must be writable by the container.

If startup logs show `unable to open database file`, check that this directory exists:

```text
/volume1/docker/connection-monitor/data
```

Then fix write access from File Station permissions or SSH:

```bash
chmod 775 /volume1/docker/connection-monitor/data
```

### LAN-Only Synology Checklist

- Do not configure router port forwarding for port `8810`.
- Do not publish the app through QuickConnect.
- Do not add it to Synology Web Station.
- Allow port `8810` only from your LAN in Synology Firewall.
- For stricter binding, map the port to the NAS LAN IP only, for example `"192.168.1.10:8810:8080"`.
- Use a VPN such as Tailscale, WireGuard, or Synology VPN Server for remote access.

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
