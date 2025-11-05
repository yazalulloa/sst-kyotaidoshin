package db

import (
	"database/sql"
	"log"
	"sync"

	"github.com/sst/sst/v3/sdk/golang/resource"
	_ "github.com/tursodatabase/libsql-client-go/libsql"
)

type SqlDB struct {
	DB *sql.DB
}

var instance *SqlDB
var once sync.Once

func initDB() *sql.DB {

	url, err := resource.Get("SecretTursoUrl", "value")
	if err != nil {
		log.Fatalf("Error getting turso url: %v", err)
	}

	db, err := sql.Open("libsql", url.(string))
	if err != nil {
		log.Fatalf("Error opening database: %v", err)
	}

	return db
}

func GetDB() *SqlDB {

	once.Do(func() {

		instance = &SqlDB{DB: initDB()}
	})

	return instance
}
