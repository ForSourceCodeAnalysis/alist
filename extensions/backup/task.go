package backup

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"strings"
	"time"

	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/internal/stream"
	"github.com/alist-org/alist/v3/pkg/utils"
	hash_extend "github.com/alist-org/alist/v3/pkg/utils/hash"
	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

const taskTypeUpload = "backup upload"

type uploadFilePayload struct {
	Path string
	*Backup
}

func newBackupUploadTask(path string, b *Backup) (*asynq.Task, error) {

	payload, err := json.Marshal(uploadFilePayload{Path: path, Backup: b})
	if err != nil {
		logrus.Error(errors.WithStack(err))
		return nil, err
	}
	logrus.Info("payload marshal success " + string(payload))
	return asynq.NewTask(taskTypeUpload, payload), nil
}

func handleBackupUploadTask(ctx context.Context, t *asynq.Task) error {
	st := time.Now()
	var p uploadFilePayload
	if err := json.Unmarshal(t.Payload(), &p); err != nil {
		logrus.Error(t.Payload(), errors.WithStack(err))
		return err
	}
	info, err := os.Stat(p.Path)

	if err != nil {
		logrus.Error(p.Path, errors.WithStack(err))
		return err
	}
	f, err := os.Open(p.Path)
	if err != nil {
		logrus.Error(p, errors.WithStack(err))
		return err
	}
	defer f.Close()

	// 目标地址可能有多个
	dsts := strings.Split(p.Dst, ";")

	for _, v := range dsts {
		// 可能要上传多次，每次上传前，确保指针在开头
		f.Seek(0, io.SeekStart)
		storage, dstDirActualPath, err := op.GetStorageAndActualPath(v)
		if err != nil {
			logrus.Error(p, errors.WithStack(err))
			continue
		}
		driverName := storage.Config().Name
		h := getHash(driverName, f)

		s := &stream.FileStream{
			Obj: &model.Object{
				Name:     info.Name(),
				Size:     info.Size(),
				Modified: info.ModTime(),
				HashInfo: utils.NewHashInfoByMap(h),
			},
			Reader:       f,
			Mimetype:     "application/octet-stream",
			WebPutAsTask: false,
		}

		if err := op.Put(ctx, storage, dstDirActualPath, s, nil); err != nil {
			logrus.Error(p, errors.WithStack(err))
			continue
		}
	}
	tc := time.Since(st)
	saveLastBackupTime(info.ModTime(), p.Path, p.Backup.ID, tc)

	return nil
}

func getHash(driverName string, f *os.File) (hash map[*utils.HashType]string) {
	hash = make(map[*utils.HashType]string)
	switch driverName {
	case "115 Cloud":
	case "AliyundriveOpen":
	case "VTencent":
		if h, err := utils.HashFile(utils.SHA1, f); err == nil {
			hash[utils.SHA1] = h
		}
		break
	case "139Yun":
		if h, err := utils.HashFile(utils.SHA256, f); err == nil {
			hash[utils.SHA256] = h
		}
		break
	case "189CloudPC":
	case "BaiduNetdisk":
		if h, err := utils.HashFile(utils.MD5, f); err == nil {
			hash[utils.MD5] = h
		}
		break
	case "PikPak":
	case "ThunderBrowser":
	case "ThunderBrowserExpert":
		if h, err := utils.HashFile(hash_extend.GCID, f); err == nil {
			hash[hash_extend.GCID] = h
		}

	}
	return
}
