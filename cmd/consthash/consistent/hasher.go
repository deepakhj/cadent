// Copyright (C) 2015 Myfitnesspal

// some hashing functions that return an uint32 for our hasher
//

package consistent

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"hash/crc32"
	"hash/fnv"
	"strconv"
)

func prepString(key string) []byte {
	if len(key) < 64 {
		var scratch [64]byte
		copy(scratch[:], key)
		return scratch[:len(key)]
	}
	return []byte(key)

}

func Crc32(str string) uint32 {
	return crc32.ChecksumIEEE(prepString(str))
}

func Fnv32(str string) uint32 {
	hash := fnv.New32()
	hash.Write([]byte(str))
	out := hash.Sum32()
	return out
}

func Fnv32a(str string) uint32 {
	hash := fnv.New32a()
	hash.Write([]byte(str))
	out := hash.Sum32()
	return out
}

// this is hashing function used by graphite
// see https://github.com/graphite-project/carbon/blob/master/lib/carbon/hashing.py
// we cannot use a raw MD5 as it will need a bigint (beyond uint64 which is just silly for
// this application)
func SmallMD5(str string) uint32 {
	h := md5.New()
	h.Write([]byte(str))
	hexstr := hex.EncodeToString(h.Sum(nil))
	num, _ := strconv.ParseUint(hexstr[0:4], 16, 32)
	return uint32(num)
}

// we cannot use a raw SHA1 as it will need a bigint (beyond uint64 which is just silly for
// this application)
func SmallSHA1(str string) uint32 {
	h := sha1.New()
	h.Write([]byte(str))
	hexstr := hex.EncodeToString(h.Sum(nil))
	num, _ := strconv.ParseUint(hexstr[0:4], 16, 32)
	return uint32(num)
}
