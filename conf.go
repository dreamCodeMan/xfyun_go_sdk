package raasr

const (
	defaultPartSize   = 10 * 1024 * 1024
	defaultRetryTimes = 3
	defaultUA         = "raasr-go-sdk-v1.0.0"
	defaultDomain     = "https://raasr.xfyun.cn/api"
)

// Conf config struct
type Conf struct {
	AppID      string
	SecretKey  string
	PartSize   int64
	RetryTimes int
	Ch         string
	UA         string
	Domain     string
}

func getDefaultConf() *Conf {
	conf := Conf{}
	conf.PartSize = defaultPartSize
	conf.RetryTimes = defaultRetryTimes
	conf.UA = defaultUA
	conf.Domain = defaultDomain

	return &conf
}
