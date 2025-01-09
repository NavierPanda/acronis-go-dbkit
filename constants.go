/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

package dbkit

import (
	"database/sql"
	"time"
)

// Default values of connection parameters
const (
	DefaultMaxIdleConns    = 2
	DefaultMaxOpenConns    = 10
	DefaultConnMaxLifetime = 10 * time.Minute // Official recommendation from the DBA team
)

// MSSQLDefaultTxLevel contains transaction isolation level which will be used by default for MSSQL.
const MSSQLDefaultTxLevel = sql.LevelReadCommitted

// MySQLDefaultTxLevel contains transaction isolation level which will be used by default for MySQL.
const MySQLDefaultTxLevel = sql.LevelReadCommitted

// PostgresDefaultTxLevel contains transaction isolation level which will be used by default for Postgres.
const PostgresDefaultTxLevel = sql.LevelReadCommitted

// PostgresDefaultSSLMode contains Postgres SSL mode which will be used by default.
const PostgresDefaultSSLMode = PostgresSSLModeVerifyCA

// PgTargetSessionAttrs session attrs parameter name
const PgTargetSessionAttrs = "target_session_attrs"

// PgReadWriteParam read-write session attribute value name
const PgReadWriteParam = "read-write"

// Dialect defines possible values for planned supported SQL dialects.
type Dialect string

// SQL dialects.
const (
	DialectSQLite   Dialect = "sqlite3"
	DialectMySQL    Dialect = "mysql"
	DialectPostgres Dialect = "postgres"
	DialectPgx      Dialect = "pgx"
	DialectMSSQL    Dialect = "mssql"
)

// PostgresSSLMode defines possible values for Postgres sslmode connection parameter.
type PostgresSSLMode string

// Postgres SSL modes.
const (
	PostgresSSLModeDisable    PostgresSSLMode = "disable"
	PostgresSSLModeRequire    PostgresSSLMode = "require"
	PostgresSSLModeVerifyCA   PostgresSSLMode = "verify-ca"
	PostgresSSLModeVerifyFull PostgresSSLMode = "verify-full"
)
