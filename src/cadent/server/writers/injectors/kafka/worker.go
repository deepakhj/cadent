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
   Workers for Kafka injector

   There are "4" basic workers each does something different

   1. RawMetric Worker -- takes a raw metric/value and injects it into an Accumulator loop
   2. UnProcessMetric Worker -- a non-process/accumulated repr metric (min/max/sum/last/count) and injects into Accumulator
   3. ProcessMetric Worker -- takes a "processed" (or accumulated) metric and injects it into a Writer
   4. SeriesMetric Worker -- takes a "processed series" (binary blob of many metrics) and injects it into a writer

*/

package kafka

import (
	"cadent/server/accumulator"
	"cadent/server/utils/options"
	"cadent/server/writers/schemas"
	"fmt"
)

type Worker interface {
	Config(options.Options) error
	SetAccumulator(acc *accumulator.Accumulator)
	GetAccumulator() *accumulator.Accumulator
	DoWork(metric schemas.KMessageBase) error
}

/****************** RawMetric worker *********************/
type EchoWorker struct {
	Name string

	accumulator *accumulator.Accumulator
}

func (w *EchoWorker) Config(options.Options) error {
	return nil
}

func (w *EchoWorker) SetAccumulator(acc *accumulator.Accumulator) {
	w.accumulator = acc
}

func (w *EchoWorker) GetAccumulator() *accumulator.Accumulator {
	return w.accumulator
}

func (w *EchoWorker) DoWork(metric schemas.KMessageBase) error {

	switch metric.(type) {
	case *schemas.KRawMetric:
		fmt.Println(metric.(*schemas.KRawMetric).Repr())
		return nil
	case *schemas.KUnProcessedMetric:
		fmt.Println(metric.(*schemas.KUnProcessedMetric).Repr())
		return nil
	case *schemas.KSingleMetric:
		fmt.Println(metric.(*schemas.KSingleMetric).Repr())
		return nil
	case *schemas.KSeriesMetric:
		fmt.Println("metric series: " + metric.(*schemas.KSeriesMetric).Metric)
		return nil
	default:
		return ErrorBadMessageType
	}
}

/****************** RawMetric worker *********************/
type RawMetricWorker struct {
	Name string

	accumulator *accumulator.Accumulator
}

func (w *RawMetricWorker) Config(options.Options) error {
	return nil
}

func (w *RawMetricWorker) SetAccumulator(acc *accumulator.Accumulator) {
	w.accumulator = acc
}

func (w *RawMetricWorker) GetAccumulator() *accumulator.Accumulator {
	return w.accumulator
}

func (w *RawMetricWorker) DoWork(metric schemas.KMessageBase) error {

	switch metric.(type) {
	case *schemas.KRawMetric:
		if w.accumulator == nil {
			return ErrorAccumulatorNotDefined
		}
		w.accumulator.PushStat(metric.(*schemas.KRawMetric).Repr())
		return nil
	default:
		return ErrorBadMessageType
	}
}

/****************** UnProcessed worker *********************/
type UnProcessedMetricWorker struct {
	Name string

	accumulator *accumulator.Accumulator
}

func (w *UnProcessedMetricWorker) Config(options.Options) error {
	return nil
}

func (w *UnProcessedMetricWorker) SetAccumulator(acc *accumulator.Accumulator) {
	w.accumulator = acc
}

func (w *UnProcessedMetricWorker) GetAccumulator() *accumulator.Accumulator {
	return w.accumulator
}

func (w *UnProcessedMetricWorker) DoWork(metric schemas.KMessageBase) error {

	switch metric.(type) {
	case *schemas.KUnProcessedMetric:
		if w.accumulator == nil {
			return ErrorAccumulatorNotDefined
		}
		w.accumulator.PushStat(metric.(*schemas.KUnProcessedMetric).Repr())
		return nil
	default:
		return ErrorBadMessageType
	}
}

/****************** UnProcessed worker *********************/
type AnyMetricWorker struct {
	Name string

	accumulator *accumulator.Accumulator
}

func (w *AnyMetricWorker) Config(options.Options) error {
	return nil
}

func (w *AnyMetricWorker) SetAccumulator(acc *accumulator.Accumulator) {
	w.accumulator = acc
}

func (w *AnyMetricWorker) GetAccumulator() *accumulator.Accumulator {
	return w.accumulator
}

func (w *AnyMetricWorker) DoWork(metric schemas.KMessageBase) error {
	if w.accumulator == nil {
		return ErrorAccumulatorNotDefined
	}
	switch metric.(type) {
	case *schemas.KMetric:
		m := metric.(*schemas.KMetric)
		r := m.Repr()
		if r != nil {
			w.accumulator.PushStat(r)
		}
		return nil
	case *schemas.KRawMetric:
		w.accumulator.PushStat(metric.(*schemas.KRawMetric).Repr())
		return nil
	case *schemas.KUnProcessedMetric:
		w.accumulator.PushStat(metric.(*schemas.KUnProcessedMetric).Repr())
		return nil
	case *schemas.KSingleMetric:
		w.accumulator.PushStat(metric.(*schemas.KSingleMetric).Repr())
		return nil
	//case *schemas.KSeriesMetric:
	// TODO
	//	return nil

	default:
		return ErrorBadMessageType
	}
}
