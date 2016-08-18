/*
Copyright 2016 Under Armour, Inc.
 
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
 
http://www.apache.org/licenses/LICENSE-2.0
 
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Copyright (C) 2015 Myfitnesspal

// some hashing functions that return an int32 for our hasher
//
package main

import (
	"crypto/md5"
	"crypto/sha1"
	"encoding/hex"
	"hash/crc32"
	"hash/fnv"
	"log"
	"strconv"
)

func Crc32(str string) uint32 {
	return crc32.ChecksumIEEE([]byte(str))
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
func SmallMd5(str string) uint32 {
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

func main() {

	str := "MOOOOGOOOO234"
	log.Println("SmallMD5", SmallMd5(str))
	log.Println("SmallSHA1", SmallSHA1(str))
	log.Println("CRC32", Crc32(str))
	log.Println("Fnv32", Fnv32(str))
	log.Println("Fnv32a", Fnv32a(str))

}
