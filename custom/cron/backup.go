package cron

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/fs"
	"github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/stream"
	"github.com/sirupsen/logrus"
)

type backupStatusS struct {
	sync.RWMutex
	running bool
}

var backupStatus backupStatusS

func Run() {
	//自动备份
	if isBackupRunning(&backupStatus) {
		logrus.Infof("backup is running, wait next......")
		return
	}
	backup()
}

func isBackupRunning(b *backupStatusS) bool {
	b.RLock()
	defer b.RUnlock()
	return b.running
}

func setBackupStatus(b *backupStatusS, status bool) {
	b.Lock()
	defer b.Unlock()
	b.running = status

}
func backup() {
	setBackupStatus(&backupStatus, true)
	//从配置文件中获取要备份的文件
	backupConf := conf.Conf.Backup
	logrus.Infof("backup config is %v", backupConf)

	for _, bc := range backupConf {
		bc.Src = filepath.ToSlash(bc.Src)
		fi, err := os.Stat(bc.Src)
		if err != nil {
			logrus.Errorf("读取文件%v信息失败,err:%v", bc.Src, err)
			continue
		}
		//如果没有配置目的文件目录名称，就使用和源目录同名的文件夹
		if len(bc.Dirname) == 0 {
			bc.Dirname = path.Base(bc.Src)
			logrus.Infof("dst dirname is:%v", bc.Dirname)
		}
		//
		cdst := make([]string, len(bc.Dst))
		copy(cdst, bc.Dst)
		for i, bd := range cdst {
			logrus.Infof("will deal dst is:%v", bd)
			cdst[i] = path.Join(bd, bc.Dirname)
		}
		logrus.Infof("dealed dst is:%v", bc.Dst)
		if !fi.IsDir() { //文件直接上传
			uploadFile(bc.Src, bc.Dst, fi)
			continue
		}
		//文件夹
		filepath.Walk(bc.Src, func(p string, info os.FileInfo, err error) error {
			if err != nil && err != filepath.SkipDir {
				logrus.Errorf("读取文件%v信息失败,err:%v", p, err)
				return err
			}
			p = filepath.ToSlash(p)
			logrus.Infof("current path:%v", p)
			//判断是否在过滤目录中
			rp, _ := strings.CutPrefix(p, path.Clean(bc.Src)+"/") //相对目录
			logrus.Infof("relative path:%v", rp)
			for _, ex := range bc.Ignore {
				ex = filepath.ToSlash(ex)
				if ex == rp {
					return filepath.SkipDir
				}
			}

			if info.IsDir() {
				//如果文件夹没有变动，跳过扫描里面的文件
				if _, m := db.IsFileModified(p, info.ModTime()); !m {
					logrus.Infof("path:%v not modified,skip", info.Name())
					return filepath.SkipDir
				} else { //更新修改时间
					m := &model.Backup{
						FilePath:     p,
						LastModified: info.ModTime(),
					}
					db.UpdateBackupFile(m)
				}
				return nil
			}

			//处理目的目录
			tmpDst := make([]string, len(cdst))
			copy(tmpDst, cdst)
			for i, d := range tmpDst {
				tmpDst[i] = path.Join(d, path.Dir(rp))
			}
			logrus.Infof("second dealed dst is:%v", tmpDst)

			//上传
			uploadFile(p, tmpDst, info)
			return nil
		})
	}
	setBackupStatus(&backupStatus, false)

}

func uploadFile(filePath string, dst []string, fileInfo os.FileInfo) {
	m, f := db.IsFileModified(filePath, fileInfo.ModTime())
	if !f { //没有变动
		return
	}

	//依次上传到指定的备份目录
	for _, d := range dst {
		//每次上传前要重新打开，否则第后面会读取失败
		fd, err := os.Open(filePath)
		if err != nil {
			logrus.Error(err)
			continue
		}
		streamer := &stream.FileStream{
			Obj: &model.Object{
				Name:     fileInfo.Name(),
				Size:     fileInfo.Size(),
				Modified: fileInfo.ModTime(), //这里修复了3.26版本直接取当前时间的bug
			},
			Reader:       fd,
			Mimetype:     "text/plain",
			WebPutAsTask: false,
		}

		logrus.Infof("puting into dst dir is:%v", d)
		err = fs.PutDirectly(context.TODO(), d, streamer, true)
		fd.Close()
		time.Sleep(5 * time.Millisecond)
		if err != nil {
			logrus.Error(err)
			continue
		}
	}
	if m == nil {
		m = &model.Backup{
			FilePath:     filePath,
			LastModified: fileInfo.ModTime(),
		}
	}
	db.UpdateBackupFile(m)
}
