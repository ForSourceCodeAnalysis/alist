package backup

import "time"

const (
	modeEvent = 1
	modeCron  = 2
)

// Backup 监听文件变化有两种模式，事件和定时
// 事件：利用fsnotify包监听文件变化，文件一有变化就会有事件触发，效率较高，适用于服务长期运行的场景
// 定时：利用定时任务查询文件变化，效率比较低，适用于服务非长期运行的场景，比如家用电脑这种每天都会关机的场景
type Backup struct {
	ID         uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	ServerID   string    `json:"server_id"`
	Src        string    `json:"src"`    //source file or dir or offline-download url
	Dst        string    `json:"dst"`    //dst dir, one or more, split by ";"
	Ignore     string    `json:"ignore"` //source file or dir that  do not backup， support multi, split by ";"
	Disabled   bool      `json:"disabled"`
	Mode       uint      `json:"mode"`                 //模式，event or cron
	Cron       string    `json:"cron"`                 //
	InitUpload bool      `json:"init_upload" gorm:"-"` //初始上传，第一次配置是否先将所有文件上传一次
	UpdatedAt  time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	CreatedAt  time.Time `json:"created_at" gorm:"autoUpdateTime"`
}

// File 备份文件
type File struct {
	ID               uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	BackupID         uint64    `json:"backup_id"`
	Name             string    `json:"name"`
	Dir              string    `json:"dir"`
	TimeConsuming    uint64    `json:"time_consuming"`
	LastModifiedTime time.Time `json:"last_modified_time"`
	UpdatedAt        time.Time `json:"updated_at" gorm:"autoUpdateTime"`
	CreatedAt        time.Time `json:"created_at" gorm:"autoUpdateTime"`
}

// TableName define table name
func (File) TableName() string {
	return "x_backup_files"
}
