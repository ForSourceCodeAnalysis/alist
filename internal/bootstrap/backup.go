package bootstrap

import (
	"github.com/alist-org/alist/v3/internal/backup"
)

func InitBackup() {
	backup.BackupInit()
}
