package db

import (
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/pkg/errors"
)

func GetBackup(pageIndex, pageSize int) ([]model.Backup, int64, error) {
	tdb := db.Model(&model.Backup{})
	var count int64
	wh := map[string]string{"server_id": conf.Conf.ServerId}
	if err := tdb.Where(wh).Count(&count).Error; err != nil {
		return nil, 0, errors.Wrapf(err, "failed get storages count")
	}
	var ts []model.Backup
	if err := tdb.Where(wh).Offset((pageIndex - 1) * pageSize).Limit(pageSize).Find(&ts).Error; err != nil {
		return nil, 0, errors.WithStack(err)
	}
	return ts, count, nil
}
func GetBackupById(id uint) (*model.Backup, error) {
	var b model.Backup
	if err := db.First(&b, id).Error; err != nil {
		return nil, errors.Wrapf(err, "failed get old user")
	}
	return &b, nil
}

func CreateBackup(t *model.Backup) error {
	return errors.WithStack(db.Create(t).Error)
}
func UpdateBackup(b *model.Backup) error {
	return errors.WithStack(db.Save(b).Error)
}

func GetServerBackup() ([]model.Backup, error) {
	var ts []model.Backup
	if err := db.Where(map[string]string{"server_id": conf.Conf.ServerId}).Find(&ts).Error; err != nil {
		return nil, errors.WithStack(err)
	}
	return ts, nil
}

func DeleteBackupById(id uint) error {
	return errors.WithStack(db.Delete(&model.Backup{}, id).Error)
}
