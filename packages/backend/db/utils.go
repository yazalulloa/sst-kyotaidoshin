package db

import (
	"database/sql"
	"log"
)

func MakeStmt(p *sql.Stmt, query string) *sql.Stmt {
	if p != nil {
		return p
	}
	stmt, err := GetDB().DB.Prepare(query)
	if err != nil {
		log.Fatalf("Error preparing query: %s", err)
	}

	p = stmt

	return p
}
