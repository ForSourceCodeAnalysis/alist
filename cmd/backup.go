package cmd

import (
	"context"
	"os"
	stdPath "path"
	"path/filepath"
	"strings"
	"time"

	"github.com/alist-org/alist/v3/internal/bootstrap"
	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/fs"
	interModel "github.com/alist-org/alist/v3/internal/model"
	"github.com/alist-org/alist/v3/internal/stream"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var src, dst, dirname string
var exclude []string

// VersionCmd represents the version command
var BackupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Show current version of AList",
	Run: func(cmd *cobra.Command, args []string) {
		Init()
		bootstrap.LoadStorages()
		//等待驱动加载完成
		for {
			time.Sleep(time.Second * 5)
			if conf.StoragesLoaded {
				break
			}

		}
		defer Release()

		backup()

	},
}

func init() {
	RootCmd.AddCommand(BackupCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	BackupCmd.Flags().StringVar(&src, "src", "", "source dir or file to backup")
	BackupCmd.Flags().StringVar(&dst, "dst", "/crypt", "backup dst dir")
	BackupCmd.Flags().StringVar(&dirname, "dirname", "", "dirname for files to put in on dst dir")
	BackupCmd.Flags().StringSliceVar(&exclude, "exclude", []string{}, "files or dir not backup")
}

func backup() {
	backupConf := conf.Conf.Backup

	if len(src) > 0 { //命令行参数优先级更高
		backupConf = []conf.BackupConfig{
			{
				Src:     src,
				Dst:     dst,
				Dirname: dirname,
				Exclue:  exclude,
			},
		}
	}

	for _, bc := range backupConf {
		fi, err := os.Stat(bc.Src)
		if err != nil {
			logrus.Errorf("读取文件%v信息失败,err:%v", bc.Src, err)
			continue
		}
		bc.Src = strings.TrimSuffix(bc.Src, "/") + "/" //确保文件夹结尾有/
		dst := strings.TrimSuffix(bc.Src, "/") + "/"
		if len(bc.Dirname) > 0 {
			dst += bc.Dirname + "/"
		}

		if !fi.IsDir() { //文件直接上传
			uploadFile(bc.Src, dst, fi)
			continue
		}

		//文件夹
		filepath.Walk(bc.Src, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				logrus.Errorf("读取文件%v信息失败,err:%v", path, err)
				return err
			}
			if info.IsDir() {
				return nil
			}
			//判断是否在过滤目录中
			p, _ := strings.CutPrefix(path, bc.Src)

			for _, ex := range bc.Exclue {
				if ex == path || strings.HasPrefix(p, ex) {
					return nil
				}
			}
			if strings.Contains(p, "/") {
				dst += stdPath.Dir(p)
			}
			//上传
			uploadFile(path, dst, info)
			return nil
		})
	}

}

func uploadFile(filePath string, dst string, fileInfo os.FileInfo) {
	fd, err := os.Open(filePath)
	if err != nil {
		logrus.Fatal(err)
		return
	}
	streamer := &stream.FileStream{
		Obj: &interModel.Object{
			Name:     fileInfo.Name(),
			Size:     fileInfo.Size(),
			Modified: fileInfo.ModTime(), //这里修复了3.26版本直接取当前时间的bug
		},
		Reader:       fd,
		Mimetype:     "text/plain",
		WebPutAsTask: false,
	}
	err = fs.PutDirectly(context.TODO(), dst, streamer, true)
	if err != nil {
		logrus.Fatal(err)
	}
}
