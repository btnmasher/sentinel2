FROM golang:1.25-bullseye AS uploader
WORKDIR /app/uploader
RUN apt-get update \
  && apt-get install -y --no-install-recommends zip \
  && rm -rf /var/lib/apt/lists/*
COPY uploader/go.mod uploader/go.sum ./
RUN go mod download
COPY uploader/ ./
RUN mkdir -p /out/downloads \
  && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /tmp/sentinel2-uploader . \
  && zip -j /out/downloads/sentinel2-uploader-linux.zip /tmp/sentinel2-uploader \
  && CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -o /tmp/sentinel2-uploader.exe . \
  && zip -j /out/downloads/sentinel2-uploader-windows.zip /tmp/sentinel2-uploader.exe \
  && CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -o /tmp/sentinel2-uploader . \
  && zip -j /out/downloads/sentinel2-uploader-macos.zip /tmp/sentinel2-uploader

FROM oven/bun:1.3.8 AS frontend
WORKDIR /app/frontend
COPY frontend/package.json frontend/bun.lockb* ./
RUN bun install --frozen-lockfile
COPY frontend/ .
COPY --from=uploader /out/downloads /app/frontend/public/downloads
RUN bun run build

FROM golang:1.25-bullseye AS backend
WORKDIR /app
ARG BUILD_VERSION=""
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download
COPY backend/ ./backend/
COPY --from=frontend /app/frontend/dist /app/backend/internal/web/dist
RUN cd backend && \
  LDFLAGS="" && \
  if [ -n "$BUILD_VERSION" ]; then LDFLAGS="-X main.BuildVersion=$BUILD_VERSION"; fi && \
  if [ -n "$LDFLAGS" ]; then \
  go build -tags embed_frontend -ldflags "$LDFLAGS" -o /app/bin/sentinel2-server ./main.go; \
  else \
  go build -tags embed_frontend -o /app/bin/sentinel2-server ./main.go; \
  fi

FROM debian:bookworm-slim
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=backend /app/bin/sentinel2-server /app/sentinel2-server
VOLUME ["/app/pb_data"]
EXPOSE 8090
ENV PB_DATA=/app/pb_data
CMD ["/app/sentinel2-server"]
