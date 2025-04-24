package users

import (
	"context"
	"db"
	"db/gen/model"
	. "db/gen/table"
	"github.com/go-jet/jet/v2/sqlite"
	"kyotaidoshin/util"
)

type Repository struct {
	ctx context.Context
}

func NewRepository(ctx context.Context) Repository {
	return Repository{ctx: ctx}
}

func (repo Repository) GetByProvider(provider Provider, providerID string) (*model.Users, error) {

	stmt := Users.SELECT(Users.AllColumns).WHERE(Users.Provider.EQ(sqlite.String(provider.Name())).AND(Users.ProviderID.EQ(sqlite.String(providerID))))

	var dest model.Users
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)

	if err != nil {
		if util.ErrNoRows.Error() == err.Error() {
			return nil, nil
		}

		return nil, err
	}

	return &dest, nil
}

func (repo Repository) Insert(user model.Users) (int64, error) {
	stmt := Users.INSERT(Users.ID, Users.ProviderID, Users.Provider, Users.Email, Users.Username, Users.Name, Users.Picture, Users.Data).
		VALUES(user.ID, user.ProviderID, user.Provider, user.Email, user.Username, user.Name, user.Picture, user.Data)

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

func (repo Repository) UpdateWithLogin(user model.Users) (int64, error) {
	stmt := Users.UPDATE(Users.LastLoginAt, Users.Email, Users.Username, Users.Name, Users.Picture, Users.Data).
		SET(sqlite.DATETIME("now"), user.Email, user.Username, user.Name, user.Picture, user.Data).
		WHERE(Users.ID.EQ(sqlite.String(user.ID)))

	res, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}

func (repo Repository) GetByID(id string) (*model.Users, error) {
	stmt := Users.SELECT(Users.AllColumns).WHERE(Users.ID.EQ(sqlite.String(id)))
	var dest model.Users
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return &dest, nil
}

func (repo Repository) deleteById(id string) (int64, error) {

	DB := db.GetDB().DB

	_, err := UserRoles.DELETE().WHERE(UserRoles.UserID.EQ(sqlite.String(id))).ExecContext(repo.ctx, DB)
	if err != nil {
		return 0, err
	}

	_, err = TelegramChats.DELETE().WHERE(TelegramChats.UserID.EQ(sqlite.String(id))).ExecContext(repo.ctx, DB)
	if err != nil {
		return 0, err
	}

	res, err := Users.DELETE().WHERE(Users.ID.EQ(sqlite.String(id))).ExecContext(repo.ctx, DB)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := res.RowsAffected()

	if err != nil {
		return 0, err
	}

	return rowsAffected, nil
}

func (repo Repository) getWitRole(id string) (*struct {
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
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return &dest, nil
}

func (repo Repository) selectList(requestQuery RequestQuery) ([]struct {
	model.Users
	Chat *model.TelegramChats
	Role *model.Roles
}, error) {
	condition := sqlite.Bool(true)

	if requestQuery.LastId != "" {
		condition = condition.AND(Users.ID.GT_EQ(sqlite.String(requestQuery.LastId)))
	}

	stmt := Users.SELECT(Users.AllColumns, Roles.AllColumns, TelegramChats.AllColumns).
		FROM(
			Users.LEFT_JOIN(UserRoles, Users.ID.EQ(UserRoles.UserID)).
				LEFT_JOIN(Roles, UserRoles.RoleID.EQ(Roles.ID)).
				LEFT_JOIN(TelegramChats, Users.ID.EQ(TelegramChats.UserID)),
		).
		WHERE(condition).LIMIT(int64(requestQuery.Limit))

	var dest []struct {
		model.Users
		Chat *model.TelegramChats
		Role *model.Roles
	}

	if requestQuery.SortOrder == util.SortOrderTypeASC {
		stmt = stmt.ORDER_BY(Users.ID.ASC())
	} else {
		stmt = stmt.ORDER_BY(Users.ID.DESC())
	}

	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil

}

func (repo Repository) getTotalCount() (int64, error) {
	var dest struct {
		Count int64
	}
	err := Users.SELECT(sqlite.COUNT(sqlite.STAR).AS("Count")).FROM(Users).QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return 0, err
	}
	return dest.Count, nil
}

func (repo Repository) getQueryCount(requestQuery RequestQuery) (*int64, error) {
	return nil, nil
}

func (repo Repository) insertUserRole(id string, roleId int32) (int64, error) {

	stmt := UserRoles.INSERT(UserRoles.UserID, UserRoles.RoleID).
		ON_CONFLICT().DO_NOTHING().VALUES(id, roleId)

	res, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}

func (repo Repository) deleteUserRole(id string, roleId *int32) (int64, error) {
	condition := UserRoles.UserID.EQ(sqlite.String(id))
	if roleId != nil {
		condition = condition.AND(UserRoles.RoleID.NOT_EQ(sqlite.Int32(*roleId)))

	}

	stmt := UserRoles.DELETE().WHERE(condition)

	res, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()

}

func (repo Repository) UpdateTelegramChat(id string, chatId int64, username, firstName, lastName string) (int64, error) {

	stmt := TelegramChats.INSERT(TelegramChats.ChatID, TelegramChats.UserID, TelegramChats.Username, TelegramChats.FirstName, TelegramChats.LastName).
		ON_CONFLICT().
		DO_UPDATE(
			sqlite.SET(
				TelegramChats.Username.SET(sqlite.String(username)),
				TelegramChats.FirstName.SET(sqlite.String(firstName)),
				TelegramChats.LastName.SET(sqlite.String(lastName)),
			),
		).VALUES(chatId, id, username, firstName, lastName)

	res, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}

func (repo Repository) updateNotificationEvents(id, events string) (int64, error) {
	stmt := Users.UPDATE(Users.NotificationEvents).
		SET(sqlite.String(events)).
		WHERE(Users.ID.EQ(sqlite.String(id)))

	res, err := stmt.ExecContext(repo.ctx, db.GetDB().DB)
	if err != nil {
		return 0, err
	}

	return res.RowsAffected()
}

func (repo Repository) GetTelegramIdsByNotificationEvent(event EventNotifications) ([]int64, error) {

	stmt := TelegramChats.SELECT(TelegramChats.ChatID).
		FROM(
			TelegramChats.LEFT_JOIN(Users, TelegramChats.UserID.EQ(Users.ID)),
		).WHERE(Users.NotificationEvents.LIKE(sqlite.String("%" + event.Name() + "%")))

	var dest []int64
	err := stmt.QueryContext(repo.ctx, db.GetDB().DB, &dest)
	if err != nil {
		return nil, err
	}

	return dest, nil
}
