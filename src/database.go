package main

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

func initDatabase(location string) (*sql.DB, error) {
	os.Remove(location)

	os.MkdirAll(filepath.Dir(location), 0755)

	log.Printf("Initializing Database at %s\n", location)

	db, err := sql.Open("sqlite3", location)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	sqlStmt := `CREATE TABLE searchIndex(id INTEGER PRIMARY KEY, name TEXT, type TEXT, path TEXT);`

	_, err = db.Exec(sqlStmt)
	if err != nil {
		defer db.Close()
		log.Printf("%q: %s\n", err, sqlStmt)
		return nil, err
	}

	return db, nil
}

func addEntryToDatabase(db *sql.DB, entryName string, entryType string, entryPath string) error {
	sqlStmt := `INSERT OR IGNORE INTO searchIndex(name, type, path) VALUES (?, ?, ?);`

	_, err := db.Exec(sqlStmt, entryName, entryType, entryPath)
	if err != nil {
		log.Printf("%q: %s\n", err, sqlStmt)
	}

	return err
}
