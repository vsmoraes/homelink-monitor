# HomeLink Monitor

<p align="center">
  <img src="img/logo.png" alt="HomeLink Monitor logo" width="140" />
</p>

<p align="center">
  <a href="https://github.com/vsmoraes/homelink-monitor/actions/workflows/ci.yml"><img alt="CI" src="https://img.shields.io/github/actions/workflow/status/vsmoraes/homelink-monitor/ci.yml?branch=main&label=ci&style=flat-square"></a>
  <a href="https://github.com/vsmoraes/homelink-monitor/actions/workflows/release-spk.yml"><img alt="SPK release" src="https://img.shields.io/github/actions/workflow/status/vsmoraes/homelink-monitor/release-spk.yml?label=spk&style=flat-square"></a>
  <a href="https://github.com/vsmoraes/homelink-monitor/pkgs/container/homelink-monitor"><img alt="GHCR" src="https://img.shields.io/badge/ghcr-container-0f172a?style=flat-square&logo=github"></a>
  <a href="https://github.com/vsmoraes/homelink-monitor/releases"><img alt="Latest release" src="https://img.shields.io/github/v/release/vsmoraes/homelink-monitor?style=flat-square"></a>
  <img alt="Go" src="https://img.shields.io/badge/go-1.25-00ADD8?style=flat-square&logo=go&logoColor=white">
  <img alt="React" src="https://img.shields.io/badge/react-19-61DAFB?style=flat-square&logo=react&logoColor=111">
  <img alt="TypeScript" src="https://img.shields.io/badge/typescript-6-3178C6?style=flat-square&logo=typescript&logoColor=white">
  <img alt="SQLite" src="https://img.shields.io/badge/sqlite-local--first-003B57?style=flat-square&logo=sqlite&logoColor=white">
  <img alt="Docker" src="https://img.shields.io/badge/docker-ready-2496ED?style=flat-square&logo=docker&logoColor=white">
  <img alt="Synology" src="https://img.shields.io/badge/synology-spk-B5B5B6?style=flat-square">
</p>

Self-hosted LAN dashboard for monitoring home internet health from a NAS or any Docker host.

It tracks latency, DNS health, speed tests, outages, and connection status over time. Data stays local in SQLite.

## Features

- Synology-like connection dashboard.
- Latency checks per target.
- DNS checks per domain.
- Optional TP-Link Archer router client traffic collection.
- Manual and scheduled Ookla speed tests.
- Outage detection.
- Local users and login.
- SQLite persistence.
- Docker Compose deployment.
- Docker-backed Synology DSM 7 SPK.

## Not This

- Not a public monitoring service.
- Not a router, QuickConnect, or reverse proxy configurator.
- Not a Prometheus replacement.
- Not intended to be exposed directly to the internet.

## Quick Start

```bash
cp .env.example .env
$EDITOR .env
make run
```

Open:

```text
http://localhost:8810
```

Set a real admin password in `.env`; Compose refuses to start without credentials.

## Synology

### Docker Compose

```bash
DATA_DIR=/volume1/docker/homelink-monitor/data docker compose up -d --build
```

Open:

```text
http://SYNOLOGY_IP:8810
```

### Package Center SPK

Download the latest `.spk` from [GitHub Releases](https://github.com/vsmoraes/homelink-monitor/releases), then install it:

```text
DSM Package Center -> Manual Install -> homelink-monitor-<version>.spk
```

The wizard asks for admin username, admin password, HTTP port, and DSM notification preference.

Detailed DSM notes: [synology/README.SYNOLOGY.md](synology/README.SYNOLOGY.md).

## Development

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

Tests:

```bash
cd services/api && go test ./...
cd apps/web && npm test && npm run build
```

## Build The SPK

```bash
make spk
```

With an explicit version:

```bash
make spk VERSION=0.1.0-0010
```

Output:

```text
dist/homelink-monitor-<version>.spk
```

## Releases

SPKs are built and published automatically by GitHub Actions.

Create a version tag:

```bash
git tag v0.1.0-0010
git push origin v0.1.0-0010
```

The release workflow:

- builds the SPK with `make spk VERSION=0.1.0-0010`
- verifies the archive structure and package icons
- creates or updates the GitHub Release
- uploads `dist/homelink-monitor-0.1.0-0010.spk`

Container images are published to GitHub Container Registry when a GitHub Release is published.

## Data And Backups

SQLite database locations:

```text
Docker Compose: ./data or $DATA_DIR
Synology SPK:   /var/packages/homelink-monitor/var/data/homelink-monitor.db
```

Stop the container before copying the database, or use SQLite online backup tooling.

## LAN-Only Notes

- Do not add router port forwarding.
- Do not expose it through QuickConnect.
- Limit access with DSM Firewall.
- Use a VPN for remote access.

## Production Notes

- `ADMIN_USERNAME` and `ADMIN_PASSWORD` are required when the first user is created.
- Set `AUTH_COOKIE_SECURE=true` only when serving the app over HTTPS.
- Keep `/data` backed up; it contains the SQLite database.
- The speed test command is configurable by admins and runs inside the unprivileged app container, without a shell.
- The app rejects cross-origin browser writes and caps API request bodies.

## Troubleshooting

Docker Compose missing:

```text
Install Docker Compose or Synology Container Manager.
```

Speed test failed:

```text
Check Settings -> speed test command and verify the CLI exists inside the container.
```

TP-Link router traffic is empty:

```text
Enable Settings -> TP-Link router traffic, use the local router password, and confirm the router web API returns client traffic fields.
```

Cannot open the UI:

```text
Confirm host port 8810 maps to container port 8080 and your firewall allows LAN access.
```

## TP-Link Router Traffic

HomeLink Monitor can poll local TP-Link Archer-style router APIs for connected clients and client traffic rates.

The implementation is capability-based:

- It probes known TP-Link web API endpoints used by Archer C6U/C80/AX/BE-family community tooling.
- It stores connected client traffic over time when the router returns usable client traffic fields.
- It exposes download/upload only when separate values are present.
- If the router exposes only total traffic, only total traffic is stored/displayed.
- If the router shape is unsupported, the UI reports the unavailable capability instead of guessing.

References used for the first adapter:

- [`AlexandrErohin/TP-Link-Archer-C6U`](https://github.com/AlexandrErohin/TP-Link-Archer-C6U)
- [`loliver1823/tplink-archer-be400-cli`](https://github.com/loliver1823/tplink-archer-be400-cli)
- [`hertzg/node-tplink-api`](https://github.com/hertzg/node-tplink-api)
