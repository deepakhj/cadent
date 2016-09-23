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

/** logger **/

package config

import (
	"gopkg.in/op/go-logging.v1"
	"io"
	"os"
	"strings"
)

var log = logging.MustGetLogger("config")

type LogConfig struct {
	Format string `toml:"format" json:"format,omitempty"`
	File   string `toml:"file" json:"file,omitempty"`
	Level  string `toml:"level" json:"level,omitempty"`
}

func (c *LogConfig) Start() {

	if c.Format == "" {
		c.Format = "%{color}%{time:2006-01-02 15:04:05.000} [%{module}] (%{shortfile}) ▶ %{level:.4s} %{color:reset} %{message}"
	}

	var file_o io.Writer
	var err error
	switch c.File {
	case "stdout":
		file_o = os.Stdout
	case "":
		file_o = os.Stdout
	case "stderr":
		file_o = os.Stderr
	default:

		file_o, err = os.OpenFile(c.File, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if os.IsNotExist(err) {
			err = nil
			file_o, err = os.Create(c.File)
		}
		if err != nil {
			panic(err)
		}
	}
	logBackend := logging.NewLogBackend(file_o, "", 0)
	logging.SetFormatter(logging.MustStringFormatter(c.Format))
	logging.SetBackend(logBackend)

	switch strings.ToUpper(c.Level) {
	case "DEBUG":
		logging.SetLevel(logging.DEBUG, "")
	case "INFO":
		logging.SetLevel(logging.INFO, "")
	case "WARNING":
		logging.SetLevel(logging.WARNING, "")
	case "ERROR":
		logging.SetLevel(logging.ERROR, "")
	case "CRITICAL":
		logging.SetLevel(logging.CRITICAL, "")

	}
}
