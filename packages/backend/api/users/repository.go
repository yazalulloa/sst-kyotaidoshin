package users

import (
	"db"
	"db/gen/model"
	. "db/gen/table"
	"github.com/go-jet/jet/v2/sqlite"
	"kyotaidoshin/util"
)

func GetByProvider(provider Provider, providerID string) (*model.Users, error) {

	stmt := Users.SELECT(Users.AllColumns).WHERE(Users.Provider.EQ(sqlite.String(provider.Name())).AND(Users.ProviderID.EQ(sqlite.String(providerID))))

	var dest model.Users
	err := stmt.Query(db.GetDB().DB, &dest)

	if err != nil {
		if util.ErrNoRows.Error() == err.Error() {
			return nil, nil
		}

		return nil, err
	}

	return &dest, nil
}

func Insert(user model.Users) (int64, error) {
	stmt := Users.INSERT(Users.ID, Users.ProviderID, Users.Provider, Users.Email, Users.Username, Users.Name, Users.Picture, Users.Data).
		VALUES(user.ID, user.ProviderID, user.Provider, user.Email, user.Username, user.Name, user.Picture, user.Data)

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

func UpdateLastLogin(id string) (int64, error) {
	stmt := Users.UPDATE(Users.LastLoginAt).
		SET(sqlite.DATETIME("now")).
		WHERE(Users.ID.EQ(sqlite.String(id)))

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

func GetByID(id string) (*model.Users, error) {
	stmt := Users.SELECT(Users.AllColumns).WHERE(Users.ID.EQ(sqlite.String(id)))
	var dest model.Users
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return &dest, nil
}

func deleteById(id string) (int64, error) {
	stmt := Users.DELETE().WHERE(Users.ID.EQ(sqlite.String(id)))
	res, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, nil
	}

	rowsAffected, err := res.RowsAffected()

	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func getWitRole(id string) (*struct {
	model.Users
	Role *model.Roles
}, error) {
	stmt := Users.SELECT(Users.AllColumns, Roles.AllColumns).FROM(
		Users.LEFT_JOIN(UserRoles, Users.ID.EQ(UserRoles.UserID)).
			LEFT_JOIN(Roles, UserRoles.RoleID.EQ(Roles.ID)),
	).WHERE(Users.ID.EQ(sqlite.String(id)))

	var dest struct {
		model.Users
		Role *model.Roles
	}
	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return &dest, nil
}

func selectList(requestQuery RequestQuery) ([]struct {
	model.Users
	Role *model.Roles
}, error) {
	condition := sqlite.Bool(true)

	if requestQuery.LastId != "" {
		condition = condition.AND(Users.ID.GT_EQ(sqlite.String(requestQuery.LastId)))
	}

	stmt := Users.SELECT(Users.AllColumns, Roles.AllColumns).
		FROM(
			Users.LEFT_JOIN(UserRoles, Users.ID.EQ(UserRoles.UserID)).
				LEFT_JOIN(Roles, UserRoles.RoleID.EQ(Roles.ID)),
		).
		WHERE(condition).LIMIT(int64(requestQuery.Limit))

	var dest []struct {
		model.Users
		Role *model.Roles
	}

	if requestQuery.SortOrder == util.SortOrderTypeASC {
		stmt = stmt.ORDER_BY(Users.ID.ASC())
	} else {
		stmt = stmt.ORDER_BY(Users.ID.DESC())
	}

	err := stmt.Query(db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil

}

func getTotalCount() (int64, error) {
	var dest struct {
		Count int64
	}
	err := Users.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(Users).Query(db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}
	return dest.Count, nil
}

func getQueryCount(requestQuery RequestQuery) (*int64, error) {
	return nil, nil
}

func insertUserRole(id string, roleId int32) (int64, error) {

	stmt := UserRoles.INSERT(UserRoles.UserID, UserRoles.RoleID).
		ON_CONFLICT().DO_NOTHING().VALUES(id, roleId)

	res, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}

func deleteUserRole(id string, roleId *int32) (int64, error) {
	condition := UserRoles.UserID.EQ(sqlite.String(id))
	if roleId != nil {
		condition = condition.AND(UserRoles.RoleID.NOT_EQ(sqlite.Int32(*roleId)))

	}

	stmt := UserRoles.DELETE().WHERE(condition)

	res, err := stmt.Exec(db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()

}
