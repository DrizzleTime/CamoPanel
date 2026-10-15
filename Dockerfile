FROM node:24-bookworm-slim AS web
COPY --from=oven/bun:1.4.3 /usr/local/bin/bun /usr/local/bin/bun
WORKDIR /build
COPY package.json bun.lock ./
COPY frontend/package.json ./frontend/package.json
RUN bun install --frozen-lockfile
COPY frontend/ ./frontend/
RUN bun run --cwd frontend build

FROM golang:1.26-alpine AS api
WORKDIR /build/backend
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /camopanel ./cmd/server

FROM alpine:3.23
RUN apk add --no-cache ca-certificates && addgroup -S camo && adduser -S -G camo camo
WORKDIR /app
COPY --from=api /camopanel /app/camopanel
COPY --from=web /build/frontend/dist /app/frontend/dist
ENV CAMO_ADDR=0.0.0.0:8080 DOCKER_MODE=demo CAMO_WEB_DIR=/app/frontend/dist
EXPOSE 8080
USER camo
ENTRYPOINT ["/app/camopanel"]
