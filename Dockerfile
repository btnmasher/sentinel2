FROM golang:1.25-bookworm AS toolchain
ARG BUN_VERSION=1.3.8
WORKDIR /app
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates curl git bash unzip libx11-dev \
  && curl -fsSL https://bun.sh/install | bash -s -- bun-v${BUN_VERSION} \
  && ln -s /root/.bun/bin/bun /usr/local/bin/bun \
  && rm -rf /var/lib/apt/lists/*

FROM toolchain AS deps
WORKDIR /app
RUN mkdir -p /root/.cache/go-build /root/.cache/go-tmp /go/pkg/mod /app/.tmp/bun /app/.tmp/bun-install
COPY backend/go.mod backend/go.sum ./backend/
RUN cd backend && go mod download
COPY frontend/package.json frontend/bun.lockb* ./frontend/
RUN cd frontend \
  && BUN_TMPDIR=/app/.tmp/bun BUN_INSTALL=/app/.tmp/bun-install bun install --frozen-lockfile

FROM toolchain AS build
WORKDIR /app
ARG BUILD_VERSION=""
ENV BUILD_VERSION=${BUILD_VERSION}
COPY --from=deps /go/pkg/mod /go/pkg/mod
COPY --from=deps /root/.cache/go-build /root/.cache/go-build
COPY --from=deps /root/.cache/go-tmp /root/.cache/go-tmp
COPY --from=deps /app/frontend/node_modules /app/frontend/node_modules
COPY --from=deps /app/.tmp /app/.tmp
COPY . .
RUN cd frontend \
  && BUN_TMPDIR=/app/.tmp/bun BUN_INSTALL=/app/.tmp/bun-install bun run build \
  && cd /app \
  && rm -rf /app/backend/internal/web/dist \
  && mkdir -p /app/backend/internal/web/dist /app/bin \
  && cp -R /app/frontend/dist/. /app/backend/internal/web/dist/ \
  && cd backend \
  && DERIVED_BUILD_VERSION="${BUILD_VERSION}" \
  && if [ -z "${DERIVED_BUILD_VERSION}" ] && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then \
  BRANCH="$(git rev-parse --abbrev-ref HEAD 2>/dev/null || true)"; \
  EXACT_TAG="$(git describe --tags --match 'v[0-9]*' --exact-match 2>/dev/null || true)"; \
  LATEST_TAG="$(git describe --tags --match 'v[0-9]*' --abbrev=0 2>/dev/null || true)"; \
  SHORT_SHA="$(git rev-parse --short HEAD 2>/dev/null || true)"; \
  if [ -n "${EXACT_TAG}" ]; then \
  DERIVED_BUILD_VERSION="${EXACT_TAG}"; \
  else \
  if [ -n "${LATEST_TAG}" ]; then DERIVED_BUILD_VERSION="${LATEST_TAG}"; else DERIVED_BUILD_VERSION="v0.0.0"; fi; \
  if { [ "${BRANCH}" = "main" ] || [ "${BRANCH}" = "HEAD" ]; } && [ -n "${SHORT_SHA}" ]; then DERIVED_BUILD_VERSION="${DERIVED_BUILD_VERSION}-${SHORT_SHA}"; fi; \
  fi; \
  if [ -n "${DERIVED_BUILD_VERSION}" ]; then \
  if [ -n "$(git status --porcelain 2>/dev/null || true)" ]; then DERIVED_BUILD_VERSION="${DERIVED_BUILD_VERSION}-dev"; fi; \
  if [ -n "${BRANCH}" ] && [ "${BRANCH}" != "HEAD" ] && [ "${BRANCH}" != "main" ]; then \
  SAFE_BRANCH="$(printf '%s' "${BRANCH}" | sed 's/[^A-Za-z0-9._-]/-/g')"; \
  DERIVED_BUILD_VERSION="${DERIVED_BUILD_VERSION}-branch-${SAFE_BRANCH}"; \
  fi; \
  fi; \
  fi \
  && GOCACHE=/root/.cache/go-build GOTMPDIR=/root/.cache/go-tmp go build -tags embed_frontend -ldflags "-X main.BuildVersion=${DERIVED_BUILD_VERSION}" -o /app/bin/sentinel2-server ./main.go

FROM debian:bookworm-slim
RUN apt-get update \
  && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*
WORKDIR /app
COPY --from=build /app/bin/sentinel2-server /app/sentinel2-server
VOLUME ["/app/pb_data"]
EXPOSE 8090
ENV PB_DATA=/app/pb_data
CMD ["/app/sentinel2-server", "serve", "--http=0.0.0.0:8090"]
