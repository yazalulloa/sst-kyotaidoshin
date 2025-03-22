package api

import "slices"

type PERM string

func (receiver PERM) Name() string {
	return string(receiver)
}

func (receiver PERM) Is(str string) bool {
	return receiver.Name() == str
}

const (
	APARTMENTS_READ            PERM = "apartments:read"
	APARTMENTS_WRITE           PERM = "apartments:write"
	APARTMENTS_UPLOAD_BACKUP   PERM = "apartments:upload_backup"
	APARTMENTS_DOWNLOAD_BACKUP PERM = "apartments:download_backup"

	BUILDINGS_READ            PERM = "buildings:read"
	BUILDINGS_WRITE           PERM = "buildings:write"
	BUILDINGS_UPLOAD_BACKUP   PERM = "buildings:upload_backup"
	BUILDINGS_DOWNLOAD_BACKUP PERM = "buildings:download_backup"

	RECEIPTS_READ            PERM = "receipts:read"
	RECEIPTS_WRITE           PERM = "receipts:write"
	RECEIPTS_UPLOAD_BACKUP   PERM = "receipts:upload_backup"
	RECEIPTS_DOWNLOAD_BACKUP PERM = "receipts:download_backup"
	RECEIPTS_DELETE_PDFS     PERM = "receipts:delete_pdfs"

	USERS_READ  PERM = "users:read"
	USERS_WRITE PERM = "users:write"

	RATES_READ  PERM = "rates:read"
	RATES_WRITE PERM = "rates:write"

	BCV_FILES_READ  PERM = "bcv_files:read"
	BCV_FILES_WRITE PERM = "bcv_files:write"

	PERMISSIONS_READ  PERM = "permissions:read"
	PERMISSIONS_WRITE PERM = "permissions:write"

	ROLES_READ  PERM = "roles:read"
	ROLES_WRITE PERM = "roles:write"
)

func All() []PERM {

	return slices.Concat(
		AllApartments(),
		AllBuildings(),
		AllReceipts(),
		AllUsers(),
		AllRates(),
		AllBcvFiles(),
		AllPermissions(),
		AllRoles(),
	)
}

func AllApartments() []PERM {
	return []PERM{
		APARTMENTS_READ,
		APARTMENTS_WRITE,
		APARTMENTS_UPLOAD_BACKUP,
		APARTMENTS_DOWNLOAD_BACKUP,
	}
}

func AllBuildings() []PERM {
	return []PERM{
		BUILDINGS_READ,
		BUILDINGS_WRITE,
		BUILDINGS_UPLOAD_BACKUP,
		BUILDINGS_DOWNLOAD_BACKUP,
	}
}

func AllReceipts() []PERM {
	return []PERM{
		RECEIPTS_READ,
		RECEIPTS_WRITE,
		RECEIPTS_UPLOAD_BACKUP,
		RECEIPTS_DOWNLOAD_BACKUP,
		RECEIPTS_DELETE_PDFS,
	}
}

func AllUsers() []PERM {
	return []PERM{
		USERS_READ,
		USERS_WRITE,
	}
}

func AllRates() []PERM {
	return []PERM{
		RATES_READ,
		RATES_WRITE,
	}
}

func AllBcvFiles() []PERM {
	return []PERM{
		BCV_FILES_READ,
		BCV_FILES_WRITE,
	}
}

func AllPermissions() []PERM {
	return []PERM{
		PERMISSIONS_READ,
		PERMISSIONS_WRITE,
	}
}

func AllRoles() []PERM {
	return []PERM{
		ROLES_READ,
		ROLES_WRITE,
	}
}

type WithLabel struct {
	Label string
	Perms []PERM
}

func WithLabels() []WithLabel {
	return []WithLabel{
		{
			Label: "apartments",
			Perms: AllApartments(),
		},
		{
			Label: "buildings",
			Perms: AllBuildings(),
		},
		{
			Label: "receipts",
			Perms: AllReceipts(),
		},
		{
			Label: "users",
			Perms: AllUsers(),
		},
		{
			Label: "rates",
			Perms: AllRates(),
		},
		{
			Label: "bcv-files",
			Perms: AllBcvFiles(),
		},
		{
			Label: "permissions",
			Perms: AllPermissions(),
		},
		{
			Label: "roles",
			Perms: AllRoles(),
		},
	}
}
