// need:  go get -u github.com/ugorji/go/codec/codecgen
// render via: codecgen -o codec.render.go codec.codec.go

package codec

// sort-hand keys for space purposes
type CodecFullStat struct {
	Time  int64   `json:"t"  codec:"t"`
	Min   float64 `json:"n"  codec:"n"`
	Max   float64 `json:"m"  codec:"m"`
	Sum   float64 `json:"s"  codec:"s"`
	First float64 `json:"f"  codec:"f"`
	Last  float64 `json:"l"  codec:"l"`
	Count int64   `json:"c"  codec:"c"`
}

type CodecStatSmall struct {
	Time int64   `json:"t" codec:"t"`
	Val  float64 `json:"v" codec:"v"`
}

type CodecStat struct {
	StatType  bool            `json:"t" codec:"t"`
	Stat      *CodecFullStat  `json:"s" codec:"s"`
	SmallStat *CodecStatSmall `json:"m" codec:"m"`
}

type CodecStats struct {
	FullTimeResolution bool         `json:"r" codec:"r"`
	Stats              []*CodecStat `json:"s" codec:"s"`
}
