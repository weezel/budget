# CGO_ENABLED=0 == static by default
GO		?= go
DOCKER		?= docker
# -s removes symbol table and -ldflags -w debugging symbols
LDFLAGS		?= -asmflags -trimpath -ldflags "-s -w"
GOARCH		?= amd64
BINARY		?= budget
CGO_ENABLED	?= 1

PSQL_CLIENT	?= psql
POSTGRES_VER	?= 14.4-alpine
DB_HOST		?= $(shell awk -F '=' '/^DB_HOST/ { print $$NF }' .env)
DB_PORT		?= $(shell awk -F '=' '/^DB_PORT/ { print $$NF }' .env)
DB_NAME		?= $(shell awk -F '=' '/^DB_NAME/ { print $$NF }' .env)
DB_USERNAME	?= $(shell awk -F '=' '/^DB_USERNAME/ { print $$NF }' .env)
DB_PASSWORD	?= $(shell awk -F '=' '/^DB_PASSWORD/ { print $$NF }' .env)


.PHONY: all analysis obsd test

build: test lint
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=$(GOARCH) \
	     $(GO) build $(LDFLAGS) -o $(BINARY)_linux_$(GOARCH) \
	     cmd/telegrambot/main.go

clean:
	rm -f budget budget_linux_amd64 budget_openbsd_amd64

lint:
	golangci-lint run ./...

escape-analysis:
	$(GO) build -gcflags="-m" 2>&1

docker-build:
	$(DOCKER) build --rm --target app -t budget-test .

docker-run:
	docker run --rm -v $(shell pwd):/app/config budget-test &

dev-migrations:
	go run cmd/dbmigrate/main.go

create-db:
	@$(PSQL_CLIENT) postgresql://$(DB_USERNAME):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/ \
		-q -c "CREATE DATABASE $(DB_NAME) OWNER postgres ENCODING UTF8;"

dev-db:
	@$(DOCKER) run \
		--name budgetdb_dev \
		--rm \
		-e POSTGRES_PASSWORD=$(DB_PASSWORD) \
		-p $(DB_PORT):$(DB_PORT) \
		-d postgres:$(POSTGRES_VER)
	@sleep 1

start-devdb: dev-db create-db dev-migrations

stop-devdb:
	@$(DOCKER) stop budgetdb_dev

obsd:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=openbsd GOARCH=$(GOARCH) \
	     $(GO) build $(LDFLAGS) -o $(BINARY)_openbsd_$(GOARCH)

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test:
	go test ./...

.PHONY: sqlc
sqlc:
	sqlc generate
