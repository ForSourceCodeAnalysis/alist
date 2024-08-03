package bootstrap

import (
	"time"

	"github.com/alist-org/alist/v3/internal/backup"
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/fs"
	"github.com/alist-org/alist/v3/internal/offline_download/tool"
	"github.com/xhofe/tache"
)

func InitTaskManager() {
	tasks := conf.Conf.Tasks
	fs.UploadTaskManager = tache.NewManager[*fs.UploadTask](tache.WithWorks(tasks.Upload.Workers), tache.WithMaxRetry(tasks.Upload.MaxRetry))
	fs.CopyTaskManager = tache.NewManager[*fs.CopyTask](tache.WithWorks(tasks.Copy.Workers), tache.WithMaxRetry(tasks.Copy.MaxRetry))
	tool.DownloadTaskManager = tache.NewManager[*tool.DownloadTask](tache.WithWorks(tasks.Download.Workers), tache.WithMaxRetry(tasks.Download.MaxRetry))
	tool.TransferTaskManager = tache.NewManager[*tool.TransferTask](tache.WithWorks(tasks.Transfer.Workers), tache.WithMaxRetry(tasks.Transfer.MaxRetry))
	backup.BackupTaskManager = tache.NewManager[*backup.BackupTask](tache.WithWorks(tasks.Backup.Workers),
		tache.WithMaxRetry(tasks.Backup.MaxRetry),
		tache.WithPersistPath(tasks.Backup.PersistPath),
		tache.WithPersistDebounce(tasks.Backup.PersistDebounce*time.Second))
}
