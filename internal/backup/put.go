package backup

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/alist-org/alist/v3/internal/driver"
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
	File             string        `json:"file"`
	Storage          driver.Driver `json:"storage"`
	DstDirActualPath string        `json:"dst_dir_actual_path"`
}

func (t *BackupTask) GetName() string {
	return fmt.Sprintf("upload %s to [%s](%s)", t.File, t.Storage.GetStorage().MountPath, t.DstDirActualPath)
}

func (t *BackupTask) GetStatus() string {
	return "uploading"
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

	return op.Put(t.Ctx(), t.Storage, t.DstDirActualPath, s, t.SetProgress, true)
}

var BackupTaskManager *tache.Manager[*BackupTask]

// putAsTask add as a put task and return immediately
func putAsTask(file string, dstDirPath string) (tache.TaskWithInfo, error) {
	storage, dstDirActualPath, err := op.GetStorageAndActualPath(dstDirPath)
	if err != nil {
		return nil, errors.WithMessage(err, "failed get storage")
	}
	if storage.Config().NoUpload {
		return nil, errors.WithStack(errs.UploadNotSupported)
	}

	t := &BackupTask{
		Storage:          storage,
		DstDirActualPath: dstDirActualPath,
		File:             file,
	}
	//设置任务id，避免重复创建任务
	t.SetID(file + filepath.Join(storage.GetStorage().MountPath, dstDirActualPath))
	BackupTaskManager.Add(t)
	return t, nil
}
