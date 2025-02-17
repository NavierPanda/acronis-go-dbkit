/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

package pgx

import (
	"context"
	"database/sql/driver"
	"fmt"
	gotesting "testing"
	"time"

	"github.com/jackc/pgx/v5/pgconn"
	pg "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	"github.com/acronis/go-dbkit"
	"github.com/acronis/go-dbkit/internal/testing"
)

func TestPostgresIsRetryable(t *gotesting.T) {
	isRetryable := dbkit.GetIsRetryable(&pg.Driver{})
	require.NotNil(t, isRetryable)
	// enum all retriable errors
	retriable := []ErrCode{
		ErrCodeDeadlockDetected,
		ErrCodeSerializationFailure,
	}
	for _, code := range retriable {
		var err error
		err = &pgconn.PgError{Code: string(code)}
		require.True(t, isRetryable(err))
		err = fmt.Errorf("Wrapped error: %w", err)
		require.True(t, isRetryable(err))
		err = fmt.Errorf("One more time wrapped error: %w", err)
		require.True(t, isRetryable(err))
	}

	require.False(t, isRetryable(driver.ErrBadConn))
}

func TestCheckInvalidCachedPlanError(t *gotesting.T) {
	ctx, ctxCancel := context.WithTimeout(context.Background(), time.Minute*2)
	defer ctxCancel()

	conn, stop := testing.MustRunAndOpenTestDB(ctx, string(dbkit.DialectPgx))
	defer func() { require.NoError(t, stop(ctx)) }()

	// Create a table and fill it with some data.
	_, err := conn.ExecContext(ctx, `
        DROP TABLE IF EXISTS drop_cols;
        CREATE TABLE drop_cols (
            id SERIAL PRIMARY KEY NOT NULL,
            f1 int NOT NULL,
            f2 int NOT NULL
        );
    `)
	require.NoError(t, err)
	_, err = conn.ExecContext(ctx, "INSERT INTO drop_cols (f1, f2) VALUES (1, 2)")
	require.NoError(t, err)

	getSQL := "SELECT * FROM drop_cols WHERE id = $1"

	// This query will populate the statement cache. We don't care about the result.
	rows, err := conn.QueryContext(ctx, getSQL, 1)
	require.NoError(t, err)
	require.NoError(t, rows.Close())
	require.NoError(t, rows.Err())

	// Now, change the schema of the table out from under the statement, making it invalid.
	_, err = conn.ExecContext(ctx, "ALTER TABLE drop_cols DROP COLUMN f1")
	require.NoError(t, err)

	rows, err = conn.QueryContext(ctx, getSQL, 1)
	if err != nil {
		require.True(t, CheckInvalidCachedPlanError(err))
	} else {
		require.True(t, CheckInvalidCachedPlanError(rows.Err()))
		require.True(t, CheckInvalidCachedPlanError(rows.Close()))
	}

	// On retry, the statement should have been flushed from the cache.
	rows, err = conn.QueryContext(ctx, getSQL, 1)
	require.NoError(t, err)
	require.True(t, rows.Next())
	require.NoError(t, rows.Close())
}
