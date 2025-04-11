package bootstrap

import (
	"context"

	"github.com/alist-org/alist/v3/internal/conf"
	"github.com/alist-org/alist/v3/internal/db"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/pkg/utils"
)

func LoadStorages() {
	storages, err := db.GetEnabledStorages()
	if err != nil {
		utils.Log.Fatalf("failed get enabled storages: %+v", err)
	}
	// storages是alist服务的基础，如果使用协程加载，可能会出现其它依赖服务已经启动，但是storages还没加载好的问题，所以这里不应该用协程，直接阻塞加载就可以
	// go func(storages []model.Storage) {
	for i := range storages {
		err := op.LoadStorage(context.Background(), storages[i])
		if err != nil {
			utils.Log.Errorf("failed get enabled storages: %+v", err)
		} else {
			utils.Log.Infof("success load storage: [%s], driver: [%s], order: [%d]",
				storages[i].MountPath, storages[i].Driver, storages[i].Order)
		}
	}
	conf.StoragesLoaded = true
	// }(storages)
}
