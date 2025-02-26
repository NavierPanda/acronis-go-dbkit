/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

package mssql

import (
	"database/sql/driver"
	"fmt"
	"testing"

	mssql "github.com/microsoft/go-mssqldb"
	"github.com/stretchr/testify/require"

	"github.com/acronis/go-dbkit"
)

func TestMSSQLIsRetryable(t *testing.T) {
	isRetryable := dbkit.GetIsRetryable(&mssql.Driver{})
	require.NotNil(t, isRetryable)
	require.True(t, isRetryable(mssql.Error{Number: 1205}))
	require.False(t, isRetryable(driver.ErrBadConn))
	require.True(t, isRetryable(fmt.Errorf("wrapped error: %w", mssql.Error{Number: 1205})))
}

func TestCheckMSSQLError(t *testing.T) {
	var err error
	err = mssql.Error{Number: 1205}
	require.True(t, CheckMSSQLError(err, ErrDeadlock))
	err = mssql.Error{Number: 9999}
	require.False(t, CheckMSSQLError(err, ErrDeadlock))
	err = fmt.Errorf("wrapped error: %w", mssql.Error{Number: 1205})
	require.True(t, CheckMSSQLError(err, ErrDeadlock))
}
