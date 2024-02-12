package flags

// 根命令命令行参数，适用于子命令，亦可参考 root.go
var (
	DataDir     string //指定数据目录
	Debug       bool   //是否开启debug模式，主要影响日志
	NoPrefix    bool   //环境变量是否有前缀
	Dev         bool   //是否开启dev模式，影响面比较大，具体可以全局搜索flag.Dev查看
	ForceBinDir bool   //强制使用可执行程序所在目录作为数据目录
	LogStd      bool   //强制日志输出到标准输出
)
