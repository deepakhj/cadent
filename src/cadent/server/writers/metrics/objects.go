/*
   Base Json objects to simulate a graphite API response
*/

package metrics

import (
	"fmt"
	"math"
	"sync"
	"time"
)

/****************** Output structs *********************/

type DataPoint struct {
	Time  int
	Value *float64 // need nils for proper "json none"
}

func NewDataPoint(time int, val float64) DataPoint {
	d := DataPoint{Time: time, Value: new(float64)}
	d.SetValue(&val)
	return d
}

func (d DataPoint) MarshalJSON() ([]byte, error) {
	if d.Value == nil || math.IsNaN(*d.Value) {
		return []byte(fmt.Sprintf("[null, %d]", d.Time)), nil
	}

	return []byte(fmt.Sprintf("[%f, %d]", *d.Value, d.Time)), nil
}
func (d DataPoint) SetValue(val *float64) {
	d.Value = val
}

// the basic metric json blob for find
type RenderItem struct {
	Target     string      `json:"target"`
	Datapoints []DataPoint `json:"datapoints"`
}

type RenderItems []RenderItem

// the basic whisper metric json blob for find

type WhisperRenderItem struct {
	RealStart int                    `json:"data_from"`
	RealEnd   int                    `json:"data_end"`
	Start     int                    `json:"from"`
	End       int                    `json:"to"`
	Step      int                    `json:"step"`
	Series    map[string][]DataPoint `json:"series"`
}

/**  LRU cacher bits **/
type WhisperRenderCacher struct {
	sync.RWMutex
	Data    []DataPoint
	expires *time.Time
}

// add some functions lru interface for caching these guys
func (wh WhisperRenderCacher) Size() int {
	return len(wh.Data) * 8 * 64 // int * float
}

func (wh WhisperRenderCacher) ToString() string {
	return "Whisper Render Cache"
}

func (wh WhisperRenderCacher) Touch(dur time.Duration) {
	wh.Lock()
	defer wh.Unlock()
	expiration := time.Now().Add(dur)
	wh.expires = &expiration
}

func (wh WhisperRenderCacher) IsExpired() bool {
	wh.RLock()
	defer wh.RUnlock()
	if wh.expires == nil {
		return true
	}
	return wh.expires.Before(time.Now())
}

/*** Raw renderers **/

type RawDataPoint struct {
	Time  int     `json:"time"`
	Sum   float64 `json:"sum"`
	Mean  float64 `json:"mean"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
	Count int64   `json:"count"`
}

type RawRenderItem struct {
	Metric    string         `json:"metric"`
	RealStart int            `json:"data_from"`
	RealEnd   int            `json:"data_end"`
	Start     int            `json:"from"`
	End       int            `json:"to"`
	Step      int            `json:"step"`
	Data      []RawDataPoint `json:"data"`
}
