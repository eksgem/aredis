package server

import (
	"time"
)

var unixtime int64
var mstime int64
var runid []byte

// UpdateCachedTime 更新缓存的时间
func UpdateCachedTime() {
	ns := time.Now().UnixNano()
	mstime = ns / 1000000
	unixtime = mstime / 1000

}

// GetUnixtime 获取系统缓存的unix时间戳
func GetUnixtime() int64 {
	return unixtime
}

// GetMstime 获取系统缓存的以毫秒为单位的时间戳
func GetMstime() int64 {
	return mstime
}

//SetRunid 设置服务运行id
func SetRunid(id []byte) {
	runid = id
}
