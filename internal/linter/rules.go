package linter

import (
	"errors"

	pg_query "github.com/pganalyze/pg_query_go/v6"
)

func ValidateSQL(sql string) error {
	// Parse the SQL using the actual Postgres parser logic
	result, err := pg_query.Parse(sql)
	if err != nil {
		// If it's a command the parser doesn't know (like a custom tool command),
		// we usually let it pass in a proxy.
		return nil
	}

	// Iterate through the statements in the query
	for _, stmt := range result.Stmts {
		// Dig into the 'Delete' statement
		if deleteStmt := stmt.GetStmt().GetDeleteStmt(); deleteStmt != nil {
			if deleteStmt.WhereClause == nil {
				return errors.New("psql-lintproxy: delete without where clause blocked")
			}
		}

		// Dig into the 'Update' statement
		if updateStmt := stmt.GetStmt().GetUpdateStmt(); updateStmt != nil {
			if updateStmt.WhereClause == nil {
				return errors.New("psql-lintproxy: update without where clause blocked")
			}
		}
	}

	return nil
}
