.PHONY: build dev-backend test dev-frontend

# Full production build: compile frontend then backend binary
build:
	cd web && pnpm install && pnpm build
	go build -o subhub ./cmd/subhub

# Backend-only dev build: create a placeholder web/dist/index.html so
# go:embed compiles without a full frontend build, then compile the binary.
dev-backend:
	mkdir -p web/dist
	echo '<html><body><div id="root">dev placeholder</div></body></html>' > web/dist/index.html
	go build -o subhub ./cmd/subhub

# Run all Go tests
test:
	go test -count=1 ./...

# Start the Vite dev server (hot-reload frontend)
dev-frontend:
	cd web && pnpm dev
