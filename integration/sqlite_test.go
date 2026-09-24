package integration_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	sqlok "github.com/candango/sqlok"
	sqlite "github.com/candango/sqlok-sqlite-modernc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type user struct {
	ID   int    `sqlok:"column=id,pk"`
	Name string `sqlok:"column=name"`
}

func (*user) TableName() string { return "users" }

type pair struct {
	TenantID int    `sqlok:"column=tenant_id,pk"`
	UserID   int    `sqlok:"column=user_id,pk"`
	Name     string `sqlok:"column=name"`
}

func (*pair) TableName() string { return "pairs" }

type invalidUser struct {
	ID   int    `sqlok:"column=id,pk"`
	Name string `sqlok:"column=name"`
}

func (*invalidUser) TableName() string { return "invalid_users" }

type missingUser struct {
	ID   int    `sqlok:"column=id,pk"`
	Name string `sqlok:"column=name"`
}

func (*missingUser) TableName() string { return "missing_users" }

func openDatabase(t *testing.T) *sql.DB {
	t.Helper()

	dataSourceName := filepath.Join(t.TempDir(), "sqlok.db")
	db, err := sqlite.Open(dataSourceName)
	require.NoError(t, err)
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	require.NoError(t, db.PingContext(ctx))
	_, err = db.ExecContext(ctx, `
		CREATE TABLE users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL
		);
		CREATE TABLE pairs (
			tenant_id INTEGER NOT NULL,
			user_id INTEGER NOT NULL,
			name TEXT NOT NULL,
			PRIMARY KEY (tenant_id, user_id)
		);
		CREATE TABLE invalid_users (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL
		);`)
	require.NoError(t, err)
	return db
}

func TestSQLiteMapperScansAndExtractsValues(t *testing.T) {
	db := openDatabase(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, "INSERT INTO users (name) VALUES (?)", "Ana")
	require.NoError(t, err)

	mapper, err := sqlok.NewMapper[user]()
	require.NoError(t, err)
	assert.Equal(t, []string{"id", "name"}, mapper.Columns())
	assert.Equal(t, []string{"id"}, mapper.PrimaryColumns())

	rows, err := db.QueryContext(ctx, "SELECT id, name FROM users WHERE name = ?", "Ana")
	require.NoError(t, err)
	require.True(t, rows.Next())
	entity, err := mapper.Scan(rows)
	require.NoError(t, err)
	require.NoError(t, rows.Close())
	require.NoError(t, rows.Err())

	assert.NotZero(t, entity.ID)
	assert.Equal(t, "Ana", entity.Name)
	values, err := mapper.Values(entity)
	require.NoError(t, err)
	assert.Equal(t, []sqlok.MappedValue{
		{Column: "id", Value: entity.ID, Primary: true},
		{Column: "name", Value: "Ana"},
	}, values)
}

func TestSQLiteAdapterFlushLoadAndGeneratedKey(t *testing.T) {
	db := openDatabase(t)
	ctx := context.Background()
	session := sqlok.NewSession(db)
	entity := &user{Name: "Ana"}
	require.NoError(t, session.Add(entity))

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, session.Flush(ctx, tx))

	var inTransaction int
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&inTransaction))
	assert.Equal(t, 1, inTransaction)
	require.NoError(t, tx.Commit())

	assert.NotZero(t, entity.ID)
	loaded, err := sqlok.LoadContext[user](ctx, session, entity.ID)
	require.NoError(t, err)
	assert.Same(t, entity, loaded)

	entity.Name = "Bia"
	tx, err = db.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, session.Flush(ctx, tx))
	require.NoError(t, tx.Commit())

	var name string
	require.NoError(t, db.QueryRowContext(
		ctx,
		"SELECT name FROM users WHERE id = ?",
		entity.ID,
	).Scan(&name))
	assert.Equal(t, "Bia", name)
}

func TestSQLiteIdentityMapReusesLoadedPointer(t *testing.T) {
	db := openDatabase(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, "INSERT INTO users (name) VALUES (?)", "Ana")
	require.NoError(t, err)

	var id int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT id FROM users WHERE name = ?", "Ana").Scan(&id))

	session := sqlok.NewSession(db)
	first, err := sqlok.LoadContext[user](ctx, session, id)
	require.NoError(t, err)
	require.NotNil(t, first)

	_, err = db.ExecContext(ctx, "UPDATE users SET name = ? WHERE id = ?", "Bia", id)
	require.NoError(t, err)
	second, err := sqlok.LoadContext[user](ctx, session, id)
	require.NoError(t, err)

	assert.Same(t, first, second)
	assert.Equal(t, "Ana", second.Name)
}

func TestSQLiteAdapterRollbackKeepsDatabaseUnchanged(t *testing.T) {
	db := openDatabase(t)
	ctx := context.Background()
	session := sqlok.NewSession(db)
	entity := &user{Name: "Rolled back"}
	require.NoError(t, session.Add(entity))

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, session.Flush(ctx, tx))
	assert.NotZero(t, entity.ID)

	var inTransaction int
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&inTransaction))
	assert.Equal(t, 1, inTransaction)
	require.NoError(t, tx.Rollback())

	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count))
	assert.Zero(t, count)
}

func TestSQLiteAdapterLoadsCompositeKeyAndMissingRows(t *testing.T) {
	db := openDatabase(t)
	ctx := context.Background()
	_, err := db.ExecContext(
		ctx,
		"INSERT INTO pairs (tenant_id, user_id, name) VALUES (?, ?, ?)",
		7,
		11,
		"Ana",
	)
	require.NoError(t, err)

	session := sqlok.NewSession(db)
	first, err := sqlok.LoadContext[pair](ctx, session, sqlok.CompositeKey{7, 11})
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, &pair{TenantID: 7, UserID: 11, Name: "Ana"}, first)

	second, err := sqlok.LoadContext[pair](ctx, session, sqlok.CompositeKey{7, 11})
	require.NoError(t, err)
	assert.Same(t, first, second)

	missing, err := sqlok.LoadContext[pair](ctx, session, sqlok.CompositeKey{7, 12})
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func TestSQLiteAdapterReportsActionableErrors(t *testing.T) {
	db := openDatabase(t)
	ctx := context.Background()

	t.Run("missing table", func(t *testing.T) {
		loaded, err := sqlok.LoadContext[missingUser](ctx, sqlok.NewSession(db), 1)
		require.Error(t, err)
		assert.Nil(t, loaded)
		assert.Contains(t, err.Error(), "query session load")
		assert.Contains(t, err.Error(), "missing_users")
	})

	t.Run("mapping failure", func(t *testing.T) {
		_, err := db.ExecContext(
			ctx,
			"INSERT INTO invalid_users (id, name) VALUES (?, ?)",
			"not-an-int",
			"Ana",
		)
		require.NoError(t, err)

		loaded, err := sqlok.LoadContext[invalidUser](ctx, sqlok.NewSession(db), "not-an-int")
		require.Error(t, err)
		assert.Nil(t, loaded)
		assert.Contains(t, err.Error(), "map session load row")
		assert.Contains(t, err.Error(), "id")
	})
}
