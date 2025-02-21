package api

type UploadBackupParams struct {
	Url              string
	Values           map[string]string
	OutOfBandsUpdate bool
}
