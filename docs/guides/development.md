# Developing narsilc

## Building

For local development, install `narsilc` under an alias. We suggest `narsilc-dev`.

```
go build -o ~/go/bin/narsilc-dev ./cmd/narsilc
```

Install `narsilc-gen-json` to avoid test failure.

```
go build -o ~/go/bin/narsilc-gen-json ./cmd/narsilc-gen-json
```

## Running Tests

```
go test ./...
```

To run the tests in the examples folder, use the `examples` tag.

```
go test --tags=examples ./...
```

These tests require locally-running database instances. Run these databases
using [Docker Compose](https://docs.docker.com/compose/).

```
docker compose up -d
```

The tests use the following environment variables to connect to the
database

### For PostgreSQL

```
Variable     Default Value
-------------------------
PG_HOST      127.0.0.1
PG_PORT      5432
PG_USER      postgres
PG_PASSWORD  mysecretpassword
PG_DATABASE  dinotest
```

### For MySQL

```
Variable     Default Value
-------------------------
MYSQL_HOST      127.0.0.1
MYSQL_PORT      3306
MYSQL_USER      root
MYSQL_ROOT_PASSWORD  mysecretpassword
MYSQL_DATABASE  dinotest
```
