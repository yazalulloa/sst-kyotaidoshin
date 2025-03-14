//
// Code generated by go-jet DO NOT EDIT.
//
// WARNING: Changes to this file may cause incorrect behavior
// and will be lost if the code is regenerated
//

package table

// UseSchema sets a new schema name for all generated table SQL builder types. It is recommended to invoke
// this method only once at the beginning of the program.
func UseSchema(schema string) {
	Apartments = Apartments.FromSchema(schema)
	Buildings = Buildings.FromSchema(schema)
	Debts = Debts.FromSchema(schema)
	Expenses = Expenses.FromSchema(schema)
	ExtraCharges = ExtraCharges.FromSchema(schema)
	Permissions = Permissions.FromSchema(schema)
	Rates = Rates.FromSchema(schema)
	Receipts = Receipts.FromSchema(schema)
	ReserveFunds = ReserveFunds.FromSchema(schema)
	RolePermissions = RolePermissions.FromSchema(schema)
	Roles = Roles.FromSchema(schema)
	UserRoles = UserRoles.FromSchema(schema)
	Users = Users.FromSchema(schema)
}
