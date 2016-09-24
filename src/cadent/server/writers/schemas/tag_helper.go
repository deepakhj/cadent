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
	something to take the probuf MetricTag to the [][]string we want
*/

package schemas

import "cadent/server/repr"

func ToMetricTag(t repr.SortingTags) []*MetricTag {
	o := make([]*MetricTag, len(t), len(t))
	for idx, tg := range t {
		o[idx] = &MetricTag{tg}
	}
	return o
}

func (t *MetricName) TagToSorted() repr.SortingTags {
	str := make([][]string, len(t.Tags), len(t.Tags))
	for idx, tg := range t.Tags {
		str[idx] = tg.Tag
	}
	return repr.SortingTags(str)
}

func (t *MetricName) MetaTagToSorted() repr.SortingTags {
	str := make([][]string, len(t.MetaTags), len(t.MetaTags))
	for idx, tg := range t.MetaTags {
		str[idx] = tg.Tag
	}
	return repr.SortingTags(str)
}

func (t *SeriesMetric) TagToSorted() repr.SortingTags {
	str := make([][]string, len(t.Tags), len(t.Tags))
	for idx, tg := range t.Tags {
		str[idx] = tg.Tag
	}
	return repr.SortingTags(str)
}

func (t *SeriesMetric) SingleMetric() repr.SortingTags {
	str := make([][]string, len(t.MetaTags), len(t.MetaTags))
	for idx, tg := range t.MetaTags {
		str[idx] = tg.Tag
	}
	return repr.SortingTags(str)
}

func (t *SingleMetric) TagToSorted() repr.SortingTags {
	str := make([][]string, len(t.Tags), len(t.Tags))
	for idx, tg := range t.Tags {
		str[idx] = tg.Tag
	}
	return repr.SortingTags(str)
}

func (t *SingleMetric) MetaTagToSorted() repr.SortingTags {
	str := make([][]string, len(t.MetaTags), len(t.MetaTags))
	for idx, tg := range t.MetaTags {
		str[idx] = tg.Tag
	}
	return repr.SortingTags(str)
}

func (t *UnProcessedMetric) TagToSorted() repr.SortingTags {
	str := make([][]string, len(t.Tags), len(t.Tags))
	for idx, tg := range t.Tags {
		str[idx] = tg.Tag
	}
	return repr.SortingTags(str)
}

func (t *UnProcessedMetric) MetaTagToSorted() repr.SortingTags {
	str := make([][]string, len(t.MetaTags), len(t.MetaTags))
	for idx, tg := range t.MetaTags {
		str[idx] = tg.Tag
	}
	return repr.SortingTags(str)
}

func (t *RawMetric) TagToSorted() repr.SortingTags {
	str := make([][]string, len(t.Tags), len(t.Tags))
	for idx, tg := range t.Tags {
		str[idx] = tg.Tag
	}
	return repr.SortingTags(str)
}

func (t *RawMetric) MetaTagToSorted() repr.SortingTags {
	str := make([][]string, len(t.MetaTags), len(t.MetaTags))
	for idx, tg := range t.MetaTags {
		str[idx] = tg.Tag
	}
	return repr.SortingTags(str)
}
