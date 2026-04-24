SERVICES := identity wallet transaction fraud notification query gateway

DB_HOST   ?= localhost
DB_USER   ?= fintech
DB_PASS   ?= fintech

DB_PORT_identity     ?= 15432
DB_PORT_wallet       ?= 5433
DB_PORT_transaction  ?= 5434
DB_PORT_fraud        ?= 5435
DB_PORT_notification ?= 5436
DB_PORT_query        ?= 5437

db_url = "postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$(DB_PORT_$(1))/$(1)?sslmode=disable"

.PHONY: build test lint infra-up infra-down infra-logs proto tidy \
        migrate-install migrate-up migrate-down migrate-status migrate-create

build:
	@for svc in $(SERVICES); do \
		echo "building $$svc..."; \
		go build ./services/$$svc/...; \
	done

test:
	go test ./... -count=1 -race -timeout 120s

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
		--go_out=. --go_opt=module=github.com/DeSouzaRafael/go-fintech-microservices \
		--go-grpc_out=. --go-grpc_opt=module=github.com/DeSouzaRafael/go-fintech-microservices \
		--grpc-gateway_out=. --grpc-gateway_opt=module=github.com/DeSouzaRafael/go-fintech-microservices \
		-I api/proto \
		-I third_party/googleapis \
		api/proto/*.proto

tidy:
	go mod tidy

migrate-install:
	go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

migrate-up:
	@for svc in $(SERVICES); do \
		dir="services/$$svc/migrations"; \
		if [ -d "$$dir" ]; then \
			echo "▶ migrating $$svc..."; \
			migrate -path "$$dir" \
				-database "postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$$(eval echo \$$DB_PORT_$$svc)/$$svc?sslmode=disable" \
				up || exit 1; \
		fi \
	done

migrate-down:
	@[ -n "$(SVC)" ] || (echo "usage: make migrate-down SVC=<service>" && exit 1)
	@[ -d "services/$(SVC)/migrations" ] || (echo "no migrations for $(SVC)" && exit 1)
	migrate -path "services/$(SVC)/migrations" \
		-database $(call db_url,$(SVC)) \
		down 1

migrate-status:
	@for svc in $(SERVICES); do \
		dir="services/$$svc/migrations"; \
		if [ -d "$$dir" ]; then \
			echo "▶ $$svc:"; \
			migrate -path "$$dir" \
				-database "postgres://$(DB_USER):$(DB_PASS)@$(DB_HOST):$$(eval echo \$$DB_PORT_$$svc)/$$svc?sslmode=disable" \
				version 2>&1 || true; \
		fi \
	done

migrate-create:
	@[ -n "$(SVC)" ] || (echo "usage: make migrate-create SVC=<service> NAME=<name>" && exit 1)
	@[ -n "$(NAME)" ] || (echo "usage: make migrate-create SVC=<service> NAME=<name>" && exit 1)
	@mkdir -p services/$(SVC)/migrations
	migrate create -ext sql -dir services/$(SVC)/migrations -seq $(NAME)

.DEFAULT_GOAL := build
