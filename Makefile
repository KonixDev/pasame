VERSION ?= dev
LDFLAGS := -s -w -X main.version=$(VERSION)
TARGETS := windows/amd64 windows/arm64 darwin/amd64 darwin/arm64 linux/amd64 linux/arm64 linux/arm

.PHONY: test build cross run
test:
	CGO_ENABLED=1 go test -race ./...
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="$(LDFLAGS)" -o pasame ./cmd/pasame
cross:
	@for t in $(TARGETS); do \
	  os=$${t%/*}; arch=$${t#*/}; \
	  echo "==> $$os/$$arch"; \
	  CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch GOARM=7 go build -trimpath -ldflags="$(LDFLAGS)" -o /dev/null ./... || exit 1; \
	done
run: build
	./pasame
