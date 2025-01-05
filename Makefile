.PHONY: build
# Define the PATH variable
PATH := $(PATH):$(shell go env GOPATH)/bin

lint:
	golangci-lint run ./...

dep:
	go mod tidy

gen:
	protoc -I=pkg/proto/ --go-grpc_out=pkg/proto --plugin=grpc  --go_out=pkg/proto pkg/proto/token_service.proto

build: gen
	go build -o build/dunebot main.go

test:
	@go clean -testcache
	@go test -count 3 -v -failfast -cover -coverprofile coverage.txt.tmp -race ./...
	@echo
	@cat coverage.txt.tmp | grep -v .pb.go > coverage.txt
	rm coverage.txt.tmp
	@go tool cover -func coverage.txt

run: build
	@env $$(cat .env) ./build/dunebot app

build-image: gen
	docker build -t dunebot .

up:
	teller env | envtor | docker-compose -f - up -d --build

down:
	teller env | envtor | docker-compose -f - down

run-dispatch: build
	teller run -- ./build/dunebot dispatch --dry

run-app: build
	teller run -- ./build/dunebot app

run-token: build
	teller run -- ./build/dunebot token service
