# HomeLink Monitor Synology SPK

This directory contains the source files used to build a Docker-backed Synology DSM package for HomeLink Monitor.

The SPK does not install the Go binary directly into DSM. It installs a package payload containing a Synology-specific Docker Compose file, a clean copy of the existing application source used as the Docker build context, lifecycle scripts, and a notification helper. Runtime execution stays inside Docker / Container Manager.

## Requirements

- Synology DSM 7.x.
- Synology Container Manager / Docker installed and running.
- Docker Compose support through either `docker compose` or `docker-compose`.
- LAN access to the NAS on the configured HTTP port.
- Internet access during first start if the image must be built on the NAS.

The exact DSM package dependency key for Container Manager can vary by DSM and package generation. The SPK therefore does not rely only on INFO metadata for dependency enforcement. The lifecycle scripts check for Docker and Compose at install/start time and fail with a clear message if unavailable.

## Build The SPK

From the repository root:

```bash
make spk
```

The generated package is written to:

```text
dist/homelink-monitor-<version>.spk
```

The build creates temporary package layout files under `build/`. Those files are generated artifacts and should not be committed.

## Manual Install

1. Open DSM Package Center.
2. Choose Manual Install.
3. Select `dist/homelink-monitor-<version>.spk`.
4. Complete the install wizard.
5. Start the package from Package Center.
6. Open `http://SYNOLOGY_IP:8810`, or the port selected in the wizard.

## Install Wizard

The wizard asks for:

- Admin username.
- Admin password.
- Password confirmation.
- HTTP port, default `8810`.
- Whether DSM notifications are enabled, default enabled.

Wizard values are written by `postinst` to:

```text
/var/packages/homelink-monitor/var/config/app.env
```

`app.env` is created with mode `600` because it contains the initial admin password. The password is used only to seed the first app user when the SQLite database has no users yet. After first boot, manage users inside the HomeLink Monitor UI.

## Runtime Layout

DSM package target:

```text
/var/packages/homelink-monitor/target
```

Runtime state:

```text
/var/packages/homelink-monitor/var
/var/packages/homelink-monitor/var/data
/var/packages/homelink-monitor/var/config
/var/packages/homelink-monitor/var/events
/var/packages/homelink-monitor/var/events/outbox
/var/packages/homelink-monitor/var/events/processed
/var/packages/homelink-monitor/var/events/failed
/var/packages/homelink-monitor/var/logs
```

SQLite database:

```text
/var/packages/homelink-monitor/var/data/homelink-monitor.db
```

Configuration:

```text
/var/packages/homelink-monitor/var/config/app.env
```

Notification helper log:

```text
/var/packages/homelink-monitor/var/logs/notification-helper.log
```

## Start, Stop, Status, And Logs

Package Center calls:

```text
/var/packages/homelink-monitor/scripts/start-stop-status start
/var/packages/homelink-monitor/scripts/start-stop-status stop
/var/packages/homelink-monitor/scripts/start-stop-status status
/var/packages/homelink-monitor/scripts/start-stop-status log
```

`start` runs Docker Compose from:

```text
/var/packages/homelink-monitor/target
```

`stop` runs Compose `down --remove-orphans`, which stops and removes the Compose services while preserving volumes and persistent package data.

`status` checks whether the main container named `homelink-monitor` is running.

To verify manually:

```bash
docker ps --filter name=homelink-monitor
docker logs homelink-monitor
```

If Synology uses the legacy Compose command, use `docker-compose` where needed.

## Docker Compose

The SPK payload includes a Synology-specific Compose file generated from:

```text
synology/templates/docker-compose.yml
```

It builds the existing application image from the source snapshot embedded in the SPK payload:

```text
/var/packages/homelink-monitor/target/source
```

This intentionally reuses the repository's existing root Dockerfile and build context instead of inventing a separate app stack.

The container mounts:

```text
/var/packages/homelink-monitor/var/data:/data
/var/packages/homelink-monitor/var/config:/config:ro
/var/packages/homelink-monitor/var/events/outbox:/events/outbox
```

The app reads environment variables from:

```text
/var/packages/homelink-monitor/var/config/app.env
```

## Notifications

The app writes notification event JSON files to:

```text
/events/outbox
```

That path is mounted from:

```text
/var/packages/homelink-monitor/var/events/outbox
```

Each event has this shape:

```json
{
  "id": "unique-event-id",
  "severity": "info|warning|critical|recovery",
  "metric": "latency|dns|speedtest|outage|connection|service",
  "title": "Short notification title",
  "message": "Human-readable notification message",
  "timestamp": "ISO-8601 UTC timestamp"
}
```

The host-side helper `notification-helper.sh` polls the outbox. Successfully processed events move to `processed`; failed or invalid events move to `failed`.

DSM notification delivery is isolated in the `send_dsm_notification` function. It tries known DSM notification commands such as `synodsmnotify` and falls back to logging if those tools are unavailable. This behavior must be verified on a real DSM 7.x NAS because Synology notification tooling can vary by DSM release and permissions.

## Troubleshooting

Docker not found:

```text
HomeLink Monitor requires Synology Container Manager / Docker to be installed and running.
```

Install or start Synology Container Manager, then retry installation or start the package.

Docker Compose not found:

```text
Docker Compose was not found. Install or enable Synology Container Manager / Docker Compose support.
```

Check whether the NAS provides `docker compose` or `docker-compose`.

Container is not running:

```bash
docker ps --filter name=homelink-monitor
docker logs homelink-monitor
```

Package helper log:

```text
/var/packages/homelink-monitor/var/logs/notification-helper.log
```

Database or permission errors:

Verify that the package var directory exists and is writable:

```text
/var/packages/homelink-monitor/var/data
```

## Uninstall

The package stop script stops the Compose stack and notification helper. Uninstall cleanup removes helper PID files but intentionally leaves persistent package data under:

```text
/var/packages/homelink-monitor/var
```

Delete that directory manually only when you want to remove the SQLite database and package configuration.

## Security And LAN-Only Notes

- The app remains containerized.
- The app container does not mount the Docker socket.
- The app container does not receive DSM admin credentials.
- The SPK does not create reverse proxy rules.
- The SPK does not configure QuickConnect.
- Only the configured HTTP port is exposed.
- Keep DSM Firewall limited to LAN clients for the selected port.
- Use a VPN for remote access instead of exposing the package publicly.

## DSM Wizard Assumptions

The wizard file uses a best-known DSM 7 `WIZARD_UIFILES/install_uifile` JSON structure with keyed fields. The `postinst` script reads those values from environment variables using both lower-case and upper-case key variants. This is intentionally isolated in `value_from_wizard` so it can be adjusted after testing on a real NAS if a DSM build exposes wizard values differently.
