/*
Copyright © 2024 Acronis International GmbH.

Released under MIT license.
*/

package migrate

import (
	"bytes"
	"database/sql"
	"embed"
	"fmt"
	"io"
	"testing"
	"time"

	"github.com/acronis/go-appkit/log/logtest"
	_ "github.com/mattn/go-sqlite3"
	migrate "github.com/rubenv/sql-migrate"
	"github.com/stretchr/testify/require"

	"github.com/acronis/go-dbkit"
)

type testMigration00001CreateTables struct {
	*NullMigration
}

func newTestMigration00001CreateTables() *testMigration00001CreateTables {
	return &testMigration00001CreateTables{NullMigration: &NullMigration{}}
}

func (m *testMigration00001CreateTables) ID() string {
	return "00001_create_users_and_notes_tables"
}

// nolint: lll
func (m *testMigration00001CreateTables) UpSQL() []string {
	return []string{
		`CREATE TABLE users (id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL)`,
		`CREATE TABLE notes (id INTEGER NOT NULL PRIMARY KEY AUTOINCREMENT, content TEXT, user_id INTEGER NOT NULL, FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE)`,
	}
}

func (m *testMigration00001CreateTables) DownSQL() []string {
	return []string{
		`DROP TABLE users`,
		`DROP TABLE notes`,
	}
}

type testMigration00002SeedTables struct {
	*NullMigration
}

func (m *testMigration00002SeedTables) ID() string {
	return "00002_seed_users_and_notes_tables"
}

func newTestMigration00002SeedTabled() *testMigration00002SeedTables {
	return &testMigration00002SeedTables{NullMigration: &NullMigration{}}
}

// nolint: lll
func (m *testMigration00002SeedTables) UpSQL() []string {
	return []string{
		`INSERT INTO users(name) VALUES("Albert"), ("Bob"), ("John"), ("Sam"), ("Sam")`,
		`INSERT INTO notes(content, user_id) VALUES("first-note", 1), ("second-note", 2)`,
	}
}

func (m *testMigration00002SeedTables) DownSQL() []string {
	return []string{
		`DELETE FROM users`,
		`DELETE FROM notes`,
	}
}

type testMigration00003RawMigration struct {
	*NullMigration
	MakeError bool
}

func (m *testMigration00003RawMigration) ID() string {
	return "00003_raw_migration"
}

func newTestMigration00003RawMigration() *testMigration00003RawMigration {
	return &testMigration00003RawMigration{}
}

// nolint: lll
func (m *testMigration00003RawMigration) RawMigration(self Migration) (*migrate.Migration, error) {
	if m.MakeError {
		return nil, fmt.Errorf("fake fatal error")
	}
	rawSQL := `
-- +migrate Up
INSERT INTO users(name) VALUES("AlbertRaw"), ("BobRaw"), ("JohnRaw"), ("SamRaw"), ("SamRaw");
INSERT INTO notes(content, user_id) VALUES ("raw-first-note", 6), ("raw-second-note", 7);
-- +migrate Down
DELETE FROM notes WHERE content LIKE 'raw%';
DELETE FROM users WHERE name LIKE '%Raw';
`
	return migrate.ParseMigration(self.ID(), bytes.NewReader([]byte(rawSQL)))
}

type testMigration00004NoTransaction struct {
	*NullMigration
	MakeError bool
}

func (m *testMigration00004NoTransaction) ID() string {
	return "00004_no_transaction"
}

func newTestMigration00004NoTransaction() *testMigration00004NoTransaction {
	return &testMigration00004NoTransaction{}
}

// nolint: lll
func (m *testMigration00004NoTransaction) UpSQL() []string {
	result := []string{
		`INSERT INTO users(name) VALUES ("LAMBERT")`,
	}
	if m.MakeError {
		result = append(result, `Some error statement not in transaction`)
	}
	return result
}

func (m *testMigration00004NoTransaction) DownSQL() []string {
	return []string{
		`DELETE FROM users WHERE name="LAMBERT"`,
	}
}

func (m *testMigration00004NoTransaction) DisableTx() bool {
	return true
}

func requireMigrationsApplied(t *testing.T, dbConn *sql.DB, wantNoTablesErr bool, wantUsersCount, wantNotesCount int) {
	t.Helper()
	var usersCount int
	var notesCount int

	if wantNoTablesErr {
		require.EqualError(t, dbConn.QueryRow("select count(*) from users").Scan(&usersCount), "no such table: users")
		require.EqualError(t, dbConn.QueryRow("select count(*) from notes").Scan(&notesCount), "no such table: notes")
		return
	}

	require.NoError(t, dbConn.QueryRow("select count(*) from users").Scan(&usersCount))
	require.Equal(t, wantUsersCount, usersCount)
	require.NoError(t, dbConn.QueryRow("select count(*) from notes").Scan(&notesCount))
	require.Equal(t, wantNotesCount, notesCount)
}

func TestMigrationsManager_Run(t *testing.T) {
	dbConn, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer requireNoErrOnClose(t, dbConn)

	migMngr, err := NewMigrationsManager(dbConn, dbkit.DialectSQLite, logtest.NewLogger())
	require.NoError(t, err)
	migrations := []Migration{newTestMigration00001CreateTables(), newTestMigration00002SeedTabled()}

	// Check users and notes tables don't exist before migrations applying .
	requireMigrationsApplied(t, dbConn, true, 0, 0)

	// Apply migrations and check that all is ok.
	require.NoError(t, migMngr.Run(migrations, MigrationsDirectionUp))
	requireMigrationsApplied(t, dbConn, false, 5, 2)

	// Rollback migrations and check that tables were dropped.
	require.NoError(t, migMngr.Run(migrations, MigrationsDirectionDown))
	requireMigrationsApplied(t, dbConn, true, 0, 0)
}

func TestMigrationsManager_RunLimit(t *testing.T) {
	dbConn, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer requireNoErrOnClose(t, dbConn)

	migMngr, err := NewMigrationsManager(dbConn, dbkit.DialectSQLite, logtest.NewLogger())
	require.NoError(t, err)
	migrations := []Migration{newTestMigration00001CreateTables(), newTestMigration00002SeedTabled()}

	// Check users and notes tables don't exist before migrations applying .
	requireMigrationsApplied(t, dbConn, true, 0, 0)

	// Apply migrations and check that all is ok.
	require.NoError(t, migMngr.RunLimit(migrations, MigrationsDirectionUp, 1))
	requireMigrationsApplied(t, dbConn, false, 0, 0)
	require.NoError(t, migMngr.RunLimit(migrations, MigrationsDirectionUp, 1))
	requireMigrationsApplied(t, dbConn, false, 5, 2)

	// Rollback migrations and check that tables were dropped.
	require.NoError(t, migMngr.RunLimit(migrations, MigrationsDirectionDown, 1))
	requireMigrationsApplied(t, dbConn, false, 0, 0)
	require.NoError(t, migMngr.RunLimit(migrations, MigrationsDirectionDown, 1))
	requireMigrationsApplied(t, dbConn, true, 0, 0)
}

func TestMigrationsManager_Status(t *testing.T) {
	dbConn, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer requireNoErrOnClose(t, dbConn)

	migMngr, err := NewMigrationsManager(dbConn, dbkit.DialectSQLite, logtest.NewLogger())
	require.NoError(t, err)

	migStatus, err := migMngr.Status()
	require.NoError(t, err)
	require.Len(t, migStatus.AppliedMigrations, 0)
	_, exist := migStatus.LastAppliedMigration()
	require.False(t, exist)

	migrations := []Migration{newTestMigration00001CreateTables(), newTestMigration00002SeedTabled()}
	require.NoError(t, migMngr.Run(migrations, MigrationsDirectionUp))

	migStatus, err = migMngr.Status()
	require.NoError(t, err)
	require.Len(t, migStatus.AppliedMigrations, 2)
	lastAppliedMig, exist := migStatus.LastAppliedMigration()
	require.True(t, exist)
	require.Equal(t, migrations[len(migrations)-1].ID(), lastAppliedMig.ID)
	require.WithinDuration(t, time.Now(), lastAppliedMig.AppliedAt, time.Second)
}

func TestCreationMigrationManagerWithOpts(t *testing.T) {
	const tableName = "custom_migrations"
	dbConn, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer requireNoErrOnClose(t, dbConn)

	migMngr, err := NewMigrationsManagerWithOpts(dbConn, dbkit.DialectSQLite, logtest.NewLogger(),
		MigrationsManagerOpts{TableName: tableName})
	require.NoError(t, err)
	require.Equal(t, tableName, migMngr.migSet.TableName)

	migrations := []Migration{newTestMigration00001CreateTables(), newTestMigration00002SeedTabled()}
	var rowsNum int

	// The table doesn't exist before migrations.
	require.Error(t, dbConn.QueryRow("select count(*) from custom_migrations").Scan(&rowsNum))

	// Run migrations.
	require.NoError(t, migMngr.Run(migrations, MigrationsDirectionUp))
	require.NoError(t, dbConn.QueryRow("select count(*) from custom_migrations").Scan(&rowsNum))
	require.Equal(t, len(migrations), rowsNum)
	require.NoError(t, migMngr.Run(migrations, MigrationsDirectionDown))

	// Table exists after migrations.
	require.NoError(t, dbConn.QueryRow("select count(*) from custom_migrations").Scan(&rowsNum))
	require.Equal(t, 0, rowsNum)
}

func requireNoErrOnClose(t *testing.T, closer io.Closer) {
	t.Helper()
	require.NoError(t, closer.Close())
}

func TestMigrationsManager_supportRawMigration(t *testing.T) {
	dbConn, err := sql.Open("sqlite3", "file::memory:?cache=shared")
	require.NoError(t, err)
	defer requireNoErrOnClose(t, dbConn)

	migMngr, err := NewMigrationsManager(dbConn, dbkit.DialectSQLite, logtest.NewLogger())
	require.NoError(t, err)
	migrations := []Migration{
		newTestMigration00001CreateTables(),
		newTestMigration00002SeedTabled(),
		newTestMigration00003RawMigration(),
		newTestMigration00004NoTransaction(),
	}
	require.NoError(t, err)

	// Check users and notes tables don't exist before migrations applying .
	requireMigrationsApplied(t, dbConn, true, 0, 0)

	// Apply migrations and check that all is ok.
	require.NoError(t, migMngr.RunLimit(migrations, MigrationsDirectionUp, 1))
	requireMigrationsApplied(t, dbConn, false, 0, 0)
	require.NoError(t, migMngr.RunLimit(migrations, MigrationsDirectionUp, 1))
	requireMigrationsApplied(t, dbConn, false, 5, 2)

	migration00003RawMigration := (migrations[2]).(*testMigration00003RawMigration)
	migration00003RawMigration.MakeError = true
	require.Error(t, migMngr.RunLimit(migrations, MigrationsDirectionUp, 1))
	requireMigrationsApplied(t, dbConn, false, 5, 2)
	migration00003RawMigration.MakeError = false
	require.NoError(t, migMngr.RunLimit(migrations, MigrationsDirectionUp, 1))
	requireMigrationsApplied(t, dbConn, false, 10, 4)

	migration00004NoTransaction := (migrations[3]).(*testMigration00004NoTransaction)
	migration00004NoTransaction.MakeError = true
	require.Error(t, migMngr.RunLimit(migrations, MigrationsDirectionUp, 1))
	requireMigrationsApplied(t, dbConn, false, 11, 4)
	migration00004NoTransaction.MakeError = false
	require.NoError(t, migMngr.RunLimit(migrations, MigrationsDirectionUp, 1))
	requireMigrationsApplied(t, dbConn, false, 12, 4)

	// Rollback migrations and check that tables were dropped.
	require.NoError(t, migMngr.RunLimit(migrations, MigrationsDirectionDown, 1))
	requireMigrationsApplied(t, dbConn, false, 10, 4)
	require.NoError(t, migMngr.RunLimit(migrations, MigrationsDirectionDown, 1))
	requireMigrationsApplied(t, dbConn, false, 5, 2)
	require.NoError(t, migMngr.RunLimit(migrations, MigrationsDirectionDown, 1))
	requireMigrationsApplied(t, dbConn, false, 0, 0)
	require.NoError(t, migMngr.RunLimit(migrations, MigrationsDirectionDown, 1))
	requireMigrationsApplied(t, dbConn, true, 0, 0)
}

//go:embed testdata/sqlite/*.sql
//go:embed testdata/missing-down-file/*.sql
//go:embed testdata/missing-up-file/*.sql
//go:embed testdata/invalid-suffix/*.sql
var testFS embed.FS

func TestAllLoadEmbedFSMigrations(t *testing.T) {
	tests := []struct {
		name        string
		fs          embed.FS
		dirName     string
		wantErrMsg  string
		expectedIDs []string
	}{
		{
			name:        "valid migrations",
			fs:          testFS,
			dirName:     "testdata/sqlite",
			expectedIDs: []string{"0001_create_users_table", "0002_create_notes_table", "0003_seed_tables"},
		},
		{
			name:       "missing up file",
			fs:         testFS,
			dirName:    "testdata/missing-up-file",
			wantErrMsg: "0001_create_users_table migration up file is missing",
		},
		{
			name:       "missing down file",
			fs:         testFS,
			dirName:    "testdata/missing-down-file",
			wantErrMsg: "0001_create_users_table migration down file is missing",
		},
		{
			name:       "invalid suffix",
			fs:         testFS,
			dirName:    "testdata/invalid-suffix",
			wantErrMsg: "migration file should have .up.sql or .down.sql suffix, got 0001_create_users_table.sql",
		},
		{
			name:       "non-existent directory",
			fs:         testFS,
			dirName:    "testdata/non-existent",
			wantErrMsg: "read migrations directory testdata/non-existent: open testdata/non-existent: file does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrations, err := LoadAllEmbedFSMigrations(tt.fs, tt.dirName)
			if tt.wantErrMsg != "" {
				require.EqualError(t, err, tt.wantErrMsg)
				return
			}
			require.NoError(t, err)
			require.Len(t, migrations, len(tt.expectedIDs))
			for i, migration := range migrations {
				require.Equal(t, tt.expectedIDs[i], migration.ID())
			}

			dbConn, err := sql.Open("sqlite3", "file::memory:?cache=shared")
			require.NoError(t, err)
			defer requireNoErrOnClose(t, dbConn)

			migManager, err := NewMigrationsManager(dbConn, dbkit.DialectSQLite, logtest.NewLogger())
			require.NoError(t, err)
			require.NoError(t, migManager.Run(migrations, MigrationsDirectionUp))

			var usersCount int
			require.NoError(t, dbConn.QueryRow("select count(*) from users").Scan(&usersCount))
			require.Equal(t, 3, usersCount)
			var notesCount int
			require.NoError(t, dbConn.QueryRow("select count(*) from notes").Scan(&notesCount))
			require.Equal(t, 2, notesCount)

			migStatus, err := migManager.Status()
			require.NoError(t, err)
			appliedIDs := make([]string, 0, len(migStatus.AppliedMigrations))
			for _, mig := range migStatus.AppliedMigrations {
				appliedIDs = append(appliedIDs, mig.ID)
			}
			require.Equal(t, tt.expectedIDs, appliedIDs)
		})
	}
}

func TestLoadEmbedFSMigrations(t *testing.T) {
	tests := []struct {
		name         string
		fs           embed.FS
		dirName      string
		migrationIDs []string
		wantErrMsg   string
		expectedIDs  []string
	}{
		{
			name:         "valid migrations",
			fs:           testFS,
			dirName:      "testdata/sqlite",
			migrationIDs: []string{"0001_create_users_table", "0002_create_notes_table"},
			expectedIDs:  []string{"0001_create_users_table", "0002_create_notes_table"},
		},
		{
			name:         "missing up file",
			fs:           testFS,
			dirName:      "testdata/missing-up-file",
			migrationIDs: []string{"0001_create_users_table"},
			wantErrMsg:   "open testdata/missing-up-file/0001_create_users_table.up.sql: file does not exist",
		},
		{
			name:         "missing down file",
			fs:           testFS,
			dirName:      "testdata/missing-down-file",
			migrationIDs: []string{"0001_create_users_table"},
			wantErrMsg:   "open testdata/missing-down-file/0001_create_users_table.down.sql: file does not exist",
		},
		{
			name:         "invalid migration ID",
			fs:           testFS,
			dirName:      "testdata/sqlite",
			migrationIDs: []string{"invalid_migration_id"},
			wantErrMsg:   "open testdata/sqlite/invalid_migration_id.up.sql: file does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			migrations, err := LoadEmbedFSMigrations(tt.fs, tt.dirName, tt.migrationIDs)
			if tt.wantErrMsg != "" {
				require.EqualError(t, err, tt.wantErrMsg)
				return
			}
			require.NoError(t, err)
			require.Len(t, migrations, len(tt.expectedIDs))
			for i, migration := range migrations {
				require.Equal(t, tt.expectedIDs[i], migration.ID())
			}

			dbConn, err := sql.Open("sqlite3", "file::memory:?cache=shared")
			require.NoError(t, err)
			defer requireNoErrOnClose(t, dbConn)

			migManager, err := NewMigrationsManager(dbConn, dbkit.DialectSQLite, logtest.NewLogger())
			require.NoError(t, err)
			require.NoError(t, migManager.Run(migrations, MigrationsDirectionUp))

			var usersCount int
			require.NoError(t, dbConn.QueryRow("select count(*) from users").Scan(&usersCount))
			require.Equal(t, 0, usersCount)
			var notesCount int
			require.NoError(t, dbConn.QueryRow("select count(*) from notes").Scan(&notesCount))
			require.Equal(t, 0, notesCount)

			migStatus, err := migManager.Status()
			require.NoError(t, err)
			appliedIDs := make([]string, 0, len(migStatus.AppliedMigrations))
			for _, mig := range migStatus.AppliedMigrations {
				appliedIDs = append(appliedIDs, mig.ID)
			}
			require.Equal(t, tt.expectedIDs, appliedIDs)
		})
	}
}
