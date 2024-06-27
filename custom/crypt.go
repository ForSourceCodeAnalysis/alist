package custom

import (
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	rcCrypt "github.com/rclone/rclone/backend/crypt"
	"github.com/rclone/rclone/fs/config/configmap"
	log "github.com/sirupsen/logrus"
)

// 加密参数
type Crypt struct {
	Pwd                string //de/encrypt password 这里的密码是处理后的带___Obfuscated___前缀的，salt也是
	Salt               string
	FilenameEncryption string //reference drivers\crypt\meta.go Addtion
	DirnameEncryption  string
	FilenameEncode     string
	Suffix             string
}

func (c *Crypt) CryptFD(src string, dst string, op string) error {
	src, _ = filepath.Abs(src)
	log.Infof("src abs is %v", src)

	fileInfo, err := os.Stat(src)
	if err != nil {
		log.Errorf("reading file %v failed,err:%v", src, err)
		return err
	}
	//create cipher
	config := configmap.Simple{
		"password":                  c.Pwd,
		"password2":                 c.Salt,
		"filename_encryption":       c.FilenameEncryption,
		"directory_name_encryption": c.DirnameEncryption,
		"filename_encoding":         c.FilenameEncode,
		"suffix":                    c.Suffix,
		"pass_bad_blocks":           "",
	}
	cipher, err := rcCrypt.NewCipher(config)
	if err != nil {
		log.Errorf("create cipher failed,err:%v", err)
		return err
	}
	//check and create dst dir
	if dst != "" {
		dst, _ = filepath.Abs(dst)
		if err := checkCreateDir(dst); err != nil {
			return err
		}
	}

	if !fileInfo.IsDir() { //file
		if dst == "" {
			dst = filepath.Dir(src)
		}
		return c.cryptFile(cipher, src, dst, op)
	} else { //dir
		if dst == "" {
			//if src is dir and not set dst dir ,create ${src}_crypt dir as dst dir
			dst = path.Join(filepath.Dir(src), fileInfo.Name()+"_crypt")
		}
		log.Infof("dst : %v", dst)
		filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				log.Errorf("get file %v info failed, err:%v", p, err)
				return err
			}
			if info.IsDir() {
				//create output dir
				d := strings.Replace(p, src, dst, 1)
				log.Infof("create output dir %v", d)
				if err := checkCreateDir(d); err != nil {
					log.Errorf("create dir %v failed,err:%v", d, err)
					return err
				}
				return nil
			}
			d := strings.Replace(filepath.Dir(p), src, dst, 1)
			return c.cryptFile(cipher, p, d, op)
		})
	}
	return nil
}
func (c *Crypt) cryptFile(cipher *rcCrypt.Cipher, src string, dst string, op string) error {
	fileInfo, err := os.Stat(src)
	if err != nil {
		log.Errorf("get file %v  info failed,err:%v", src, err)
		return err
	}
	fd, err := os.Open(src)
	if err != nil {
		log.Errorf("open file %v failed,err:%v", src, err)
		return err
	}
	defer fd.Close()

	var cryptSrcReader io.Reader
	var outFile string
	if op == "encrypt" {
		cryptSrcReader, err = cipher.EncryptData(fd)
		if err != nil {
			log.Errorf("encrypt file %v failed,err:%v", src, err)
			return err
		}
		outFile = path.Join(dst, fileInfo.Name()+c.Suffix)
	} else {
		cryptSrcReader, err = cipher.DecryptData(fd)
		if err != nil {
			log.Errorf("decrypt file %v failed,err:%v", src, err)
			return err
		}
		outFile = path.Join(dst, strings.Replace(fileInfo.Name(), c.Suffix, "", -1))
	}
	//write new file
	wr, err := os.OpenFile(outFile, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		log.Errorf("create file %v failed,err:%v", outFile, err)
		return err
	}
	defer wr.Close()

	_, err = io.Copy(wr, cryptSrcReader)
	if err != nil {
		log.Errorf("write file %v failed,err:%v", outFile, err)
		return err
	}
	return nil

}
