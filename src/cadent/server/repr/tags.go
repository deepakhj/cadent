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
   A Helper to basically list the Tags that are concidered "Id worthy"
   and those that are not
*/

package repr

import (
	"io"
	"strings"
)

type SortingTags [][]string

// make a tag array from a string input inder a few conditions
// tag=val.tag=val.tag=val
// or
// tag_is_val.tag_is_val
// or
// tag=val,tag=val
// or
// tag_is_val,tag_is_val
// or
// tag=val tag=val
// or
// tag_is_val tag_is_val
// or
// tag:val,tag:val

func SortingTagsFromString(key string) SortingTags {

	var parse_tsg []string
	if strings.Contains(key, DOT_SEPARATOR) {
		parse_tsg = strings.Split(key, DOT_SEPARATOR)
	} else if strings.Contains(key, COMMA_SEPARATOR) {
		parse_tsg = strings.Split(key, COMMA_SEPARATOR)
	} else {
		parse_tsg = strings.Split(key, SPACE_SEPARATOR)
	}
	return SortingTagsFromArray(parse_tsg)
}

func SortingTagsFromArray(keys []string) SortingTags {

	outs := make(SortingTags, 0)

	for _, tgs := range keys {
		spls := strings.Split(tgs, EQUAL_SEPARATOR)
		if len(spls) < 2 {
			// try "_is_"
			spls = strings.Split(tgs, IS_SEPARATOR)
			if len(spls) < 2 {
				// try ":"
				spls = strings.Split(tgs, COLON_SEPARATOR)
				if len(spls) < 2 {
					continue
				}
			}

		}

		outs = append(outs, []string{spls[0], spls[1]})
	}
	return outs
}

func (p SortingTags) Len() int           { return len(p) }
func (p SortingTags) Less(i, j int) bool { return strings.Compare(p[i][0], p[j][0]) < 0 }
func (p SortingTags) Swap(i, j int)      { p[i], p[j] = p[j], p[i] }

func (s SortingTags) String() string {
	return s.ToStringSep(EQUAL_SEPARATOR, SPACE_SEPARATOR)
}

func (s SortingTags) IsEmpty() bool {
	return len(s) == 0
}
func (s SortingTags) Tags() [][]string {
	return s
}

func (s SortingTags) SetTags(tgs [][]string) {
	s = SortingTags(tgs)
}

func (s SortingTags) Merge(tags SortingTags) SortingTags {
	if len(tags) == 0 {
		return s
	}
	if len(s) == 0 {
		return tags
	}
	n_tags := make(SortingTags, 0)
	for _, tag := range tags {
		got := false
		for _, o_tag := range s {
			if tag[0] == o_tag[0] {
				n_tags = append(n_tags, []string{tag[0], tag[1]})
				got = true
				break
			}
		}
		if !got {
			n_tags = append(n_tags, []string{tag[0], tag[1]})
		}
	}
	return n_tags
}

func (s SortingTags) HasAllTags(tags SortingTags) bool {
	if len(tags) == 0 {
		return true
	}
	if len(s) == 0 && len(tags) > 0 {
		return false
	}
	have_ct := 0
	for _, tag := range tags {
		for _, o_tag := range s {
			if tag[0] == o_tag[0] && tag[1] == o_tag[1] {
				have_ct++
			}
		}
	}
	return have_ct == len(tags)
}

func (s SortingTags) ToStringSep(wordsep string, tagsep string) string {
	str := make([]string, len(s))
	for idx, tag := range s {
		str[idx] = strings.Join(tag, wordsep)
	}
	return strings.Join(str, tagsep)
}

func (s SortingTags) WriteBytes(buf io.Writer, wordsep []byte, tagsep []byte) {

	l := len(s)
	for idx, tag := range s {
		buf.Write([]byte(tag[0]))
		buf.Write(wordsep)
		buf.Write([]byte(tag[1]))
		if idx < l-1 {
			buf.Write(tagsep)
		}
	}
}

func (s SortingTags) Find(name string) string {
	for _, tag := range s {
		if tag[0] == name {
			return tag[1]
		}
	}
	return ""
}

func (s SortingTags) Set(name string, val string) SortingTags {
	for idx, tag := range s {
		if tag[0] == name {
			s[idx][1] = val
			return s
		}
	}
	s = append(s, []string{name, val})
	return s
}

// Hz, B, etc
func (s SortingTags) Unit() string {
	return s.Find("unit")
}

// counter, rate, gauge, count, timestamp
func (s SortingTags) Mtype() string {
	got := s.Find("mtype")
	if got == "" {
		got = s.Find("target_type")
	}
	return got
}

// min, max, lower, upper, mean, std, sum, upper_\d+, lower_\d+, min_\d+, max_\d+
func (s SortingTags) Stat() string {
	return s.Find("stat")
}
