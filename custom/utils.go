package custom

import (
	"os"

	log "github.com/sirupsen/logrus"
)

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
