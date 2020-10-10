package main

import (
	"math/rand"
	"os"
	"redis-server/dict"
	"redis-server/server"
	"redis-server/util"
	"time"
)

func main() {
	rand.Seed(time.Now().UnixNano() ^ int64(os.Getpid()))
	hashseed := util.GetRandomHexChars(16)
	dict.InitDictHashFunctionSeed(hashseed)
	initServerConfig()
}

func initServerConfig() {
	server.UpdateCachedTime()
	server.SetRunid(util.GetRandomHexChars(41))
}
