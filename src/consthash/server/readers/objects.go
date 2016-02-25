/*
   Base Json objects to simulate a graphite API response
*/

package readers

import (
	"fmt"
)

/****************** Output structs *********************/

// the basic metric json blob for find
type MetricFindItem struct {
	Text          string `json:"text"`
	Expandable    int    `json:"expandable"`
	Leaf          int    `json:"leaf"`
	Id            string `json:"id"`
	Path          string `json:"path"`
	AllowChildren int    `json:"allowChildren"`
}

type MetricFindItems []MetricFindItem

// the basic metric json blob for find
type MetricExpandItem struct {
	Results []string `json:"results"`
}

type DataPoint struct {
	Time  int
	Value *float64 // need nils for proper "json none"
}

func NewDataPoint(time int, val float64) DataPoint {
	d := DataPoint{Time: time, Value: new(float64)}
	d.SetValue(val)
	return d
}

func (d DataPoint) MarshalJSON() ([]byte, error) {
	if d.Value == nil {
		return []byte(fmt.Sprintf("[null, %d]", d.Time)), nil
	}
	return []byte(fmt.Sprintf("[%f, %d]", *d.Value, d.Time)), nil
}
func (d DataPoint) SetValue(val float64) {
	v := d.Value
	*v = val
}

// the basic metric json blob for find
type RenderItem struct {
	Target     string      `json:"target"`
	Datapoints []DataPoint `json:"datapoints"`
}

type RenderItems []RenderItem

// the basic whisper metric json blob for find

type WhisperRenderItem struct {
	Start  int                    `json:"from"`
	End    int                    `json:"to"`
	Step   int                    `json:"step"`
	Series map[string][]DataPoint `json:"series"`
}
