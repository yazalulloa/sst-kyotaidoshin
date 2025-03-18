package permissions

import (
	"db"
	"db/gen/model"
	. "db/gen/table"
	"github.com/go-jet/jet/v2/sqlite"
	"log"
)

func insertBulk(perms []string) (int64, error) {

	stmt := Permissions.INSERT(Permissions.Name).ON_CONFLICT().DO_NOTHING()

	for _, perm := range perms {
		stmt = stmt.VALUES(perm)
	}

	res, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		log.Printf("Error insertBulk perms: %v\n", err)
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func deleteById(id int32) (int64, error) {
	stmt := Permissions.DELETE().WHERE(Permissions.ID.EQ(sqlite.Int32(id)))
	res, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		log.Printf("Error deleteById perms: %v\n", err)
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func selectAll() ([]model.Permissions, error) {
	var dest []model.Permissions
	err := Permissions.SELECT(Permissions.AllColumns).Query(db.GetDB().DB, &dest)
	if err != nil {
		log.Printf("Error selectAll perms: %v\n", err)
		return nil, err
	}

	return dest, nil
}
