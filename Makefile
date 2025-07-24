all: build

build:
	go build -gcflags='all=-N -l'

install:
	go install

run: build
	./cinnamon

# Run `make run-debug` in one terminal tab and `make print-log` in another to view the program and its log output side by side
run-debug:
	go run main.go -debug

print-log:
	go run main.go --logs

unit-test:
	go test ./... -short

generate:
	go generate ./...

format:
	golangci-lint fmt

.PHONY: lint
lint:
	golangci-lint run

vendor:
	go mod vendor && go mod tidy
