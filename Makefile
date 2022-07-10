# CGO_ENABLED=0 == static by default
GO		?= go
DOCKER		?= docker
# -s removes symbol table and -ldflags -w debugging symbols
LDFLAGS		?= -asmflags -trimpath -ldflags "-s -w"
GOARCH		?= amd64
BINARY		?= budget
CGO_ENABLED	?= 1

.PHONY: all analysis obsd test

build: test lint
	CGO_ENABLED=$(CGO_ENABLED) GOOS=linux GOARCH=$(GOARCH) \
	     $(GO) build $(LDFLAGS) -o $(BINARY)_linux_$(GOARCH)

lint:
	golangci-lint run ./...

escape-analysis:
	$(GO) build -gcflags="-m" 2>&1

docker-build:
	$(DOCKER) build --rm --target app -t budget-test .

docker-run:
	docker run --rm -v $(shell pwd):/app/config budget-test &

obsd:
	CGO_ENABLED=$(CGO_ENABLED) GOOS=openbsd GOARCH=$(GOARCH) \
	     $(GO) build $(LDFLAGS) -o $(BINARY)_openbsd_$(GOARCH)

coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test:
	go test ./...

clean:
	rm -f budget budget_linux_amd64 budget_openbsd_amd64

