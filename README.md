# HomeLink Monitor

HomeLink Monitor is a self-hosted dashboard for monitoring your home internet connection from a NAS or any Docker host on your LAN.

It tracks latency, DNS health, speed tests, outages, and connection status over time. The app is local-first, stores data in SQLite, and is designed to run well on Synology Container Manager.

## Features

- Connection status dashboard.
- Latency checks against configurable targets.
- DNS resolution checks against configurable domains.
- Manual and scheduled speed tests through the Ookla `speedtest` CLI.
- Outage detection.
- Local users and login.
- SQLite persistence.
- Docker Compose deployment.
- Docker-backed Synology SPK packaging.

## What It Is Not

- It is not a public monitoring service.
- It does not configure router port forwarding, QuickConnect, or reverse proxies.
- It does not require Synology Web Station.
- It is not a replacement for Prometheus or full observability stacks.

## Repository Layout

```text
apps/web/       React, TypeScript, Vite, Ant Design
services/api/   Go API, SQLite, monitoring jobs
synology/       DSM SPK packaging files
Dockerfile      Production image
docker-compose.yml
Makefile
```

Future clients can be added under `apps/`, for example `apps/ios` or `apps/android`.

## Quick Start With Docker Compose

Create a local env file:

```bash
cp .env.example .env
```

Edit `.env` and change the default admin password.

Start the app:

```bash
make run
```

Equivalent command:

```bash
docker compose -f docker-compose.yml up -d --build
```

Open:

```text
http://localhost:8810
```

The default local database is stored in `./data`.

## Synology Container Manager

For a normal Compose deployment on Synology, set `DATA_DIR` to a persistent NAS directory:

```bash
DATA_DIR=/volume1/docker/homelink-monitor/data docker compose up -d --build
```

Then open:

```text
http://SYNOLOGY_IP:8810
```

Keep the app LAN-only:

- Do not add router port forwarding.
- Do not expose it through QuickConnect.
- Limit the port to LAN clients in DSM Firewall.
- Use a VPN for remote access.

## Synology Package Center SPK

Build the Docker-backed SPK:

```bash
make spk
```

Install it through:

```text
DSM Package Center -> Manual Install -> dist/homelink-monitor-<version>.spk
```

The install wizard asks for:

- Admin username.
- Admin password and confirmation.
- HTTP port, default `8810`.
- DSM notifications, enabled by default.

Detailed SPK documentation is in [synology/README.SYNOLOGY.md](synology/README.SYNOLOGY.md).

## Local Development

Backend:

```bash
cd services/api
go mod download
DB_PATH=./homelink-monitor.db STATIC_PATH=../../apps/web/dist go run ./cmd/server
```

Frontend:

```bash
cd apps/web
npm install
npm run dev
```

Open:

```text
http://localhost:5173
```

The Vite dev server proxies API requests to `http://localhost:8080`.

## Tests

Backend:

```bash
cd services/api
go test ./...
```

Frontend:

```bash
cd apps/web
npm test
npm run build
```

SPK packaging validation:

```bash
make spk-validate
```

## Speed Tests

The default command is:

```text
speedtest --accept-license --accept-gdpr --format=json
```

The Docker image installs the official Ookla CLI by default. If the command fails or is missing, the app stores the failure and shows the error in the UI.

To build without bundling the Ookla CLI:

```bash
docker build --build-arg INSTALL_SPEEDTEST=false -t homelink-monitor .
```

## Configuration

Important environment variables:

```text
ADMIN_USERNAME
ADMIN_PASSWORD
DB_PATH
APP_ENV
STATIC_PATH
EVENT_OUTBOX_DIR
DSM_NOTIFICATIONS_ENABLED
```

`ADMIN_USERNAME` and `ADMIN_PASSWORD` are only used to create the first admin user when the database has no users. After that, manage users in the UI.

## Backups

Back up the SQLite database from the data directory.

For Docker Compose on Synology:

```text
/volume1/docker/homelink-monitor/data/homelink-monitor.db
```

For the SPK:

```text
/var/packages/homelink-monitor/var/data/homelink-monitor.db
```

Stop the container before copying the database, or use SQLite online backup tooling.

## CI/CD

GitHub Actions validate pull requests and pushes to `main`:

- Go formatting, vet, and tests.
- Web tests and production build.
- Docker Compose validation and image build.

Dependabot PRs are configured to auto-merge after the validation workflow passes.

Container images are published to GitHub Container Registry when a GitHub Release is published.

Release PR automation is intentionally not enabled because GitHub repository settings may block `GITHUB_TOKEN` from creating pull requests. Create GitHub releases manually, or enable “Allow GitHub Actions to create and approve pull requests” before adding Release Please back.

## Troubleshooting

Docker Compose not found:

```text
Install Docker Compose or Synology Container Manager.
```

Speed test failed:

```text
Check Settings -> speed test command and verify the CLI exists inside the container.
```

Cannot open the UI:

```text
Confirm host port 8810 maps to container port 8080 and DSM Firewall allows LAN access.
```

Database errors:

```text
Confirm the mounted data directory exists and is writable.
```
