package apartments

import (
	"db"
	"db/gen/model"
	. "db/gen/table"
	"fmt"
	"github.com/go-jet/jet/v2/sqlite"
	"log"
	"strings"
)

func searchExpression(search string) sqlite.BoolExpression {
	q := fmt.Sprintf("%%%s%%", search)
	return sqlite.RawBool(
		"concat(apartments.building_id, apartments.number, apartments.name, apartments.emails) LIKE :search",
		sqlite.RawArgs{":search": q}).IS_TRUE()
}

func getTotalCount() (int64, error) {

	var dest struct {
		Count int64
	}
	err := Apartments.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(Apartments).Query(db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}
	return dest.Count, nil
}

func queryCondition(requestQuery RequestQuery) *sqlite.BoolExpression {
	condition := sqlite.Bool(true)
	isThereAnyCondition := false

	if requestQuery.q != "" {
		log.Printf("Search : %s\n", requestQuery.q)

		condition = condition.AND(searchExpression(requestQuery.q))
		isThereAnyCondition = true
	}

	if len(requestQuery.buildings) > 0 {
		log.Printf("Buildings : %v\n", requestQuery.buildings)
		var buildingIds []sqlite.Expression
		for _, buildingId := range requestQuery.buildings {
			buildingId = strings.TrimSpace(buildingId)
			if buildingId == "" {
				continue
			}

			buildingIds = append(buildingIds, sqlite.String(buildingId))
		}

		if len(buildingIds) > 0 {
			condition = condition.AND(Apartments.BuildingID.IN(buildingIds...))
			isThereAnyCondition = true
		}
	}

	if !isThereAnyCondition {
		return nil
	}

	return &condition
}

func getQueryCount(requestQuery RequestQuery) (*int64, error) {

	condition := queryCondition(requestQuery)
	if condition == nil {
		return nil, nil
	}

	stmt := Apartments.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(Apartments).WHERE(*condition)

	log.Printf("CountQuery : %v\n", stmt.DebugSql())
	var dest struct {
		Count int64
	}

	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return &dest.Count, nil
}

func selectList(requestQuery RequestQuery) ([]model.Apartments, error) {
	condition := sqlite.Bool(true)

	if requestQuery.lastBuildingId != "" && requestQuery.lastNumber != "" {
		condition = condition.AND(
			sqlite.RawBool(
				"(apartments.building_id,apartments.number) > (:LastBuildingId,:LastNumber)",
				sqlite.RawArgs{":LastBuildingId": requestQuery.lastBuildingId, ":LastNumber": requestQuery.lastNumber}).IS_TRUE(),
		)
	}

	commonQueryCondition := queryCondition(requestQuery)
	if commonQueryCondition != nil {
		condition = condition.AND(*commonQueryCondition)
	}

	stmt := Apartments.SELECT(Apartments.AllColumns).FROM(Apartments).WHERE(condition).
		GROUP_BY(Apartments.BuildingID, Apartments.Number).
		ORDER_BY(Apartments.BuildingID.ASC(), Apartments.Number.ASC()).
		LIMIT(int64(requestQuery.Limit))

	log.Printf("selectList : %v\n", stmt.DebugSql())

	var list []model.Apartments

	err := stmt.Query(db.GetDB().DB, &list)
	if err != nil {
		return nil, err
	}

	return list, nil
}

func deleteByKeys(keys Keys) (int64, error) {

	stmt := Apartments.DELETE().WHERE(Apartments.BuildingID.EQ(sqlite.String(keys.BuildingId)).
		AND(Apartments.Number.EQ(sqlite.String(keys.Number))))
	result, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func insertBulk(apartments []model.Apartments) (int64, error) {
	stmt := Apartments.INSERT(Apartments.BuildingID, Apartments.Number, Apartments.Name, Apartments.Aliquot, Apartments.Emails).
		ON_CONFLICT().DO_NOTHING()

	for _, apartment := range apartments {
		stmt = stmt.VALUES(apartment.BuildingID, apartment.Number, apartment.Name, apartment.Aliquot, apartment.Emails)
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
