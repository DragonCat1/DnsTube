.PHONY: tidy build build-web test dev-api dev-web bench-dns

BINARY := dnstube
GOFLAGS := -trimpath

tidy:
	go mod tidy

build-web:
	cd web && npm ci && npm run build
	rm -rf cmd/dnstube/web/dist
	cp -R web/dist cmd/dnstube/web/dist

build-api:
	go build $(GOFLAGS) -o $(BINARY) ./cmd/dnstube

build: build-web build-api

test:
	go test ./...

bench-dns:
	go build $(GOFLAGS) -o dnsbench ./cmd/dnsbench

load-dev-env:
	@echo "Loading environment variables..."
	@cat .env.local | grep -E '^[^#]+' | while IFS='=' read -r key value; do \
		if [ -n "$$key" ]; then \
			export "$$key=$$value"; \
		fi; \
	done; \
	echo "Environment variables loaded:"; \
	cat .env.local | grep -E '^[^#]+' | while IFS='=' read -r key value; do \
		if [ -n "$$key" ]; then \
			echo "  $$key=$$(eval echo "$$value")"; \
		fi; \
	done

dev-api: build-api
	@echo "Starting with environment variables..."
	@env $$(cat .env.local | grep -E '^[^#]+' | xargs) ./$(BINARY)

dev-web:
	cd web && npm run dev
