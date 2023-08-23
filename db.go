package main

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/ekzyis/sn-goapi"
	_ "github.com/mattn/go-sqlite3"
)

var (
	db *sql.DB
)

func init() {
	var err error
	db, err = sql.Open("sqlite3", "unpaywall.sqlite3")
	if err != nil {
		log.Fatal(err)
	}
	migrate(db)
}

func migrate(db *sql.DB) {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS comments (
			id INTEGER PRIMARY KEY,
			text TEXT NOT NULL,
			parent_id INTEGER NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		err = fmt.Errorf("error during migration: %w", err)
		log.Fatal(err)
	}
}

func ItemHasComment(parentId int) bool {
	var count int
	err := db.QueryRow(`SELECT COUNT(1) FROM comments WHERE parent_id = ?`, parentId).Scan(&count)
	if err != nil {
		err = fmt.Errorf("error during item check: %w", err)
		log.Fatal(err)
	}
	return count > 0
}

func SaveComment(comment *sn.Comment) {
	_, err := db.Exec(`INSERT INTO comments(id, text, parent_id) VALUES (?, ?, ?)`, comment.Id, comment.Text, comment.ParentId)
	if err != nil {
		err = fmt.Errorf("error during item insert: %w", err)
		log.Fatal(err)
	}
}
