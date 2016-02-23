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
	AllowChildren int    `json:"allowChildren"`
}

type MetricFindItems []MetricFindItem

// the basic metric json blob for find
type MetricExpandItem struct {
	Results []string `json:"results"`
}

type DataPoint struct {
	Time  int
	Value float64
}

func (s DataPoint) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("[%f, %d]", s.Value, s.Time)), nil
}

// the basic metric json blob for find
type RenderItem struct {
	Target     string      `json:"target"`
	Datapoints []DataPoint `json:"datapoints"`
}

type RenderItems []RenderItem
