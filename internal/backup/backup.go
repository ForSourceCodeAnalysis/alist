package backup

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/op"

	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/pkg/cron"
	"github.com/alist-org/alist/v3/pkg/generic_sync"
	"github.com/alist-org/alist/v3/pkg/utils"
	"github.com/fsnotify/fsnotify"
	"github.com/pkg/errors"
	ignore "github.com/sabhiram/go-gitignore"
	"github.com/sirupsen/logrus"
)

type backupT struct {
	model.Backup
	*ignore.GitIgnore
	*cron.Cron
}

var backupMemory generic_sync.MapOf[string, backupT]
var fsnotifyWatcher *fsnotify.Watcher

// 监听以文件夹为单位，不支持单个文件
func CreateBackup(c context.Context, b model.Backup) (uint64, error) {
	//判断源文件夹是否已经被监听
	if backupMemory.Has(b.Src) {
		return 0, fmt.Errorf("src %s has already been watched", b.Src)
	}
	//自动备份的是本地文件，与服务器强相关，所以必须配置server_id
	if len(conf.Conf.ServerId) <= 0 {
		return 0, errors.WithMessage(nil, "server id is not set")
	}
	b.ServerId = conf.Conf.ServerId

	//写入数据库
	if err := db.CreateBackup(&b); err != nil {
		return 0, errors.WithMessage(err, "failed create backup in database")
	}
	//加入监听
	addWatch(b)
	//事件监听才需要初始上传
	if b.InitUpload && b.Mode == model.MODE_EVENT {
		initUpload(b.Src)
	}

	return b.Id, nil
}

func UpdateBackup(c context.Context, b model.Backup) error {

	oldB, err := db.GetBackupById(uint(b.Id))
	if err != nil {
		return errors.WithMessage(err, "failed read old data, can not update ")
	}
	b.ServerId = conf.Conf.ServerId

	//写入数据库
	if err := db.UpdateBackup(&b); err != nil {
		return errors.WithMessage(err, "failed create backup in database")
	}
	if oldB.Disabled != b.Disabled { //启用状态有变化
		removeWatch(b.Src) //移除监听
		if !b.Disabled {
			addWatch(b)
		}
		return nil
	} else if !b.Disabled && (oldB.Mode != b.Mode || oldB.Ignore != b.Ignore) { //启用状态没变，且是启用，其它内容改变了,
		removeWatch(oldB.Src)
		addWatch(b)
	}
	//如果只是dst变化，不用调整监听，但是需要更新 backup
	if b.Dst != oldB.Dst {
		nb, ok := backupMemory.Load(b.Src)
		if ok {
			nb.Dst = b.Dst
			backupMemory.Store(b.Src, nb)
		}
	}

	return nil
}

func DeleteBackupById(id uint) error {
	m, err := db.GetBackupById(id)
	if err != nil {
		return errors.WithMessage(err, "failed get backup in database")
	}
	if err := db.DeleteBackupById(id); err != nil {
		return errors.WithMessage(err, "failed delete backup in database")
	}
	removeWatch(m.Src)
	backupMemory.Delete(m.Src)
	return nil
}

func ValidateBackup(b *model.Backup) error {
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
	if b.Mode == model.MODE_POLLing && b.PollingInterval == 0 {
		b.PollingInterval = 60
	}

	return nil
}

// 加入监听
func addWatch(b model.Backup) {
	// 检查文件（夹）是否存在
	fi, err := os.Stat(b.Src)
	if err != nil {
		logrus.Error(errors.WithStack(err))
		return
	}
	if !fi.IsDir() {
		logrus.Errorf("src %s is not a dir", b.Src)
		return
	}

	gi := compileGitignore(b.Ignore)
	bt := backupT{
		Backup:    b,
		GitIgnore: gi,
	}
	//不论是否启用都加入缓存中，新增时用来判断是否已存在
	backupMemory.Store(b.Src, bt)
	if b.Disabled {
		return
	}

	if b.Mode == model.MODE_POLLing { //轮询模式
		cron := cron.NewCron(b.PollingInterval * time.Minute)
		bt.Cron = cron
		backupMemory.Store(b.Src, bt)
		cron.Do(func() {
			pollBackup(bt)
		})
		return
	}
	watchOp(b.Src, "add")
}
func removeWatch(src string) {
	b, ok := backupMemory.Load(src)
	if !ok {
		return
	}
	//停止定时任务
	if b.Cron != nil {
		b.Cron.Stop()
		b.Cron = nil
	}
	watchOp(src, "remove")
}
func watchOp(src string, op string) {
	bt, _ := backupMemory.Load(src)

	// fsnotify不支持监听子文件夹，这里手动处理子文件夹
	filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil && err != filepath.SkipDir {
			logrus.Error(errors.WithStack(err))
			return err
		}

		if info.IsDir() {
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

// 处理事件
func eventDeal(event fsnotify.Event) {
	logrus.Infof("event trigger, file path: %v, op: %v", event.Name, event.Op.String())
	//只处理新增，修改
	if !event.Has(fsnotify.Create) && !event.Has(fsnotify.Write) {
		return
	}
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
	//上传文件
	for k, v := range backupMemory.ToMap() {
		if !utils.IsSubPath(k, event.Name) {
			continue
		}
		if isIgnored(v, event.Name) {
			return
		}
		backupUpload(event.Name, v.Backup)
		return
	}

}

func initUpload(srcDir string) {
	bt, ok := backupMemory.Load(srcDir)
	if !ok {
		logrus.Warningf("not found %s", srcDir)
		return
	}

	filepath.Walk(bt.Src, func(path string, info os.FileInfo, err error) error {
		if err != nil && err != filepath.SkipDir {
			logrus.Error(errors.WithStack(err))
			return err
		}
		if info.IsDir() {
			if isIgnored(bt, path) {
				return filepath.SkipDir
			}
			return nil
		}
		if isIgnored(bt, path) {
			return nil
		}
		backupUpload(path, bt.Backup)
		return nil
	})
}

func backupUpload(file string, b model.Backup) {
	if filepath.Base(file) == ".alistPollBackup" {
		return
	}
	dstDir := strings.Split(b.Dst, ";")
	//上传时需要保持原来的目录结构
	relativePath := strings.TrimPrefix(file, filepath.Dir(b.Src))
	relativeDir := filepath.Dir(relativePath)

	for _, v := range dstDir {
		tmpDir := filepath.Join(v, relativeDir)
		putAsTask(file, tmpDir)
	}

}

// 扫描文件夹，查询文件变动
// 首次备份后，会在每个目录下生成一个.alistPollBackup文件，用于记录当前文件夹下
// 每个文件上次备份时的修改时间
func pollBackup(bt backupT) {
	fi, err := os.Stat(bt.Src)
	if err != nil {
		logrus.Error(errors.WithStack(err))
		return
	}
	if !fi.IsDir() {
		logrus.Errorf("src %s is not a dir", bt.Src)
		return
	}
	var lastBackupTime = make(map[string]map[string]time.Time)
	modifiedTimes, err := readBackupTimeFile(filepath.Join(bt.Src, ".alistPollBackup"))
	if err != nil {
		logrus.Error(errors.WithStack(err))
		return
	}
	lastBackupTime[bt.Src] = modifiedTimes

	filepath.Walk(bt.Src, func(path string, info os.FileInfo, err error) error {
		if err != nil && err != filepath.SkipDir {
			logrus.Error(errors.WithStack(err))
			return err
		}
		if info.IsDir() {
			if isIgnored(bt, path) {
				return filepath.SkipDir
			}
			mt, err := readBackupTimeFile(filepath.Join(path, ".alistPollBackup"))
			if err != nil {
				logrus.Error(errors.WithStack(err))
			} else {
				lastBackupTime[path] = mt
			}

			return nil
		}

		if isIgnored(bt, path) || !isModified(lastBackupTime, path, info) {
			return nil
		}
		backupUpload(path, bt.Backup)
		return nil
	})
	//更新
	for k, v := range lastBackupTime {
		content, err := json.Marshal(v)
		if err != nil {
			logrus.Error(errors.WithStack(err))
			continue
		}
		if err := os.WriteFile(filepath.Join(k, ".alistPollBackup"), content, 0777); err != nil {
			logrus.Error(errors.WithStack(err))
		}
	}

}

func isModified(lastBackupTime map[string]map[string]time.Time, path string, info os.FileInfo) bool {
	if t, ok := lastBackupTime[filepath.Dir(path)][path]; ok {
		return info.ModTime().After(t)
	}
	return true
}
func readBackupTimeFile(path string) (map[string]time.Time, error) {
	modifiedTimes, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		logrus.Error(errors.WithStack(err))
		return nil, err
	}
	st := make(map[string]time.Time)
	if len(modifiedTimes) > 0 {
		if err := json.Unmarshal(modifiedTimes, &st); err != nil {
			logrus.Warn(errors.WithStack(err))
		}
	}
	return st, nil
}

// 从数据库初始化备份配置
func BackupInit() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		logrus.Errorf("create fs watcher failed: %v", err)
		return
	}
	fsnotifyWatcher = watcher

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

	//add watches
	bps, err := db.GetServerBackup()
	if err != nil {
		logrus.Error(errors.WithStack(err))
		return
	}
	for _, bp := range bps {
		addWatch(bp)
	}

}

func compileGitignore(ig string) *ignore.GitIgnore {
	s := strings.Split(ig, ";")
	gi := ignore.CompileIgnoreLines(s...)
	return gi
}

// match checks if a file or directory matches any of the compiled patterns.
func isIgnored(bt backupT, path string) bool {
	path = filepath.Clean(strings.TrimPrefix(filepath.ToSlash(path), bt.Src+"/"))
	return bt.MatchesPath(path)
}
