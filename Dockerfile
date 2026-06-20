FROM node:24-alpine AS web
WORKDIR /workspace/apps/web
COPY apps/web/package.json apps/web/package-lock.json apps/web/tsconfig.json apps/web/vite.config.ts apps/web/index.html ./
COPY apps/web/src ./src
RUN npm ci
RUN npm run build

FROM golang:1.26-alpine AS api
WORKDIR /workspace/services/api
COPY services/api/go.mod services/api/go.sum ./
RUN go mod download
COPY services/api ./
RUN CGO_ENABLED=0 GOOS=linux go build -o /connection-monitor ./cmd/server

FROM debian:bookworm-slim
ARG INSTALL_SPEEDTEST=true
ENV DEBIAN_FRONTEND=noninteractive
RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl gnupg \
    && if [ "$INSTALL_SPEEDTEST" = "true" ]; then \
      curl -fsSL https://packagecloud.io/ookla/speedtest-cli/gpgkey | gpg --dearmor -o /usr/share/keyrings/ookla-speedtest-archive-keyring.gpg \
      && echo "deb [signed-by=/usr/share/keyrings/ookla-speedtest-archive-keyring.gpg] https://packagecloud.io/ookla/speedtest-cli/debian/ bookworm main" > /etc/apt/sources.list.d/ookla-speedtest.list \
      && apt-get update \
      && apt-get install -y --no-install-recommends speedtest; \
    fi \
    && apt-get purge -y --auto-remove curl gnupg \
    && rm -rf /var/lib/apt/lists/* \
    && useradd --system --home-dir /nonexistent --shell /usr/sbin/nologin appuser \
    && mkdir -p /data /app/apps/web/dist \
    && chown -R appuser:appuser /data /app
USER appuser
WORKDIR /app
COPY --from=api /connection-monitor /app/connection-monitor
COPY --from=web /workspace/apps/web/dist /app/apps/web/dist
ENV ADDR=:8080
ENV DB_PATH=/data/connection-monitor.db
ENV STATIC_PATH=/app/apps/web/dist
EXPOSE 8080
CMD ["/app/connection-monitor"]
