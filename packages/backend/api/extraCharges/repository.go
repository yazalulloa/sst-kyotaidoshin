package extraCharges

import (
	"db"
	"db/gen/model"
	. "db/gen/table"
	"fmt"
	"github.com/go-jet/jet/v2/sqlite"
	"log"
)

func selectById(id int32) (*model.ExtraCharges, error) {

	stmt := ExtraCharges.SELECT(ExtraCharges.AllColumns).FROM(ExtraCharges).WHERE(ExtraCharges.ID.EQ(sqlite.Int32(id)))
	var dest model.ExtraCharges
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return &dest, nil
}

func selectByBuilding(buildingId string) ([]model.ExtraCharges, error) {

	stmt := ExtraCharges.SELECT(ExtraCharges.AllColumns).FROM(ExtraCharges).
		WHERE(ExtraCharges.BuildingID.EQ(sqlite.String(buildingId)).AND(ExtraCharges.ParentReference.EQ(sqlite.String(buildingId))))

	var dest []model.ExtraCharges
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}

func countByBuilding(buildingId string) (int64, error) {
	stmt := ExtraCharges.SELECT(sqlite.COUNT(ExtraCharges.ID).AS("Count")).FROM(ExtraCharges).
		WHERE(ExtraCharges.BuildingID.EQ(sqlite.String(buildingId)).AND(ExtraCharges.ParentReference.EQ(sqlite.String(buildingId))))

	var dest struct {
		Count int64
	}
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}

	return dest.Count, nil
}

func deleteById(id int32) (int64, error) {
	stmt := ExtraCharges.DELETE().WHERE(ExtraCharges.ID.EQ(sqlite.Int32(id)))
	res, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func InsertBackup(array []model.ExtraCharges) (int64, error) {
	if len(array) == 0 {
		return 0, nil
	}

	stmt := ExtraCharges.INSERT(ExtraCharges.BuildingID, ExtraCharges.ParentReference, ExtraCharges.Type, ExtraCharges.Description, ExtraCharges.Amount, ExtraCharges.Currency, ExtraCharges.Active, ExtraCharges.Apartments)

	for _, extraCharge := range array {
		stmt = stmt.VALUES(extraCharge.BuildingID, extraCharge.ParentReference, extraCharge.Type, extraCharge.Description, extraCharge.Amount, extraCharge.Currency, extraCharge.Active, extraCharge.Apartments)
	}

	result, err := stmt.Exec(db.GetDB().DB)
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

func insert(extraCharge model.ExtraCharges) (int64, error) {
	stmt := ExtraCharges.INSERT(ExtraCharges.BuildingID, ExtraCharges.ParentReference, ExtraCharges.Type, ExtraCharges.Description, ExtraCharges.Amount, ExtraCharges.Currency, ExtraCharges.Active, ExtraCharges.Apartments).
		VALUES(extraCharge.BuildingID, extraCharge.ParentReference, extraCharge.Type, extraCharge.Description, extraCharge.Amount, extraCharge.Currency, extraCharge.Active, extraCharge.Apartments)

	result, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	lastInsertId, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastInsertId, nil
}

func update(extraCharge model.ExtraCharges) (int64, error) {
	stmt := ExtraCharges.UPDATE(ExtraCharges.Description, ExtraCharges.Amount, ExtraCharges.Currency, ExtraCharges.Active, ExtraCharges.Apartments).
		WHERE(ExtraCharges.ID.EQ(sqlite.Int32(*extraCharge.ID))).
		SET(extraCharge.Description, extraCharge.Amount, extraCharge.Currency, extraCharge.Active, extraCharge.Apartments)
	result, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func SelectByReceipt(receiptID int32) ([]model.ExtraCharges, error) {
	//receiptID to string
	parentReference := fmt.Sprint(receiptID)
	stmt := ExtraCharges.SELECT(ExtraCharges.AllColumns).WHERE(ExtraCharges.ParentReference.EQ(sqlite.String(parentReference)))
	var dest []model.ExtraCharges
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}
	return dest, nil
}

func DeleteByReceipt(receiptID int32) (int64, error) {
	//receiptID to string
	parentReference := fmt.Sprint(receiptID)
	stmt := ExtraCharges.DELETE().WHERE(ExtraCharges.ParentReference.EQ(sqlite.String(parentReference)))
	res, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	return rowsAffected, nil
}
