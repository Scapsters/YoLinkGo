package mysql

import (
	"context"
	"database/sql"
	"fmt"
)

func dropTable(ctx context.Context, db *sql.DB, tableName string) error{
	sqlctx, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	_, err := db.ExecContext(sqlctx, `SET FOREIGN_KEY_CHECKS = 0`)
	if err != nil {
		return fmt.Errorf("error disabling FK checks: %w", err)
	}
	sqlctx, cancel = context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	_, err = db.ExecContext(sqlctx, `DROP TABLE IF EXISTS ` + tableName)
	if err != nil {
		return fmt.Errorf("error dropping logs table: %w", err)
	}
	sqlctx, cancel = context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	_, err = db.ExecContext(sqlctx, `SET FOREIGN_KEY_CHECKS = 1`)
	if err != nil {
		return fmt.Errorf("error enabling FK checks: %w", err)
	}
	return nil
}