package permissions

import (
	"context"
	"github.com/go-jet/jet/v2/sqlite"
	"github.com/yaz/kyo-repo/internal/db"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	. "github.com/yaz/kyo-repo/internal/db/gen/table"
	"log"
)

type Repository struct {
	ctx context.Context
}

func NewRepository(ctx context.Context) Repository {
	return Repository{ctx: ctx}
}

func (repo Repository) insertBulk(perms []string) (int64, error) {

	stmt := Permissions.INSERT(Permissions.Name).ON_CONFLICT().DO_NOTHING()

	for _, perm := range perms {
		stmt = stmt.VALUES(perm)
	}

	res, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
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

func (repo Repository) deleteById(id int32) (int64, error) {
	stmt := Permissions.DELETE().WHERE(Permissions.ID.EQ(sqlite.Int32(id)))
	res, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
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

func (repo Repository) selectAll() ([]model.Permissions, error) {
	var dest []model.Permissions
	err := Permissions.SELECT(Permissions.AllColumns).QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		log.Printf("Error selectAll perms: %v\n", err)
		return nil, err
	}

	return dest, nil
}
