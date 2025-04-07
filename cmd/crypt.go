package cmd

import (
	log "github.com/sirupsen/logrus"

	"io"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	rcCrypt "github.com/rclone/rclone/backend/crypt"
	"github.com/rclone/rclone/fs/config/configmap"
)

///加解密命令
///crypt驱动加解密的命令形式

type cryptCmdParamS struct {
	op  string //decrypt or encrypt
	src string //source dir or file
	dst string //out destination
}

var cryptParam Crypt
var cryptCmdParam cryptCmdParamS

// CryptCmd represents the crypt command
var CryptCmd = &cobra.Command{
	Use:   "crypt",
	Short: "Encrypt or decrypt local file or dir to local",
	Run: func(cmd *cobra.Command, args []string) {
		log.Info(args)
		Init()
		log.Info("init over")
		log.Info(cryptCmdParam)
		cryptParam.CryptFD(cryptCmdParam.src, cryptCmdParam.dst, cryptCmdParam.op)
	},
}

func init() {
	RootCmd.AddCommand(CryptCmd)
	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// versionCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	CryptCmd.Flags().StringVarP(&cryptCmdParam.src, "src", "s", "", "")
	CryptCmd.Flags().StringVarP(&cryptCmdParam.dst, "dst", "d", "", "")
	CryptCmd.Flags().StringVar(&cryptCmdParam.op, "op", "", "")

	CryptCmd.Flags().StringVar(&cryptParam.Pwd, "pwd", "", "")
	CryptCmd.Flags().StringVar(&cryptParam.Salt, "salt", "", "")
	CryptCmd.Flags().StringVar(&cryptParam.FilenameEncryption, "filename_encrypt", "off", "")
	CryptCmd.Flags().StringVar(&cryptParam.DirnameEncryption, "dirname_encrypt", "false", "")
	CryptCmd.Flags().StringVar(&cryptParam.FilenameEncode, "filename_encode", "base64", "")
	CryptCmd.Flags().StringVar(&cryptParam.Suffix, "suffix", ".bin", "")
}

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

// check dir exist ,if not ,create
func checkCreateDir(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			log.Printf("create dir %v failed,err:%v", dir, err)
			return err
		}
	}
	return nil
}
