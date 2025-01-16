/*
Copyright © 2019-2023 Acronis International GmbH.
*/

package postgres

import (
	gotesting "testing"

	_ "github.com/lib/pq"

	"github.com/acronis/go-dbkit"
	"github.com/acronis/go-dbkit/internal/testing"
)

func TestDeadlockErrorHandling(t *gotesting.T) {
	testing.DeadlockTest(t, dbkit.DialectPostgres,
		func(err error) bool {
			return CheckPostgresError(err, ErrCodeDeadlockDetected)
		})
}
