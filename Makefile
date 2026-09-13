.PHONY: build build-endtoend test test-ci test-examples test-endtoend test-goldeneye start psql mysqlsh proto narsilc-dev narsilc-gen-json

build:
	go build ./...

install:
	go install ./...

test:
	go test ./...

test-managed:
	MYSQL_SERVER_URI="invalid" POSTGRESQL_SERVER_URI="postgres://postgres:mysecretpassword@localhost:5432/postgres" go test -v ./...

vet:
	go vet ./...

test-examples:
	go test --tags=examples ./...

build-endtoend:
	cd ./internal/endtoend/testdata && go build ./...

test-ci: test-examples build-endtoend vet

narsilc-dev:
	go build -o ~/bin/narsilc ./cmd/narsilc/

goldeneye:
	cd ./internal/goldeneye && go build -o ~/bin/goldeneye ./cmd/goldeneye

test-goldeneye:
	cd ./internal/goldeneye && go test ./...

narsilc-gen-json:
	go build -o ~/bin/narsilc-gen-json ./cmd/narsilc-gen-json

test-json-process-plugin:
	go build -o ~/bin/test-json-process-plugin ./scripts/test-json-process-plugin/

start:
	docker compose up -d

fmt:
	go fmt ./...

psql:
	PGPASSWORD=mysecretpassword psql --host=127.0.0.1 --port=5432 --username=postgres dinotest

mysqlsh:
	mysqlsh --sql --user root --password mysecretpassword --database dinotest 127.0.0.1:3306

proto:
	buf generate

remote-proto:
	protoc \
		--go_out=. --go_opt="Minternal/remote/gen.proto=github.com/mbvlabs/narsilc/internal/remote" --go_opt=module=github.com/mbvlabs/narsilc \
        --go-grpc_out=. --go-grpc_opt="Minternal/remote/gen.proto=github.com/mbvlabs/narsilc/internal/remote" --go-grpc_opt=module=github.com/mbvlabs/narsilc \
        internal/remote/gen.proto
