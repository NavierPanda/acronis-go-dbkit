/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

// Package postgres provides helpers for working with the Postgres database using the github.com/lib/pq driver.
// Should be imported explicitly.
// To register postgres as retryable func use side effect import like so:
//
//	import _ "github.com/acronis/go-dbkit/postgres"
package postgres

import (
	"errors"

	"github.com/lib/pq"

	"github.com/acronis/go-dbkit"
)

// nolint
func init() {
	dbkit.RegisterIsRetryableFunc(&pq.Driver{}, func(err error) bool {
		var pgErr *pq.Error
		if errors.As(err, &pgErr) {
			name := ErrCode(pgErr.Code.Name())
			switch name {
			case ErrCodeDeadlockDetected:
				return true
			case ErrCodeSerializationFailure:
				return true
			}
		}
		return false
	})
}

// ErrCode defines the type for Postgres error codes.
type ErrCode string

// Postgres error codes (will be filled gradually).
const (
	ErrCodeUniqueViolation      ErrCode = "unique_violation"
	ErrCodeDeadlockDetected     ErrCode = "deadlock_detected"
	ErrCodeSerializationFailure ErrCode = "serialization_failure"
)

// CheckPostgresError checks if the passed error relates to Postgres,
// and it's internal code matches the one from the argument.
func CheckPostgresError(err error, errCode ErrCode) bool {
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		return pgErr.Code.Name() == string(errCode)
	}
	return false
}
