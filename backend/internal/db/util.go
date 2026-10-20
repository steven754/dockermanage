package db

import "time"

// nowUnix 返回当前 unix 秒
func nowUnix() int64 { return time.Now().Unix() }
