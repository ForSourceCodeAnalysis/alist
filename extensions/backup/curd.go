package backup

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/op"

	"github.com/pkg/errors"
)

// 监听以文件夹为单位，不支持单个文件
func create(b *Backup) (uint64, error) {
	//判断源文件夹是否已经被监听
	if is, err := checkBackupExistBySrc(b.Src); err != nil || is {
		return 0, fmt.Errorf("src %s has already been watched", b.Src)
	}

	b.ServerID = conf.Conf.ServerID

	//写入数据库
	if err := createBackupDB(b); err != nil {
		return 0, errors.WithMessage(err, "failed create backup in database")
	}
	//加入监听
	addWatch(b)
	//事件监听才需要初始上传
	if b.InitUpload && b.Mode == modeEvent {
		initUpload(b.Src)
	}

	return b.ID, nil
}

func update(b *Backup) error {

	oldB, err := getBackupByIDDB(b.ID)
	if err != nil {
		return errors.WithMessage(err, "failed read old data, can not update ")
	}
	b.ServerID = conf.Conf.ServerID

	//写入数据库
	if err := updateBackupDB(b); err != nil {
		return errors.WithMessage(err, "failed create backup in database")
	}

	if oldB.Disabled != b.Disabled { //启用状态有变化
		removeWatch(oldB.Src) //尝试移除旧的监听
		if !b.Disabled {
			addWatch(b)
		}
		return nil
	} else if !b.Disabled && (oldB.Mode != b.Mode || oldB.Ignore != b.Ignore || oldB.Cron != b.Cron) { //启用状态没变，且是启用，其它内容改变了,
		removeWatch(oldB.Src)
		addWatch(b)
		return nil
	}
	//如果只是dst变化，不用调整监听，但是需要更新 backup
	if b.Dst != oldB.Dst {
		nb, ok := bts.Load(b.Src)
		if ok {
			nb.Dst = b.Dst
			bts.Store(b.Src, nb)
		}
	}

	return nil
}

func deleteByID(id uint64) error {
	m, err := getBackupByIDDB(id)
	if err != nil {
		return errors.WithMessage(err, "failed get backup in database")
	}
	if err := deleteBackupByIDDB(id); err != nil {
		return errors.WithMessage(err, "failed delete backup in database")
	}
	removeWatch(m.Src)
	bts.Delete(m.Src)
	return nil
}

func validate(b *Backup) error {
	b.Src = filepath.Clean(filepath.ToSlash(b.Src))
	fi, err := os.Stat(b.Src)
	if err != nil {
		return errors.WithMessage(err, "failed read src file/dir info")
	}
	if !fi.IsDir() {
		return errors.WithMessage(err, "src is not a dir")
	}
	dsts := strings.Split(b.Dst, ";")
	for k, v := range dsts {
		v = filepath.Clean(filepath.ToSlash(v))
		_, _, err := op.GetStorageAndActualPath(v)
		if err != nil {
			return errors.WithMessagef(err, "failed get directory: %s", v)
		}
		dsts[k] = v
	}
	b.Dst = strings.Join(dsts, ";")

	return nil
}
