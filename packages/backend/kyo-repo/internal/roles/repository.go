package roles

import (
	"fmt"
	"github.com/go-jet/jet/v2/sqlite"
	"github.com/yaz/kyo-repo/internal/db"
	"github.com/yaz/kyo-repo/internal/db/gen/model"
	. "github.com/yaz/kyo-repo/internal/db/gen/table"
)

func getTotalCount() (int64, error) {

	var dest struct {
		Count int64
	}
	err := Roles.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(Roles).Query(db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}
	return dest.Count, nil
}

func queryCondition(requestQuery RequestQuery) *sqlite.BoolExpression {
	condition := sqlite.Bool(true)
	isThereAnyCondition := false

	if requestQuery.Q != "" {
		q := fmt.Sprintf("%%%s%%", requestQuery.Q)
		condition = condition.AND(Roles.Name.LIKE(sqlite.String(q)))
		isThereAnyCondition = true
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

	stmt := Roles.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(Roles).WHERE(*condition)

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

func selectAll() ([]struct {
	model.Roles
	Permissions []model.Permissions
}, error) {
	return selectList(RequestQuery{})
}

func selectList(requestQuery RequestQuery) ([]struct {
	model.Roles
	Permissions []model.Permissions
}, error) {
	condition := sqlite.Bool(true)

	if requestQuery.LastId > 0 {
		condition = condition.AND(Rates.ID.LT(sqlite.Int32(requestQuery.LastId)))
	}

	if commonQueryCondition := queryCondition(requestQuery); commonQueryCondition != nil {
		condition = condition.AND(*commonQueryCondition)
	}

	stmt := Roles.SELECT(
		Roles.AllColumns,
		Permissions.AllColumns,
	).FROM(
		Roles.LEFT_JOIN(RolePermissions, Roles.ID.EQ(RolePermissions.RoleID)).
			LEFT_JOIN(Permissions, RolePermissions.PermissionID.EQ(Permissions.ID)),
	).WHERE(condition).
		ORDER_BY(Roles.ID.ASC())

	//if requestQuery.Limit > 0 {
	//	stmt = stmt.LIMIT(int64(requestQuery.Limit))
	//}

	//log.Printf("selectList : %v\n", stmt.DebugSql())

	var dest []struct {
		model.Roles
		Permissions []model.Permissions
	}

	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}

func insert(role model.Roles) (int64, error) {
	stmt := Roles.INSERT(Roles.Name).VALUES(role.Name)

	res, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	lastId, err := res.LastInsertId()
	if err != nil {
		return 0, err
	}

	return lastId, nil
}

func insertPerms(roleId int32, perms []int32) (int64, error) {
	stmt := RolePermissions.INSERT(RolePermissions.RoleID, RolePermissions.PermissionID).
		ON_CONFLICT().DO_NOTHING()

	for _, perm := range perms {
		stmt = stmt.VALUES(roleId, perm)
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

func update(role model.Roles) (int64, error) {
	stmt := Roles.UPDATE(Roles.Name).SET(Roles.Name).
		WHERE(Roles.ID.EQ(sqlite.Int32(*role.ID))).
		SET(role.Name)

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

func deleteById(id int32) (int64, error) {
	res, err := RolePermissions.DELETE().WHERE(RolePermissions.RoleID.EQ(sqlite.Int32(id))).Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	stmt := Roles.DELETE().WHERE(Roles.ID.EQ(sqlite.Int32(id)))
	res, err = stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()

	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func deleteOnUpdate(roleId int32, perms []int32) (int64, error) {

	notInArray := make([]sqlite.Expression, len(perms))

	for i, perm := range perms {
		notInArray[i] = sqlite.Int32(perm)
	}

	stmt := RolePermissions.DELETE().WHERE(RolePermissions.RoleID.EQ(sqlite.Int32(roleId)).
		AND(RolePermissions.PermissionID.NOT_IN(notInArray...)))
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

func selectById(id int32) (*struct {
	model.Roles
	Permissions []model.Permissions
}, error) {
	stmt := Roles.SELECT(
		Roles.AllColumns,
		Permissions.AllColumns,
	).FROM(
		Roles.LEFT_JOIN(RolePermissions, Roles.ID.EQ(RolePermissions.RoleID)).
			LEFT_JOIN(Permissions, RolePermissions.PermissionID.EQ(Permissions.ID)),
	).WHERE(Roles.ID.EQ(sqlite.Int32(id))).
		ORDER_BY(Roles.ID.ASC())

	var dest struct {
		model.Roles
		Permissions []model.Permissions
	}

	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return &dest, nil
}
