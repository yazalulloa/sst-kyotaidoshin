package rates

import (
	"db"
	"db/gen/model"
	. "db/gen/table"
	"github.com/go-jet/jet/v2/sqlite"
	"kyotaidoshin/util"
	"time"
)

func queryCondition(rateQuery *RequestQuery) (sqlite.BoolExpression, bool) {
	condition := sqlite.Bool(true)
	justTrue := true

	if rateQuery.DateOfRate != nil {
		condition = condition.AND(Rates.DateOfRate.LT_EQ(sqlite.Date(rateQuery.DateOfRate.Date())))
		justTrue = false
	}

	if len(rateQuery.Currencies) > 0 {
		var sqlIDs []sqlite.Expression
		for _, str := range rateQuery.Currencies {
			sqlIDs = append(sqlIDs, sqlite.String(str))
		}

		condition = condition.AND(Rates.FromCurrency.IN(sqlIDs...))
		justTrue = false
	}

	return condition, justTrue
}

func SelectList(requestQuery RequestQuery) ([]model.Rates, error) {
	condition, _ := queryCondition(&requestQuery)

	if requestQuery.LastId > 0 {
		condition = condition.AND(Rates.ID.LT(sqlite.Int64(requestQuery.LastId)))
	}

	stmt := Rates.SELECT(Rates.AllColumns).FROM(Rates).WHERE(condition).LIMIT(int64(requestQuery.Limit))

	if requestQuery.SortOrder == util.SortOrderTypeASC {
		stmt = stmt.ORDER_BY(Rates.ID.ASC())
	} else {
		stmt = stmt.ORDER_BY(Rates.ID.DESC())
	}

	var list []model.Rates
	err := stmt.Query(db.GetDB().DB, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func getTotalCount() (int64, error) {
	var dest struct {
		Count int64
	}
	err := Rates.SELECT(
		//sqlite.COUNT(Rates.ID).
		sqlite.COUNT(sqlite.STAR).
			AS("Count")).FROM(Rates).Query(db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}
	return dest.Count, nil
}

func getQueryCount(rateQuery RequestQuery) (*int64, error) {
	condition, justTrue := queryCondition(&rateQuery)
	if justTrue {
		return nil, nil
	}

	stmt := Rates.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(Rates).WHERE(condition)
	var dest struct {
		Count int64
	}
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}
	return &dest.Count, nil
}

func CheckRateExist(id int64) (bool, error) {
	stmt := Rates.SELECT(Rates.ID.AS("ID")).FROM(Rates).
		WHERE(Rates.ID.EQ(sqlite.Int64(id)))

	var dest []struct {
		ID *int32
	}

	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {

		if util.ErrNoRows.Error() == err.Error() {
			return false, nil
		}

		return false, err
	}

	return len(dest) > 0, nil
}

func Insert(rates []model.Rates) (int64, error) {

	stmt := Rates.INSERT(Rates.ID, Rates.FromCurrency, Rates.ToCurrency, Rates.Rate, Rates.DateOfRate, Rates.Source,
		Rates.DateOfFile, Rates.Etag, Rates.LastModified).
		ON_CONFLICT().DO_NOTHING()
	//.MODELS(rates)
	for _, rate := range rates {
		stmt = stmt.VALUES(rate.ID, rate.FromCurrency, rate.ToCurrency, rate.Rate, rate.DateOfRate.Format(time.DateOnly), rate.Source, rate.DateOfFile, rate.Etag, rate.LastModified)
	}

	res, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	//id, err := res.LastInsertId()
	//if err != nil {
	//	return 0, err
	//}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func deleteRateById(id int64) (int64, error) {
	stmt := Rates.DELETE().WHERE(Rates.ID.EQ(sqlite.Int64(id)))
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

func GetFirstBeforeDate(fromCurrency string, date time.Time) (model.Rates, error) {

	stmt := Rates.SELECT(Rates.AllColumns).FROM(Rates).
		WHERE(Rates.FromCurrency.EQ(sqlite.String(fromCurrency)).AND(Rates.DateOfRate.LT_EQ(sqlite.Date(date.Date())))).
		ORDER_BY(Rates.DateOfRate.DESC()).LIMIT(1)

	var dest []model.Rates
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return model.Rates{}, err
	}

	if len(dest) == 0 {
		return model.Rates{}, util.ErrNoRows
	}

	return dest[0], nil

}

func GetFromDate(fromCurrency string, date time.Time, limit int64, isLt bool) ([]model.Rates, error) {

	condition := Rates.FromCurrency.EQ(sqlite.String(fromCurrency))

	if isLt {
		condition = condition.AND(Rates.DateOfRate.LT(sqlite.Date(date.Date())))
	} else {
		condition = condition.AND(Rates.DateOfRate.GT_EQ(sqlite.Date(date.Date())))
	}

	stmt := Rates.SELECT(Rates.AllColumns).FROM(Rates).
		WHERE(condition).LIMIT(limit)

	if isLt {
		stmt = stmt.ORDER_BY(Rates.ID.DESC())
	} else {
		stmt = stmt.ORDER_BY(Rates.ID.ASC())
	}

	var dest []model.Rates
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	if len(dest) == 0 {
		return nil, err
	}

	return dest, nil

}
