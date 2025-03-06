package util

import (
	"fmt"
	"strings"
)

type BackupType string

const (
	APARTMENTS BackupType = "APARTMENTS"
	BUILDINGS  BackupType = "BUILDINGS"
	RECEIPTS   BackupType = "RECEIPTS"
)

func GetBackupTypes() []BackupType {
	return []BackupType{APARTMENTS, BUILDINGS, RECEIPTS}
}

func (receiver BackupType) Name() string {
	return string(receiver)
}

func (receiver BackupType) Is(str string) bool {
	return receiver.Name() == str
}

func (receiver BackupType) StartsWith(str string) bool {
	return strings.HasPrefix(str, receiver.Name())
}

func GetBackupTypeStartsWith(str string) (BackupType, error) {

	for _, v := range GetBackupTypes() {
		if v.StartsWith(str) {
			return v, nil
		}
	}

	return "", fmt.Errorf("backup type not allowed: %s", str)
}
