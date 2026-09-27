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

type nullableUser struct {
	ID   int     `sqlok:"column=id,pk"`
	Name *string `sqlok:"column=name"`
}

func (*nullableUser) TableName() string { return "nullable_users" }

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
		);
		CREATE TABLE nullable_users (
			id INTEGER PRIMARY KEY,
			name TEXT
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

func TestSQLiteSelectMappedProjection(t *testing.T) {
	db := openDatabase(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, "INSERT INTO users (name) VALUES (?)", "Ana")
	require.NoError(t, err)

	var id int64
	require.NoError(t, db.QueryRowContext(
		ctx,
		"SELECT id FROM users WHERE name = ?",
		"Ana",
	).Scan(&id))

	rows, err := sqlok.Select(user{}).
		Where(sqlok.Eq("id", id)).
		Columns("id", "name").
		All(ctx, sqlok.NewSession(db))
	require.NoError(t, err)
	require.Len(t, rows, 1)

	assert.Equal(t, []string{"id", "name"}, rows[0].Columns())
	assert.Equal(t, []any{id, "Ana"}, rows[0].Values())
	value, ok := rows[0].Value("id")
	assert.True(t, ok)
	assert.Equal(t, id, value)
	value, ok = rows[0].Value("name")
	assert.True(t, ok)
	assert.Equal(t, "Ana", value)
	_, ok = rows[0].Value("missing")
	assert.False(t, ok)
}

func TestSQLiteSelectScalars(t *testing.T) {
	db := openDatabase(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, "INSERT INTO users (name) VALUES (?), (?)", "Ana", "Bia")
	require.NoError(t, err)

	values, err := sqlok.Select(user{}).
		Columns("name").
		Scalars(ctx, sqlok.NewSession(db))
	require.NoError(t, err)
	assert.Equal(t, []any{"Ana", "Bia"}, values)

	_, err = sqlok.Select(user{}).
		Columns("id", "name").
		Scalars(ctx, sqlok.NewSession(db))
	assert.ErrorIs(t, err, sqlok.ErrScalarSelectProjection)
}

func TestSQLiteSelectPredicates(t *testing.T) {
	db := openDatabase(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `
		INSERT INTO nullable_users (id, name) VALUES
			(1, NULL),
			(2, 'Ana'),
			(3, 'Bia')`)
	require.NoError(t, err)

	nullRows, err := sqlok.Select(nullableUser{}).
		Where(sqlok.IsNull("name")).
		All(ctx, sqlok.NewSession(db))
	require.NoError(t, err)
	require.Len(t, nullRows, 1)
	assert.Equal(t, 1, nullRows[0].ID)
	assert.Nil(t, nullRows[0].Name)

	notNullRows, err := sqlok.Select(nullableUser{}).
		Where(sqlok.IsNotNull("name")).
		All(ctx, sqlok.NewSession(db))
	require.NoError(t, err)
	require.Len(t, notNullRows, 2)
	assert.Equal(t, "Ana", *notNullRows[0].Name)
	assert.Equal(t, "Bia", *notNullRows[1].Name)

	greaterRows, err := sqlok.Select(nullableUser{}).
		Where(sqlok.Gt("id", 1)).
		All(ctx, sqlok.NewSession(db))
	require.NoError(t, err)
	require.Len(t, greaterRows, 2)
	assert.Equal(t, []int{2, 3}, []int{greaterRows[0].ID, greaterRows[1].ID})
}

func TestSQLiteAdapterFlushSelectAndGeneratedKey(t *testing.T) {
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
	loaded, err := sqlok.Select(user{}).
		Where(sqlok.Eq("id", entity.ID)).
		OneOrNone(ctx, session)
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

func TestSQLiteBoundSelectAutoflushesPendingInsert(t *testing.T) {
	db := openDatabase(t)
	ctx := context.Background()
	session := sqlok.NewSession(db)
	entity := &user{Name: "Pending"}
	require.NoError(t, session.Add(entity))

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, session.BindTransaction(tx))

	selected, err := sqlok.Select(user{}).
		Where(sqlok.Eq("name", "Pending")).
		OneOrNone(ctx, session)
	require.NoError(t, err)
	assert.Same(t, entity, selected)
	assert.NotZero(t, entity.ID)

	var count int
	require.NoError(t, tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count))
	assert.Equal(t, 1, count)

	session.UnbindTransaction()
	require.NoError(t, tx.Commit())
	require.NoError(t, db.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM users WHERE id = ?",
		entity.ID,
	).Scan(&count))
	assert.Equal(t, 1, count)
}

func TestSQLiteBoundSelectAutoflushesDirtyUpdate(t *testing.T) {
	db := openDatabase(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, "INSERT INTO users (name) VALUES (?)", "Ana")
	require.NoError(t, err)

	session := sqlok.NewSession(db)
	entity, err := sqlok.Select(user{}).
		Where(sqlok.Eq("name", "Ana")).
		OneOrNone(ctx, session)
	require.NoError(t, err)
	require.NotNil(t, entity)
	entity.Name = "Bia"

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	require.NoError(t, session.BindTransaction(tx))

	selected, err := sqlok.Select(user{}).
		Where(sqlok.Eq("id", entity.ID)).
		OneOrNone(ctx, session)
	require.NoError(t, err)
	assert.Same(t, entity, selected)
	assert.Equal(t, "Bia", selected.Name)

	var name string
	require.NoError(t, tx.QueryRowContext(
		ctx,
		"SELECT name FROM users WHERE id = ?",
		entity.ID,
	).Scan(&name))
	assert.Equal(t, "Bia", name)

	session.UnbindTransaction()
	require.NoError(t, tx.Rollback())
	require.NoError(t, db.QueryRowContext(
		ctx,
		"SELECT name FROM users WHERE id = ?",
		entity.ID,
	).Scan(&name))
	assert.Equal(t, "Ana", name)
}

func TestSQLiteIdentityMapReusesLoadedPointer(t *testing.T) {
	db := openDatabase(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, "INSERT INTO users (name) VALUES (?)", "Ana")
	require.NoError(t, err)

	var id int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT id FROM users WHERE name = ?", "Ana").Scan(&id))

	session := sqlok.NewSession(db)
	first, err := sqlok.Select(user{}).
		Where(sqlok.Eq("id", id)).
		OneOrNone(ctx, session)
	require.NoError(t, err)
	require.NotNil(t, first)

	_, err = db.ExecContext(ctx, "UPDATE users SET name = ? WHERE id = ?", "Bia", id)
	require.NoError(t, err)
	second, err := sqlok.Select(user{}).
		Where(sqlok.Eq("id", id)).
		OneOrNone(ctx, session)
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

func TestSQLiteAdapterSelectsCompositeKeyAndMissingRows(t *testing.T) {
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
	first, err := sqlok.Select(pair{}).
		Where(
			sqlok.Eq("tenant_id", 7),
			sqlok.Eq("user_id", 11),
		).
		OneOrNone(ctx, session)
	require.NoError(t, err)
	require.NotNil(t, first)
	assert.Equal(t, &pair{TenantID: 7, UserID: 11, Name: "Ana"}, first)

	second, err := sqlok.Select(pair{}).
		Where(
			sqlok.Eq("tenant_id", 7),
			sqlok.Eq("user_id", 11),
		).
		OneOrNone(ctx, session)
	require.NoError(t, err)
	assert.Same(t, first, second)

	missing, err := sqlok.Select(pair{}).
		Where(
			sqlok.Eq("tenant_id", 7),
			sqlok.Eq("user_id", 12),
		).
		OneOrNone(ctx, session)
	require.NoError(t, err)
	assert.Nil(t, missing)
}

func TestSQLiteAdapterReportsActionableErrors(t *testing.T) {
	db := openDatabase(t)
	ctx := context.Background()

	t.Run("missing table", func(t *testing.T) {
		loaded, err := sqlok.Select(missingUser{}).
			Where(sqlok.Eq("id", 1)).
			OneOrNone(ctx, sqlok.NewSession(db))
		require.Error(t, err)
		assert.Nil(t, loaded)
		assert.Contains(t, err.Error(), "query selected entities")
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

		loaded, err := sqlok.Select(invalidUser{}).
			Where(sqlok.Eq("id", "not-an-int")).
			OneOrNone(ctx, sqlok.NewSession(db))
		require.Error(t, err)
		assert.Nil(t, loaded)
		assert.Contains(t, err.Error(), "map selected entity")
		assert.Contains(t, err.Error(), "id")
	})
}
