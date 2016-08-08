// need:  go get -u github.com/ugorji/go/codec/codecgen

//go:generate codecgen -o codec.render.go codec.go

package codec

// sort-hand keys for space purposes
type FullStat struct {
	Time  int64   `json:"t"  codec:"t" msg:"t"`
	Min   float64 `json:"n"  codec:"n" msg:"n"`
	Max   float64 `json:"m"  codec:"m" msg:"m"`
	Sum   float64 `json:"s"  codec:"s" msg:"s"`
	First float64 `json:"f"  codec:"f" msg:"f"`
	Last  float64 `json:"l"  codec:"l" msg:"l"`
	Count int64   `json:"c"  codec:"c" msg:"c"`
}

type StatSmall struct {
	Time int64   `json:"t" codec:"t" msg:"t"`
	Val  float64 `json:"v" codec:"v" msg:"v"`
}

type Stat struct {
	StatType  bool       `json:"t" codec:"t" msg:"t"`
	Stat      *FullStat  `json:"s" codec:"s" msg:"s"`
	SmallStat *StatSmall `json:"m" codec:"m" msg:"m"`
}

type Stats struct {
	FullTimeResolution bool    `json:"r" codec:"r" msg:"r"`
	Stats              []*Stat `json:"s" codec:"s" msg:"s"`
}
