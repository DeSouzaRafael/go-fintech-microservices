SERVICES := identity wallet transaction fraud notification query gateway

.PHONY: build test test-integration lint infra-up infra-down proto tidy

build:
	@for svc in $(SERVICES); do \
		echo "building $$svc..."; \
		go build ./services/$$svc/...; \
	done

test:
	go test ./... -count=1 -race -timeout 60s

test-integration:
	go test ./tests/integration/... -count=1 -timeout 120s -tags integration

lint:
	golangci-lint run ./...

infra-up:
	docker compose -f deploy/docker-compose.yml up -d

infra-down:
	docker compose -f deploy/docker-compose.yml down

infra-logs:
	docker compose -f deploy/docker-compose.yml logs -f

proto:
	@which protoc > /dev/null || (echo "protoc not installed" && exit 1)
	protoc \
		--go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		--grpc-gateway_out=. --grpc-gateway_opt=paths=source_relative \
		-I api/proto \
		-I third_party/googleapis \
		api/proto/*.proto

tidy:
	go mod tidy

.DEFAULT_GOAL := build
