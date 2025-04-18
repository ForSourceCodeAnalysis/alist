package backup

import (
	"log"
	"path/filepath"
	"time"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func initDB() {
	if err := db.GetDb().AutoMigrate(&Backup{}, &File{}); err != nil {
		log.Fatalf("failed migrate database: %s", err.Error())
	}
}
func getBackupsDB(pageIndex, pageSize int) ([]Backup, int64, error) {
	tdb := db.GetDb().Model(&Backup{})
	var count int64
	wh := map[string]string{"server_id": conf.Conf.ServerID}
	if err := tdb.Where(wh).Count(&count).Error; err != nil {
		return nil, 0, errors.Wrapf(err, "failed get storages count")
	}
	var ts []Backup
	if err := tdb.Where(wh).Offset((pageIndex - 1) * pageSize).Limit(pageSize).Find(&ts).Error; err != nil {
		return nil, 0, errors.WithStack(err)
	}
	return ts, count, nil
}
func getServerEnabledBackupDB() ([]Backup, error) {
	var ts []Backup
	if err := db.GetDb().Where("`disabled`=? AND  server_id=?", false, conf.Conf.ServerID).Find(&ts).Error; err != nil {
		return nil, errors.Wrapf(err, "failed get storages count")
	}
	return ts, nil
}
func getBackupByIDDB(id uint64) (*Backup, error) {
	var b Backup
	if err := db.GetDb().First(&b, id).Error; err != nil {
		return nil, errors.Wrapf(err, "failed get old user")
	}
	return &b, nil
}

func createBackupDB(t *Backup) error {
	return errors.WithStack(db.GetDb().Create(t).Error)
}
func updateBackupDB(b *Backup) error {
	return errors.WithStack(db.GetDb().Save(b).Error)
}

func checkBackupExistBySrc(src string) (bool, error) {
	var ts Backup
	if err := db.GetDb().Where(map[string]string{"server_id": conf.Conf.ServerID, "src": src}).First(&ts).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, errors.WithStack(err)
	}
	return true, nil
}

func deleteBackupByIDDB(id uint64) error {
	err := db.GetDb().Delete(&Backup{}, id).Error
	db.GetDb().Where("backup_id = ?", id).Delete(&File{})
	if err != nil {
		return errors.WithStack(err)
	}
	return nil
}

func saveFileDB(bt *File) error {
	return errors.WithStack(db.GetDb().Transaction(
		func(tx *gorm.DB) error {
			var existing File
			err := tx.Clauses(clause.Locking{
				Strength: "UPDATE "}).Where("backup_id = ? AND name = ? AND dir = ?", bt.BackupID, bt.Name, bt.Dir).First(&existing).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return tx.Create(bt).Error
			}
			return tx.Model(&File{}).Where("id=?", existing.ID).Update("last_modified_time", bt.LastModifiedTime).Error

		}))

}

func getLastModifiedTime(bid uint64) map[string]time.Time {
	m := make(map[string]time.Time)
	var t []File
	if err := db.GetDb().Where("backup_id = ?", bid).Find(&t).Error; err != nil {
		return m
	}

	for _, v := range t {

		m[filepath.Join(v.Dir, v.Name)] = v.LastModifiedTime

	}
	return m
}

func getBackupFilesDB(bid uint64, page, pageSize int) ([]File, int64, error) {
	tdb := db.GetDb().Model(&File{})
	var count int64

	if err := tdb.Where("backup_id=?", bid).Count(&count).Error; err != nil {
		return nil, 0, errors.Wrapf(err, "failed get last backup count")
	}
	var ts []File
	if err := tdb.Where("backup_id=?", bid).Offset((page - 1) * pageSize).Limit(pageSize).Find(&ts).Error; err != nil {
		return nil, 0, errors.WithStack(err)
	}
	return ts, count, nil
}
