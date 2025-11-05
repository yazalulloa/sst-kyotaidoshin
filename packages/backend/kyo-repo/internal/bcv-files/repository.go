package bcv_files

import (
	"context"

	"github.com/go-jet/jet/v2/sqlite"
	"github.com/yaz/kyo-repo/internal/db"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	. "github.com/yaz/kyo-repo/internal/db/gen/table"
	"github.com/yaz/kyo-repo/internal/util"
)

type Repository struct {
	ctx context.Context
}

func NewRepository(ctx context.Context) Repository {
	return Repository{ctx: ctx}
}

func (repo Repository) getTotalCount() (int64, error) {
	var dest struct {
		Count int64
	}
	err := BcvFiles.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(BcvFiles).QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}
	return dest.Count, nil
}

func (repo Repository) Get(link string) (*model.BcvFiles, error) {

	stmt := BcvFiles.SELECT(BcvFiles.AllColumns).FROM(BcvFiles).
		WHERE(BcvFiles.Link.EQ(sqlite.String(link)))

	var dest model.BcvFiles
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)

	if err != nil {
		if util.ErrNoRows.Error() == err.Error() {
			return nil, nil
		}

		return nil, err
	}

	return &dest, nil
}

func (repo Repository) selectAll() ([]model.BcvFiles, error) {
	stmt := BcvFiles.SELECT(BcvFiles.AllColumns).FROM(BcvFiles).ORDER_BY(BcvFiles.FileDate)

	var dest []model.BcvFiles
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}

func (repo Repository) Delete(link string) (int64, error) {
	stmt := BcvFiles.DELETE().WHERE(BcvFiles.Link.EQ(sqlite.String(link)))

	res, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func (repo Repository) Insert(bcvFile model.BcvFiles) (int64, error) {
	stmt := BcvFiles.INSERT(BcvFiles.Link, BcvFiles.RateCount, BcvFiles.SheetCount, BcvFiles.FileSize, BcvFiles.Etag, BcvFiles.LastModified, BcvFiles.FileDate, BcvFiles.ProcessedAt).
		VALUES(bcvFile.Link, bcvFile.RateCount, bcvFile.SheetCount, bcvFile.FileSize, bcvFile.Etag, bcvFile.LastModified, bcvFile.FileDate, bcvFile.ProcessedAt).
		ON_CONFLICT().DO_UPDATE(
		sqlite.SET(
			BcvFiles.RateCount.SET(BcvFiles.EXCLUDED.RateCount),
			BcvFiles.SheetCount.SET(BcvFiles.EXCLUDED.SheetCount),
			BcvFiles.FileSize.SET(BcvFiles.EXCLUDED.FileSize),
			BcvFiles.Etag.SET(BcvFiles.EXCLUDED.Etag),
			BcvFiles.LastModified.SET(BcvFiles.EXCLUDED.LastModified),
			BcvFiles.FileDate.SET(BcvFiles.EXCLUDED.FileDate),
			BcvFiles.ProcessedAt.SET(BcvFiles.EXCLUDED.ProcessedAt),
		),
	)

	res, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}
