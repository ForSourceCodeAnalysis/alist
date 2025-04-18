package backup

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alist-org/alist/v3/extensions/queue"
	"github.com/alist-org/alist/v3/pkg/generic_sync"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/fsnotify/fsnotify"
	"github.com/go-co-op/gocron/v2"
	"github.com/hibiken/asynq"
	"github.com/pkg/errors"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/sirupsen/logrus"
)

type backupT struct {
	*Backup
	*ignore.GitIgnore
	gocron.Job
}

var fsnotifyWatcher *fsnotify.Watcher
var cronSchedule gocron.Scheduler

var bts generic_sync.MapOf[string, *backupT]
var delayTimers generic_sync.MapOf[string, *time.Timer]

// Init 从数据库初始化备份配置
func Init() {
	initDB()
	queue.RegisterHandler(taskTypeUpload, handleBackupUploadTask)
	logrus.Info("register backup task handler success")
	// 先查询是否有任务
	bps, err := getServerEnabledBackupDB()
	if err != nil {
		logrus.Error(errors.WithStack(err))
		return
	}

	//add
	for _, bp := range bps {
		if bp.Disabled {
			continue
		}
		addWatch(&bp)
	}

}

func initScheduleAndWatcher() error {
	// 监听
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logrus.Error(errors.WithStack(err))
		return err
	}
	fsnotifyWatcher = watcher

	s, err := gocron.NewScheduler()
	if err != nil {
		logrus.Error(errors.WithStack(err))
		fsnotifyWatcher.Close()
		fsnotifyWatcher = nil
		return err
	}
	cronSchedule = s
	cronSchedule.Start()
	logrus.Info("init fsnotify and cron schedule success")
	// Start listening for events.
	go func() {
		defer fsnotifyWatcher.Close()
		for {
			select {
			case event, ok := <-fsnotifyWatcher.Events:
				if !ok {
					return
				}
				go eventDeal(event)

			case err, ok := <-fsnotifyWatcher.Errors:
				if !ok {
					return
				}
				logrus.Error(errors.WithStack(err))
			}
		}
	}()
	return nil
}

// 加入监听/创建定时任务
func addWatch(b *Backup) {
	// 检查文件（夹）是否存在
	fi, err := os.Stat(b.Src)
	if err != nil {
		logrus.Error(errors.WithStack(err))
		return
	}
	if fsnotifyWatcher == nil {
		if err := initScheduleAndWatcher(); err != nil {
			logrus.Error(errors.WithStack(err))
			return
		}
	}
	if !fi.IsDir() {
		logrus.Errorf("src %s is not a dir", b.Src)
		return
	}

	gi := compileGitignore(b.Ignore)
	bt := &backupT{
		Backup:    b,
		GitIgnore: gi,
	}
	bts.Store(b.Src, bt)

	//定时模式
	if b.Mode == modeCron {
		if b.Cron == "" {
			b.Cron = "0 * * * * "
		}
		job, err := cronSchedule.NewJob(
			gocron.CronJob(b.Cron, false),
			gocron.NewTask(func(b *backupT) {
				cronJob(b)
			}, bt),
			gocron.WithSingletonMode(gocron.LimitModeReschedule))
		if err != nil {
			logrus.Error(errors.WithStack(err))
			return
		}
		bt.Job = job
		bts.Store(b.Src, bt)

		return
	}
	// 监听模式
	watchOp(bt, "add")
}

func removeWatch(src string) {
	bt, ok := bts.Load(src)
	if !ok || bt.Disabled {
		return
	}
	if bt.Mode == modeEvent {
		watchOp(bt, "remove")
	} else if bt.Job != nil && len(bt.Job.ID()) > 0 {
		if err := cronSchedule.RemoveJob(bt.Job.ID()); err != nil {
			logrus.Error(errors.WithStack(err))
		}
	}
}
func watchOp(bt *backupT, op string) {
	// fsnotify不支持监听子文件夹，这里手动处理子文件夹，path是相对于src的路径，如果src是绝对路径，那么path也是绝对路径
	filepath.WalkDir(bt.Src, func(path string, d fs.DirEntry, err error) error {
		if err != nil && err != filepath.SkipDir {
			logrus.Error(errors.WithStack(err))
			return err
		}

		if d.IsDir() {
			if isIgnored(bt, path) {
				return filepath.SkipDir
			}
			var fsnerr error
			if op == "add" {
				fsnerr = fsnotifyWatcher.Add(path)
			} else {
				fsnerr = fsnotifyWatcher.Remove(path)
			}
			if fsnerr != nil {
				logrus.Error(errors.WithStack(fsnerr))
			}
			return nil
		}
		return nil
	})
}

// 监听事件处理
func eventDeal(event fsnotify.Event) {
	logrus.Infof("event trigger, file path: %v, op: %v", event.Name, event.Op.String())
	//只处理新增，修改
	if !event.Has(fsnotify.Create) && !event.Has(fsnotify.Write) {
		return
	}
	//event.Name是文件路径，如果Add时是绝对路径，那么这里的Name也是绝对路径
	info, err := os.Stat(event.Name)
	if err != nil {
		logrus.Error(errors.WithStack(err))
		return
	}
	//新创建的文件夹，加入监听
	if info.IsDir() {
		if !event.Has(fsnotify.Create) {
			return
		}
		if err := fsnotifyWatcher.Add(event.Name); err != nil {
			logrus.Error(errors.WithStack(err))
		}
		return
	}
	// 监听只返回了变动的文件，并没有返回是哪个任务，所以需要我们自己判断
	for _, bt := range bts.ToMap() {
		if !utils.IsSubPath(bt.Src, event.Name) {
			continue
		}
		if isIgnored(bt, event.Name) {
			return
		}
		// 加入上传队列
		addQueue(event.Name, bt.Backup)
		break
	}

}

// 同一个文件的同一个事件可能短时间内多次触发，比如复制一个文件到监听文件夹中，会首先触发一次Op.Create事件
// 随着复制进行，文件内容不断变化，会一直触发Op.Write事件,所以要做优化处理，避免一直加入上传队列
// 延迟1min加入队列，如果相同事件再次触发，顺延1min，直到没有相同事件触发，到达定时时间再添加任务
func addQueue(path string, b *Backup) {
	logrus.Infof("add queue, path: %s", path)
	t := time.AfterFunc(time.Minute, func() {
		defer delayTimers.Delete(path)
		logrus.Infof("timer trigger, execute add queue, path:%v", path)

		t, err := newBackupUploadTask(path, b)
		if err != nil {
			logrus.Error(path, errors.WithStack(err))
			return
		}
		options := []asynq.Option{
			asynq.TaskID(path),
			// asynq.ProcessAt(time.Now().Add(10 * time.Minute)),
			asynq.MaxRetry(1),
			asynq.Unique(time.Hour), //1h内保持唯一，如果执行了，会提前释放
		}
		if _, err = queue.GetClient().Enqueue(t, options...); err != nil {
			if errors.Is(err, asynq.ErrTaskIDConflict) {
				logrus.Infof("task id conflict,%v", path)
				return
			}
			logrus.Error(path, errors.WithStack(err))
			return
		}

	})

	timer, loaded := delayTimers.LoadOrStore(path, t)

	if loaded {
		t.Stop()
		timer.Reset(time.Minute)
		logrus.Infof("timer reset, path:%v", path)
	}
}

func initUpload(srcDir string) {
	bt, ok := bts.Load(srcDir)
	if !ok {
		logrus.Warningf("not found %s", srcDir)
		return
	}

	filepath.WalkDir(bt.Src, func(path string, d os.DirEntry, err error) error {
		if err != nil && err != filepath.SkipDir {
			logrus.Error(errors.WithStack(err))
			return err
		}
		if d.IsDir() {
			if isIgnored(bt, path) {
				return filepath.SkipDir
			}
			return nil
		}
		if isIgnored(bt, path) {
			return nil
		}
		// 加入上传队列
		addQueue(path, bt.Backup)
		return nil
	})
}

// 扫描文件夹，查询文件变动
func cronJob(bt *backupT) {
	fi, err := os.Stat(bt.Src)
	if err != nil {
		logrus.Error(bt, errors.WithStack(err))
		return
	}
	if !fi.IsDir() {
		logrus.Error(bt, errors.WithStack(err))
		return
	}
	lmt := getLastModifiedTime(bt.Backup.ID)
	logrus.Infof("cron job start, src: %s", bt.Src)

	filepath.WalkDir(bt.Src, func(path string, d os.DirEntry, err error) error {
		if err != nil && err != filepath.SkipDir {
			logrus.Error(path, errors.WithStack(err))
			return err
		}
		if d.IsDir() {
			if isIgnored(bt, path) {
				return filepath.SkipDir
			}
			return nil
		}

		if isIgnored(bt, path) || !isModified(lmt, path, d) {
			return nil
		}
		logrus.Infof("cron job add queue, file path: %v", path)
		addQueue(path, bt.Backup)

		return nil
	})

}

func isModified(lastBackupTime map[string]time.Time, path string, d os.DirEntry) bool {
	if t, ok := lastBackupTime[path]; ok {
		if info, err := d.Info(); err == nil {
			return info.ModTime().After(t)
		}
		return false
	}
	return true
}

func compileGitignore(ig string) *ignore.GitIgnore {
	s := strings.Split(ig, ";")
	gi := ignore.CompileIgnoreLines(s...)
	return gi
}

// match checks if a file or directory matches any of the compiled patterns.
func isIgnored(bt *backupT, path string) bool {
	path = filepath.Clean(strings.TrimPrefix(filepath.ToSlash(path), bt.Src+"/"))
	return bt.MatchesPath(path)
}
