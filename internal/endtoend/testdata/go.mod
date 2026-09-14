module github.com/mbvlabs/narsilc/endtoend

go 1.27.0

toolchain go1.27.1

require (
	github.com/go-sql-driver/mysql v1.10.1
	github.com/gofrs/uuid v4.0.0+incompatible
	github.com/google/uuid v1.3.0
	github.com/hexon/mysqltsv v0.1.0
	github.com/jackc/pgconn v1.5.1-0.20200601181101-fa742c524853
	github.com/jackc/pgtype v1.6.2
	github.com/jackc/pgx/v4 v4.6.1-0.20200606145419-4e5062306904
	github.com/jackc/pgx/v5 v5.11.0
	github.com/lib/pq v1.12.3
	github.com/mbvlabs/narsilc v0.0.0
	github.com/pgvector/pgvector-go v0.1.1
	github.com/sqlc-dev/pqtype v0.2.0
	github.com/sqlc-dev/sqlc-testdata v1.0.0
	github.com/volatiletech/null/v8 v8.1.2
	gopkg.in/guregu/null.v4 v4.0.0
)

replace github.com/mbvlabs/narsilc => ../../..

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/friendsofgo/errors v0.9.2 // indirect
	github.com/jackc/chunkreader/v2 v2.0.1 // indirect
	github.com/jackc/pgio v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgproto3/v2 v2.0.1 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/volatiletech/inflect v0.0.1 // indirect
	github.com/volatiletech/randomize v0.0.1 // indirect
	github.com/volatiletech/strmangle v0.0.1 // indirect
	golang.org/x/crypto v0.17.0 // indirect
	golang.org/x/text v0.41.0 // indirect
	golang.org/x/xerrors v0.0.0-20200804184101-5ec99f83aff1 // indirect
)
