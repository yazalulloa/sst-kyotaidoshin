package receipts

import (
	"db"
	"db/gen/model"
	. "db/gen/table"
	"github.com/go-jet/jet/v2/sqlite"
	"strings"
)

func getTotalCount() (int64, error) {
	var dest struct {
		Count int64
	}
	err := Receipts.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(Receipts).Query(db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}
	return dest.Count, nil
}

func InsertBulk(receipt []model.Receipts) (int64, error) {
	stmt := Receipts.INSERT(Receipts.ID, Receipts.BuildingID, Receipts.Year, Receipts.Month, Receipts.Date, Receipts.RateID, Receipts.Sent, Receipts.LastSent, Receipts.CreatedAt)

	for _, r := range receipt {
		stmt = stmt.VALUES(r.ID, r.BuildingID, r.Year, r.Month, r.Date, r.RateID, r.Sent, r.LastSent, r.CreatedAt)
	}

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

func getQueryCount(requestQuery RequestQuery) (*int64, error) {
	condition := queryCondition(requestQuery)
	if condition == nil {
		return nil, nil
	}

	stmt := Receipts.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(Receipts).WHERE(*condition)

	//log.Printf("CountQuery : %v\n", stmt.DebugSql())
	var dest struct {
		Count int64
	}

	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return &dest.Count, nil
}

func queryCondition(requestQuery RequestQuery) *sqlite.BoolExpression {
	condition := sqlite.Bool(true)
	isThereAnyCondition := false

	if len(requestQuery.Buildings) > 0 {
		//log.Printf("Buildings : %v\n", requestQuery.buildings)
		var buildingIds []sqlite.Expression
		for _, buildingId := range requestQuery.Buildings {
			buildingId = strings.TrimSpace(buildingId)
			if buildingId == "" {
				continue
			}

			buildingIds = append(buildingIds, sqlite.String(buildingId))
		}

		if len(buildingIds) > 0 {
			condition = condition.AND(Receipts.BuildingID.IN(buildingIds...))
			isThereAnyCondition = true
		}
	}

	if len(requestQuery.Months) > 0 {
		var months []sqlite.Expression
		for _, month := range requestQuery.Months {
			months = append(months, sqlite.Int16(month))
		}
		condition = condition.AND(Receipts.Month.IN(months...))
		isThereAnyCondition = true
	}

	if len(requestQuery.Years) > 0 {
		var years []sqlite.Expression
		for _, year := range requestQuery.Years {
			years = append(years, sqlite.Int16(year))
		}
		condition = condition.AND(Receipts.Year.IN(years...))
		isThereAnyCondition = true
	}

	if !isThereAnyCondition {
		return nil
	}

	return &condition
}

func selectList(requestQuery RequestQuery) ([]model.Receipts, error) {
	condition := sqlite.Bool(true)

	if requestQuery.LastId != "" {
		condition = condition.AND(Receipts.ID.LT(sqlite.String(requestQuery.LastId)))
	}

	commonQueryCondition := queryCondition(requestQuery)
	if commonQueryCondition != nil {
		condition = condition.AND(*commonQueryCondition)
	}

	stmt := Receipts.SELECT(Receipts.AllColumns).FROM(Receipts).WHERE(condition).
		ORDER_BY(Receipts.ID.DESC()).
		LIMIT(requestQuery.Limit)

	var list []model.Receipts

	err := stmt.Query(db.GetDB().DB, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func selectYears() ([]int16, error) {
	stmt := Receipts.SELECT(Receipts.Year).DISTINCT().FROM(Receipts)

	var years []int16
	err := stmt.Query(db.GetDB().DB, &years)
	if err != nil {
		return nil, err
	}

	return years, nil
}

func selectById(id string) (*model.Receipts, error) {
	stmt := Receipts.SELECT(Receipts.AllColumns).FROM(Receipts).WHERE(Receipts.ID.EQ(sqlite.String(id)))

	var dest model.Receipts
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return &dest, nil
}

func selectByIdWithRate(id string) (*struct {
	model.Receipts
	model.Rates
}, error) {

	var dest struct {
		model.Receipts
		model.Rates
	}

	stmt := Receipts.SELECT(Receipts.AllColumns, Rates.AllColumns).
		FROM(Receipts.LEFT_JOIN(Rates, Receipts.RateID.EQ(Rates.ID))).WHERE(Receipts.ID.EQ(sqlite.String(id)))

	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return &dest, nil
}

func update(receipt model.Receipts) (int64, error) {
	stmt := Receipts.UPDATE(Receipts.Year, Receipts.Month, Receipts.Date, Receipts.RateID).
		WHERE(Receipts.ID.EQ(sqlite.String(receipt.ID))).
		SET(receipt.Year, receipt.Month, receipt.Date, receipt.RateID)

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

func UpdateLastSent(id string) (int64, error) {
	stmt := Receipts.UPDATE(Receipts.Sent, Receipts.LastSent).WHERE(Receipts.ID.EQ(sqlite.String(id))).
		SET(sqlite.Bool(true), sqlite.DATETIME("now"))

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

func deleteById(id string) (int64, error) {
	stmt := Receipts.DELETE().WHERE(Receipts.ID.EQ(sqlite.String(id)))

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

func insert(receipt model.Receipts) (int64, error) {
	stmt := Receipts.INSERT(Receipts.ID, Receipts.BuildingID, Receipts.Year, Receipts.Month, Receipts.Date, Receipts.RateID, Receipts.Sent).
		VALUES(receipt.ID, receipt.BuildingID, receipt.Year, receipt.Month, receipt.Date, receipt.RateID, receipt.Sent)

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
