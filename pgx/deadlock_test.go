/*
Copyright © 2019-2023 Acronis International GmbH.
*/

package pgx

import (
	gotesting "testing"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/acronis/go-dbkit"
	"github.com/acronis/go-dbkit/internal/testing"
)

func TestDeadlockErrorHandling(t *gotesting.T) {
	testing.DeadlockTest(t, dbkit.DialectPgx,
		func(err error) bool {
			return CheckPostgresError(err, dbkit.PgxErrCodeDeadlockDetected)
		})
}
