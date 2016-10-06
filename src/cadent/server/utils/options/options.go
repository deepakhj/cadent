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

/*
 little helper to get "options" from a map[string]interface{}
*/

package options

import (
	"fmt"
	"time"
)

type Options map[string]interface{}

func New() Options {
	return Options(make(map[string]interface{}))
}

func (o *Options) get(name string) (interface{}, bool) {
	c := map[string]interface{}(*o)
	gots, ok := c[name]
	return gots, ok
}

func (o *Options) Set(name string, val interface{}) {
	c := map[string]interface{}(*o)
	c[name] = val
}

func (o *Options) String(name, def string) string {
	got, ok := o.get(name)
	if ok {
		return got.(string)
	}
	return def
}

func (o *Options) StringRequired(name string) (string, error) {
	got, ok := o.get(name)
	if ok {
		return got.(string), nil
	}
	return "", fmt.Errorf("%s is required", name)
}

func (o *Options) Object(name, def string) interface{} {
	got, ok := o.get(name)
	if ok {
		return got
	}
	return def
}

func (o *Options) ObjectRequired(name string) (interface{}, error) {
	got, ok := o.get(name)
	if ok {
		return got, nil
	}
	return nil, fmt.Errorf("%s is required", name)
}

func (o *Options) Int64(name string, def int64) int64 {
	got, ok := o.get(name)
	if ok {
		switch got.(type) {
		case int64:
			return got.(int64)
		case uint8:
			return int64(got.(uint8))
		case uint16:
			return int64(got.(uint16))
		case uint32:
			return int64(got.(uint32))
		case uint64:
			return int64(got.(uint64))
		case int8:
			return int64(got.(int8))
		case int16:
			return int64(got.(int16))
		case int32:
			return int64(got.(int32))
		case int:
			return int64(got.(int))
		case float64:
			return int64(got.(float64))
		case float32:
			return int64(got.(float32))
		default:
			panic(fmt.Errorf("Cannot convert int type"))
		}
	}
	return def
}

func (o *Options) Int64Required(name string) (int64, error) {
	got, ok := o.get(name)
	if ok {
		switch got.(type) {
		case int64:
			return got.(int64), nil
		case uint8:
			return int64(got.(uint8)), nil
		case uint16:
			return int64(got.(uint16)), nil
		case uint32:
			return int64(got.(uint32)), nil
		case uint64:
			return int64(got.(uint64)), nil
		case int8:
			return int64(got.(int8)), nil
		case int16:
			return int64(got.(int16)), nil
		case int32:
			return int64(got.(int32)), nil
		case int:
			return int64(got.(int)), nil
		case float64:
			return int64(got.(float64)), nil
		case float32:
			return int64(got.(float32)), nil
		default:
			return 0, fmt.Errorf("%s is not an int (it's a `%T`)", name, got)
		}
	}
	return 0, fmt.Errorf("%s is required", name)
}

func (o *Options) Float64(name string, def float64) float64 {
	got, ok := o.get(name)
	if ok {
		switch got.(type) {
		case int64:
			return float64(got.(int64))
		case uint8:
			return float64(got.(uint8))
		case uint16:
			return float64(got.(uint16))
		case uint32:
			return float64(got.(uint32))
		case uint64:
			return float64(got.(uint64))
		case int8:
			return float64(got.(int8))
		case int16:
			return float64(got.(int16))
		case int32:
			return float64(got.(int32))
		case int:
			return float64(got.(int))
		case float32:
			return float64(got.(float32))
		case float64:
			return got.(float64)
		default:
			panic(fmt.Errorf("%s is not a float", name))
		}
	}
	return def
}

func (o *Options) Float64Required(name string) (float64, error) {
	got, ok := o.get(name)
	if ok {
		switch got.(type) {
		case int64:
			return float64(got.(int64)), nil
		case uint8:
			return float64(got.(uint8)), nil
		case uint16:
			return float64(got.(uint16)), nil
		case uint32:
			return float64(got.(uint32)), nil
		case uint64:
			return float64(got.(uint64)), nil
		case int8:
			return float64(got.(int8)), nil
		case int16:
			return float64(got.(int16)), nil
		case int32:
			return float64(got.(int32)), nil
		case int:
			return float64(got.(int)), nil
		case float32:
			return float64(got.(float32)), nil
		case float64:
			return got.(float64), nil
		default:
			return 0, fmt.Errorf("%s is not a float", name)

		}
	}
	return 0, fmt.Errorf("%s is required", name)
}

func (o *Options) Bool(name string, def bool) bool {
	got, ok := o.get(name)
	if ok {
		return got.(bool)
	}
	return def
}

func (o *Options) BoolRequired(name string) (bool, error) {
	got, ok := o.get(name)
	if ok {
		return got.(bool), nil
	}
	return false, fmt.Errorf("%s is required", name)
}

func (o *Options) Duration(name string, def time.Duration) time.Duration {
	got, ok := o.get(name)
	if ok {
		rdur, err := time.ParseDuration(got.(string))
		if err != nil {
			panic(err)
		}
		return rdur
	}
	return def
}

func (o *Options) DurationRequired(name string) (time.Duration, error) {
	got, ok := o.get(name)
	if ok {
		rdur, err := time.ParseDuration(got.(string))
		if err != nil {
			return time.Duration(0), err
		}
		return rdur, nil
	}
	return time.Duration(0), fmt.Errorf("%s is required", name)
}

func (o *Options) ToString() string {
	out := "Options("
	for k, v := range *o {
		out += fmt.Sprintf("%s=%v, ", k, v)
	}
	out += ")"
	return out
}
