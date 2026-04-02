.PHONY: build build-web build-server test-server dev dev-server dev-web

build: build-web build-server

build-web:
	cd web && bun install && bun run build
	rm -rf server/internal/webui/dist
	mkdir -p server/internal/webui/dist
	cp -R web/dist/. server/internal/webui/dist/

build-server:
	cd server && go build ./cmd/camopanel

test-server:
	cd server && go test ./...

dev-server:
	cd server && go run ./cmd/camopanel

dev-web:
	cd web && bun run dev

dev:
	@trap 'kill 0' INT TERM EXIT; \
	(cd server && go run ./cmd/camopanel) & \
	(cd web && bun install && bun run dev) & \
	wait
