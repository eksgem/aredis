package dict

var dictHashFunctionSeed []byte

// InitDictHashFunctionSeed 初始化字典函数种子
func InitDictHashFunctionSeed(seed []byte) {
	dictHashFunctionSeed = seed
}
