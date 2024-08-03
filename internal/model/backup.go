package model

import "time"

const (
	MODE_EVENT   = 1
	MODE_POLLing = 2
)

// 监听文件变化有两种模式，事件和轮询
// 事件：利用fsnotify包监听文件变化，文件一有变化就会有事件触发，效率较高，适用于服务长期运行的场景
// 轮询：利用定时任务查询文件变化，效率比较低，适用于服务非长期运行的场景，比如家用电脑这种每天都会关机的场景
type Backup struct {
	Id              uint64        `json:"id" gorm:"primaryKey;autoIncrement"`
	ServerId        string        `json:"server_id"`
	Src             string        `json:"src"`    //source file or dir or offline-download url
	Dst             string        `json:"dst"`    //dst dir, one or more, split by ";"
	Ignore          string        `json:"ignore"` //source file or dir that  do not backup， support multi, split by ";"
	Disabled        bool          `json:"disabled"`
	Mode            uint          `json:"mode"`             //模式，event or poll
	PollingInterval time.Duration `json:"polling_interval"` //
	InitUpload      bool          `json:"init_upload"`      //初始上传，第一次配置是否先将所有文件上传一次
	UpdatedAt       time.Time     `json:"updated_at" gorm:"autoUpdateTime"`
	CreatedAt       time.Time     `json:"created_at" gorm:"autoUpdateTime"`
}
