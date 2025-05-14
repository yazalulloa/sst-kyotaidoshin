package extraCharges

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

func (repo Repository) selectById(id int32) (*model.ExtraCharges, error) {

	stmt := ExtraCharges.SELECT(ExtraCharges.AllColumns).FROM(ExtraCharges).WHERE(ExtraCharges.ID.EQ(sqlite.Int32(id)))
	var dest model.ExtraCharges
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return &dest, nil
}

func (repo Repository) SelectByBuilding(buildingId string) ([]model.ExtraCharges, error) {

	stmt := ExtraCharges.SELECT(ExtraCharges.AllColumns).FROM(ExtraCharges).
		WHERE(ExtraCharges.BuildingID.EQ(sqlite.String(buildingId)).
			AND(ExtraCharges.ParentReference.EQ(sqlite.String(buildingId))))

	var dest []model.ExtraCharges
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}

func (repo Repository) countByBuilding(buildingId string) (int64, error) {
	stmt := ExtraCharges.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(ExtraCharges).
		WHERE(ExtraCharges.BuildingID.EQ(sqlite.String(buildingId)).AND(ExtraCharges.ParentReference.EQ(sqlite.String(buildingId))))

	var dest struct {
		Count int64
	}
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}

	return dest.Count, nil
}

func (repo Repository) deleteById(id int32) (int64, error) {
	stmt := ExtraCharges.DELETE().WHERE(ExtraCharges.ID.EQ(sqlite.Int32(id)))
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

func (repo Repository) DeleteByBuilding(buildingId string) (int64, error) {
	stmt := ExtraCharges.DELETE().WHERE(ExtraCharges.BuildingID.EQ(sqlite.String(buildingId)).
		AND(ExtraCharges.ParentReference.EQ(sqlite.String(buildingId))))
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

func (repo Repository) InsertBulk(array []model.ExtraCharges) (int64, error) {
	if len(array) == 0 {
		return 0, nil
	}

	stmt := ExtraCharges.INSERT(ExtraCharges.BuildingID, ExtraCharges.ParentReference, ExtraCharges.Type, ExtraCharges.Description, ExtraCharges.Amount, ExtraCharges.Currency, ExtraCharges.Active, ExtraCharges.Apartments)

	for _, extraCharge := range array {
		stmt = stmt.VALUES(extraCharge.BuildingID, extraCharge.ParentReference, extraCharge.Type, extraCharge.Description, extraCharge.Amount, extraCharge.Currency, extraCharge.Active, extraCharge.Apartments)
	}

	result, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		log.Printf("Error inserting extra charges: %s\n%v\n", stmt.DebugSql(), err)
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func (repo Repository) insert(extraCharge model.ExtraCharges) (int64, error) {
	stmt := ExtraCharges.INSERT(ExtraCharges.BuildingID, ExtraCharges.ParentReference, ExtraCharges.Type, ExtraCharges.Description, ExtraCharges.Amount, ExtraCharges.Currency, ExtraCharges.Active, ExtraCharges.Apartments).
		VALUES(extraCharge.BuildingID, extraCharge.ParentReference, extraCharge.Type, extraCharge.Description, extraCharge.Amount, extraCharge.Currency, extraCharge.Active, extraCharge.Apartments)

	result, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	lastInsertId, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastInsertId, nil
}

func (repo Repository) update(extraCharge model.ExtraCharges) (int64, error) {
	stmt := ExtraCharges.UPDATE(ExtraCharges.Description, ExtraCharges.Amount, ExtraCharges.Currency, ExtraCharges.Active, ExtraCharges.Apartments).
		WHERE(ExtraCharges.ID.EQ(sqlite.Int32(*extraCharge.ID))).
		SET(extraCharge.Description, extraCharge.Amount, extraCharge.Currency, extraCharge.Active, extraCharge.Apartments)
	result, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func (repo Repository) SelectByReceipt(receiptID string) ([]model.ExtraCharges, error) {

	stmt := ExtraCharges.SELECT(ExtraCharges.AllColumns).WHERE(ExtraCharges.ParentReference.EQ(sqlite.String(receiptID)))
	var dest []model.ExtraCharges
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}
	return dest, nil
}

func (repo Repository) SelectByReceipts(ids []string) ([]model.ExtraCharges, error) {
	receipts := make([]sqlite.Expression, len(ids))
	for i, p := range ids {
		receipts[i] = sqlite.String(p)
	}

	stmt := ExtraCharges.SELECT(ExtraCharges.AllColumns).WHERE(ExtraCharges.Type.EQ(sqlite.String(TypeReceipt)).
		AND(ExtraCharges.ParentReference.IN(receipts...)))

	var dest []model.ExtraCharges
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}
	return dest, nil
}

func (repo Repository) DeleteByReceipt(receiptID string) (int64, error) {

	stmt := ExtraCharges.DELETE().WHERE(ExtraCharges.ParentReference.EQ(sqlite.String(receiptID)))
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
