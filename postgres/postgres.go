/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

// Package postgres provides helpers for working Postgres database.
// Should be imported explicitly.
// To register postgres as retryable func use side effect import like so:
//
//	import _ "github.com/acronis/go-dbkit/postgres"
package postgres

import (
	"errors"

	pg "github.com/lib/pq"

	"github.com/acronis/go-dbkit"
)

// nolint
func init() {
	dbkit.RegisterIsRetryableFunc(&pg.Driver{}, func(err error) bool {
		var pgErr *pg.Error
		if errors.As(err, &pgErr) {
			name := dbkit.PostgresErrCode(pgErr.Code.Name())
			switch name {
			case dbkit.PostgresErrCodeDeadlockDetected:
				return true
			case dbkit.PostgresErrCodeSerializationFailure:
				return true
			}
		}
		return false
	})
}

// CheckPostgresError checks if the passed error relates to Postgres and it's internal code matches the one from the argument.
// nolint: staticcheck // lib/pq using is deprecated. Use pgx Postgres driver.
func CheckPostgresError(err error, errCode dbkit.PostgresErrCode) bool {
	var pgErr *pg.Error
	if errors.As(err, &pgErr) {
		return pgErr.Code.Name() == string(errCode)
	}
	return false
}
