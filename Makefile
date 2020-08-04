# CGO_ENABLED=0 == static by default
GO		 = go
# -s removes symbol table and -ldflags -w debugging symbols
LDFLAGS		 = -trimpath -ldflags "-s -w"
GOARCH		 = amd64
BINARY		 = budget
# XX For future release -gcflags=all=-d=checkptr -run=Rights syscall

.PHONY: all analysis obsd test

default:
	GOOS=linux GOARCH=$(GOARCH) CGO_ENABLED=1 \
	     $(GO) build $(LDFLAGS) -o $(BINARY)_linux_$(GOARCH)
debug:
	CGO_ENABLED=1 $(GO) build $(LDFLAGS)
obsd:
	GOOS=openbsd GOARCH=$(GOARCH) CGO_ENABLED=1 \
	     $(GO) build $(LDFLAGS) -o $(BINARY)_openbsd_$(GOARCH)
test:
	go test ./...

clean:
	rm -f budget budget_linux_amd64 budget_openbsd_amd64

