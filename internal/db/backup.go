package db

import (
	"time"

	"github.com/alist-org/alist/v3/internal/model"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

func GetBackupFile(filePath string) (*model.Backup, error) {
	file := model.Backup{FilePath: filePath}
	if err := db.Where(file).Take(&file).Error; err != nil {
		return nil, err
	}
	return &file, nil
}

func UpdateBackupFile(f *model.Backup) error {
	return errors.WithStack(db.Save(f).Error)
}

func IsFileModified(filePath string, modified time.Time) (f *model.Backup, flag bool) {
	flag = true
	f, err := GetBackupFile(filePath)
	if err != nil {
		logrus.Error("查询备份文件信息失败", err)
		return
	}
	flag = !modified.Equal(f.LastModified)

	return f, flag
}
