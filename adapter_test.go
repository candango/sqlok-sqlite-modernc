package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenPingSchemaAndOperations(t *testing.T) {
	db, err := Open(filepath.Join(t.TempDir(), "adapter.db"))
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	ctx := context.Background()
	require.NoError(t, db.PingContext(ctx))

	_, err = db.ExecContext(ctx, `
		CREATE TABLE widgets (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			quantity INTEGER NOT NULL
		)`)
	require.NoError(t, err)

	result, err := db.ExecContext(
		ctx,
		"INSERT INTO widgets (id, name, quantity) VALUES (?, ?, ?)",
		1,
		"blue",
		2,
	)
	require.NoError(t, err)
	affected, err := result.RowsAffected()
	require.NoError(t, err)
	assert.Equal(t, int64(1), affected)

	var name string
	var quantity int
	require.NoError(t, db.QueryRowContext(
		ctx,
		"SELECT name, quantity FROM widgets WHERE id = ?",
		1,
	).Scan(&name, &quantity))
	assert.Equal(t, "blue", name)
	assert.Equal(t, 2, quantity)

	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, "UPDATE widgets SET quantity = ? WHERE id = ?", 3, 1)
	require.NoError(t, err)
	require.NoError(t, tx.Rollback())

	require.NoError(t, db.QueryRowContext(
		ctx,
		"SELECT quantity FROM widgets WHERE id = ?",
		1,
	).Scan(&quantity))
	assert.Equal(t, 2, quantity)

	tx, err = db.BeginTx(ctx, nil)
	require.NoError(t, err)
	_, err = tx.ExecContext(ctx, "UPDATE widgets SET quantity = ? WHERE id = ?", 4, 1)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	require.NoError(t, db.QueryRowContext(
		ctx,
		"SELECT quantity FROM widgets WHERE id = ?",
		1,
	).Scan(&quantity))
	assert.Equal(t, 4, quantity)

	_, err = db.ExecContext(ctx, "DELETE FROM widgets WHERE id = ?", 1)
	require.NoError(t, err)
	var count int
	require.NoError(t, db.QueryRowContext(ctx, "SELECT COUNT(*) FROM widgets").Scan(&count))
	assert.Zero(t, count)
}
