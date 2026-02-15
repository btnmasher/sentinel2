FROM golang:1.25-bookworm AS toolchain
ARG BUN_VERSION=1.3.8
WORKDIR /app
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates curl git bash unzip \
  && curl -fsSL https://bun.sh/install | bash -s -- bun-v${BUN_VERSION} \
  && ln -s /root/.bun/bin/bun /usr/local/bin/bun \
  && go install github.com/go-task/task/v3/cmd/task@latest \
  && ln -s /go/bin/task /usr/local/bin/task \
  && rm -rf /var/lib/apt/lists/*

FROM toolchain AS deps
WORKDIR /app
RUN mkdir -p /root/.cache/go-build /go/pkg/mod /app/.tmp/bun /app/.tmp/bun-install
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download
COPY frontend/package.json frontend/bun.lockb* ./frontend/
RUN cd frontend \
  && BUN_TMPDIR=/app/.tmp/bun BUN_INSTALL=/app/.tmp/bun-install bun install --frozen-lockfile

FROM toolchain AS build
WORKDIR /app
ARG BUILD_VERSION=""
ENV BUILD_VERSION=${BUILD_VERSION}
ENV BUN_TMPDIR=/app/.tmp/bun
ENV BUN_INSTALL=/app/.tmp/bun-install
COPY --from=deps /go/pkg/mod /go/pkg/mod
COPY --from=deps /root/.cache/go-build /root/.cache/go-build
COPY --from=deps /app/frontend/node_modules /app/frontend/node_modules
COPY --from=deps /app/.tmp /app/.tmp
COPY . .
RUN rm -f /app/.tmp/bin/taskutil /app/.tmp/bin/taskutil.exe \
  && task build

FROM debian:bookworm-slim
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /app/bin/sentinel2-server /app/sentinel2-server
VOLUME ["/app/pb_data"]
EXPOSE 8090
ENV PB_DATA=/app/pb_data
CMD ["/app/sentinel2-server"]
