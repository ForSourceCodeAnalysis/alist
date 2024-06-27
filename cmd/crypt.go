package cmd

import (
	log "github.com/sirupsen/logrus"

	"github.com/alist-org/alist/v3/custom"
	"github.com/spf13/cobra"
)

///加解密命令
///crypt驱动加解密的命令形式

type cryptCmdParamS struct {
	op  string //decrypt or encrypt
	src string //source dir or file
	dst string //out destination
}

var cryptParam custom.Crypt
var cryptCmdParam cryptCmdParamS

// VersionCmd represents the version command
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
