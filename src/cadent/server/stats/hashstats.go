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

//  stat object for the consthash servers

package stats

import "sync"

//helper object for json'ing the basic stat data
type HashServerStats struct {
	Mu *sync.RWMutex

	ValidLineCount       int64 `json:"valid_line_count"`
	WorkerValidLineCount int64 `json:"worker_line_count"`
	InvalidLineCount     int64 `json:"invalid_line_count"`
	SuccessSendCount     int64 `json:"success_send_count"`
	FailSendCount        int64 `json:"fail_send_count"`
	UnsendableSendCount  int64 `json:"unsendable_send_count"`
	UnknownSendCount     int64 `json:"unknown_send_count"`
	AllLinesCount        int64 `json:"all_lines_count"`
	RedirectedLinesCount int64 `json:"redirected_lines_count"`
	RejectedLinesCount   int64 `json:"rejected_lines_count"`
	BytesWrittenCount    int64 `json:"bytes_written"`
	BytesReadCount       int64 `json:"bytes_read"`

	CurrentValidLineCount       int64 `json:"current_valid_line_count"`
	CurrentWorkerValidLineCount int64 `json:"current_worker_line_count"`
	CurrentInvalidLineCount     int64 `json:"current_invalid_line_count"`
	CurrentSuccessSendCount     int64 `json:"current_success_send_count"`
	CurrentFailSendCount        int64 `json:"current_fail_send_count"`
	CurrentUnsendableSendCount  int64 `json:"current_unsendable_send_count"`
	CurrentUnknownSendCount     int64 `json:"current_unknown_send_count"`
	CurrentAllLinesCount        int64 `json:"current_all_lines_count"`
	CurrentRejectedLinesCount   int64 `json:"current_rejected_lines_count"`
	CurrentRedirectedLinesCount int64 `json:"current_redirected_lines_count"`
	CurrentBytesReadCount       int64 `json:"current_bytes_read_count"`
	CurrentBytesWrittenCount    int64 `json:"current_bytes_written_count"`

	ValidLineCountList       []int64 `json:"valid_line_count_list"`
	WorkerValidLineCountList []int64 `json:"worker_line_count_list"`
	InvalidLineCountList     []int64 `json:"invalid_line_count_list"`
	SuccessSendCountList     []int64 `json:"success_send_count_list"`
	FailSendCountList        []int64 `json:"fail_send_count_list"`
	UnsendableSendCountList  []int64 `json:"unsendable_send_count_list"`
	UnknownSendCountList     []int64 `json:"unknown_send_count_list"`
	AllLinesCountList        []int64 `json:"all_lines_count_list"`
	RedirectedCountList      []int64 `json:"redirected_lines_count_list"`
	RejectedCountList        []int64 `json:"rejected_lines_count_list"`
	BytesReadCountList       []int64 `json:"bytes_read_count_list"`
	BytesWrittenCountList    []int64 `json:"bytes_written_count_list"`
	GoRoutinesList           []int   `json:"go_routines_list"`
	TicksList                []int64 `json:"ticks_list"`

	GoRoutines                 int      `json:"go_routines"`
	UpTimeSeconds              int64    `json:"uptime_sec"`
	ValidLineCountPerSec       float32  `json:"valid_line_count_persec"`
	WorkerValidLineCountPerSec float32  `json:"worker_line_count_persec"`
	InvalidLineCountPerSec     float32  `json:"invalid_line_count_persec"`
	SuccessSendCountPerSec     float32  `json:"success_send_count_persec"`
	UnsendableSendCountPerSec  float32  `json:"unsendable_count_persec"`
	UnknownSendCountPerSec     float32  `json:"unknown_send_count_persec"`
	AllLinesCountPerSec        float32  `json:"all_lines_count_persec"`
	RedirectedLinesCountPerSec float32  `json:"redirected_lines_count_persec"`
	RejectedLinesCountPerSec   float32  `json:"rejected_lines_count_persec"`
	BytesReadCountPerSec       float32  `json:"bytes_read_count_persec"`
	BytesWrittenCountPerSec    float32  `json:"bytes_written_count_persec"`
	Listening                  string   `json:"listening"`
	ServersUp                  []string `json:"servers_up"`
	ServersDown                []string `json:"servers_down"`
	ServersChecks              []string `json:"servers_checking"`

	CurrentReadBufferSize int64 `json:"current_read_buffer_size"`
	MaxReadBufferSize     int64 `json:"max_read_buffer_size"`
	InputQueueSize        int   `json:"input_queue_size"`
	WorkQueueSize         int   `json:"work_queue_size"`
}
