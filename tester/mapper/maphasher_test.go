package mapper

import (
	"testing"
	"math/rand"
	"hash/crc32"
)

var mp_str map[string]bool
var r_list_str []string


var mp_int map[int]bool
var r_list_int []int

var mp_crc map[uint32]bool

func init() {

	mp_str = make(map[string]bool)
	mp_int = make(map[int]bool)
	mp_crc = make(map[uint32]bool)

	r_list_str = make([]string, 100000)
	for idx, _ := range r_list_str {
		r_list_str[idx] = RandStr()
	}

	r_list_int = make([]int, 100000)
	for idx, _ := range r_list_int {
		r_list_int[idx] = rand.Intn(123123123123)
	}


}

func BenchmarkMapStringPut(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, st := range r_list_str {
			mp_str[st] = true
		}
	}
}

func BenchmarkMapStringGet(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, st := range r_list_str {
			_ = mp_str[st]
		}
	}
}

func BenchmarkMapIntPut(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, st := range r_list_int {
			mp_int[st] = true
		}
	}
}

func BenchmarkMapIntGet(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, st := range r_list_int {
			_ = mp_int[st]
		}
	}
}


func BenchmarkMapCRCPut(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		for _, st := range r_list_str {
			k := crc32.ChecksumIEEE([]byte(st))
			mp_crc[k] = true
		}
	}
}

func BenchmarkMapCRCGet(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, st := range r_list_str {
			k := crc32.ChecksumIEEE([]byte(st))
			_ = mp_crc[k]
		}
	}
}