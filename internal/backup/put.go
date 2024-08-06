package backup

import (
	"crypto/md5"
	"fmt"
	"os"

	"github.com/alist-org/alist/v3/internal/errs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/internal/stream"

	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	"github.com/xhofe/tache"
)

type BackupTask struct {
	tache.Base
	File string `json:"file"`
	// Storage          driver.Driver `json:"storage"`
	DstDir string `json:"dst_dir"`
}

func (t *BackupTask) GetName() string {
	return fmt.Sprintf("upload %s to (%s)", t.File, t.DstDir)
}

func (t *BackupTask) GetStatus() string {
	return ""
}

func (t *BackupTask) Run() error {
	fi, err := os.Stat(t.File)
	if err != nil {
		logrus.Error(errors.WithStack(err))
		return err
	}
	fd, err := os.Open(t.File)
	if err != nil {
		logrus.Error(errors.WithStack(err))
		return err
	}
	s := &stream.FileStream{
		Reader: fd,
		Obj: &model.Object{
			Name:     fi.Name(),
			Size:     fi.Size(),
			Modified: fi.ModTime(),
		},
	}
	storage, dstDirActualPath, err := op.GetStorageAndActualPath(t.DstDir)
	if err != nil {
		logrus.Error(errors.WithStack(err))
		return errors.WithMessage(err, "failed get storage")
	}
	logrus.Infof("backup task running, upload %s to %s, actualPath: %s +++++++++", t.File, t.DstDir, dstDirActualPath)

	return op.Put(t.Ctx(), storage, dstDirActualPath, s, t.SetProgress, true)
}

var BackupTaskManager *tache.Manager[*BackupTask]

// putAsTask add as a put task and return immediately
func putAsTask(file string, dstDirPath string) (tache.TaskWithInfo, error) {
	storage, _, err := op.GetStorageAndActualPath(dstDirPath)
	if err != nil {
		return nil, errors.WithMessage(err, "failed get storage")
	}
	if storage.Config().NoUpload {
		return nil, errors.WithStack(errs.UploadNotSupported)
	}

	t := &BackupTask{
		DstDir: dstDirPath,
		File:   file,
	}
	// logrus.Infof("upload %s to %s, actualPath: %s ------------------------", file, dstDirPath, actualPath)
	//设置任务id，避免重复创建任务
	t.SetID(fmt.Sprintf("%x", md5.Sum([]byte(file+dstDirPath))))
	BackupTaskManager.Add(t)
	return t, nil
}
