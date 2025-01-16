/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

// Package mysql provides helpers for working with the MySQL database using the github.com/go-sql-driver/mysql driver.
// Should be imported explicitly.
// To register mysql as retryable func use side effect import like so:
//
//	import _ "github.com/acronis/go-dbkit/mysql"
package mysql

import (
	"errors"

	"github.com/go-sql-driver/mysql"

	"github.com/acronis/go-dbkit"
)

// nolint
func init() {
	dbkit.RegisterIsRetryableFunc(&mysql.MySQLDriver{}, func(err error) bool {
		var mySQLError *mysql.MySQLError
		if errors.As(err, &mySQLError) {
			switch mySQLError.Number {
			case uint16(ErrDeadlock), uint16(ErrLockTimedOut):
				return true
			}
		}
		if errors.Is(err, mysql.ErrInvalidConn) {
			return true
		}
		return false
	})
}

// ErrCode defines the type for MySQL error codes.
type ErrCode uint16

// MySQL error codes (will be filled gradually).
const (
	ErrCodeDupEntry ErrCode = 1062
	ErrDeadlock     ErrCode = 1213
	ErrLockTimedOut ErrCode = 1205
)

// CheckMySQLError checks if the passed error relates to MySQL,
// and it's internal code matches the one from the argument.
func CheckMySQLError(err error, errCode ErrCode) bool {
	var mySQLError *mysql.MySQLError
	if errors.As(err, &mySQLError) {
		return mySQLError.Number == uint16(errCode)
	}
	return false
}
