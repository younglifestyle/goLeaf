package model

type SnowFlakeEtcdHolder struct {
	EtcdAddressNode string //保存自身的key  ip:port-1
	ListenAddress   string //保存自身的key ip:port
	Ip              string
	Port            string
	LastUpdateTime  int64
	WorkerId        int
}

type Endpoint struct {
	IP        string `json:"ip"`
	Port      string `json:"port"`
	Timestamp int64  `json:"timestamp"`
}
