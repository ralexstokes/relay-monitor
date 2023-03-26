VERSION ?= $(shell git describe --tags --always --dirty="-dev")

all: clean build

v:
	@echo "Version: ${VERSION}"

clean:
	git clean -fdx

build:
	go build -ldflags "-X cmd.Version=${VERSION} -X main.Version=${VERSION}" -o ./bin/relay-monitor ./cmd/relay-monitor

run:
	./bin/relay-monitor

test:
	go test ./...

test-race:
	go test -race ./...

lint:
	gofmt -d -s .
	gofumpt -d -extra .
	go vet ./...
	staticcheck ./...
	golangci-lint run

gofumpt:
	gofumpt -l -w -extra .

test-coverage:
	go test -race -v -covermode=atomic -coverprofile=coverage.out ./...
	go tool cover -func coverage.out

cover-html:
	go test -coverprofile=/tmp/relay-monitor.cover.tmp ./...
	go tool cover -html=/tmp/relay-monitor.cover.tmp
	unlink /tmp/relay-monitor.cover.tmp

docker-image:
	DOCKER_BUILDKIT=1 docker build --build-arg VERSION=${VERSION} . -t ralexstokes/relay-monitor

docker-image-amd:
	DOCKER_BUILDKIT=1 docker build --platform linux/amd64 --build-arg VERSION=${VERSION} . -t ralexstokes/relay-monitor
