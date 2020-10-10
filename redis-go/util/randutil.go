package util

import (
	"bytes"
	"crypto/sha1"
	"encoding/binary"
	"math/rand"
	"os"
	"sync/atomic"
	"time"
)

var seedInitialized = false
var seed = make([]byte, 20)
var counter uint64 = 0

// GetRandomHexChars 获取随机16机制字符串
func GetRandomHexChars(length int) []byte {
	var charset = "0123456789abcdef"
	rs := getRandomBytes(length)
	for j := 0; j < length; j++ {
		rs[j] = charset[rs[j]&0x0F]
	}
	return rs
}

func getRandomBytes(length int) []byte {
	if !seedInitialized {
		file, err := os.Open("/dev/urandom")
		if err == nil {
			defer file.Close()
			if _, ferr := file.Read(seed); ferr != nil {
				seedInitialized = true
			}
		}
		if !seedInitialized {
			for j := 0; j < len(seed); j++ {
				seed[j] = byte((time.Now().UnixNano() ^ int64(os.Getpid()) ^ int64(rand.Int())) & 0xFF)
			}
			seedInitialized = true
		}
	}

	rs := make([]byte, length)
	remaining := length
	copyPos := 0
	for remaining > 0 {
		atomic.AddUint64(&counter, 1)
		cv := atomic.LoadUint64(&counter)
		buf := bytes.NewBuffer([]byte{})
		binary.Write(buf, binary.BigEndian, cv)

		sh := sha1.New()
		sh.Write(append(seed, buf.Bytes()...))
		bs := sh.Sum(nil)
		copyLen := remaining
		if remaining > len(bs) {
			copyLen = len(bs)
		}
		copy(rs[copyPos:], bs[:copyLen])
		copyPos += copyLen
		remaining -= copyLen
	}
	return rs
}
