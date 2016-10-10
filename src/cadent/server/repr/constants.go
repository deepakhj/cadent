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
   Some constants/globa vars
*/

package repr

// for GC purposes
const SPACE_SEPARATOR = " "
const DOUBLE_SPACE_SEPARATOR = "  "
const DOT_SEPARATOR = "."
const COMMA_SEPARATOR = ","
const EQUAL_SEPARATOR = "="
const COLON_SEPARATOR = ":"
const IS_SEPARATOR = "_is_"
const UNDERSCORE_SEPARATOR = "_"
const DASH_SEPARATOR = "-"
const NEWLINE_SEPARATOR = "\n"
const DATAGRAM_SEPARATOR = "|#"
const SPACES_STRING = "\r\n\t "

var SPACE_SEPARATOR_BYTE = []byte(" ")
var DOUBLE_SPACE_SEPARATOR_BYTE = []byte("  ")
var DOT_SEPARATOR_BYTE = []byte(".")
var COLON_SEPARATOR_BYTE = []byte(":")
var COMMA_SEPARATOR_BYTE = []byte(",")
var EQUAL_SEPARATOR_BYTE = []byte("=")
var IS_SEPARATOR_BYTE = []byte("_is_")
var NEWLINE_SEPARATOR_BYTES = []byte("\n")
var NEWLINE_SEPARATOR_BYTE = NEWLINE_SEPARATOR_BYTES[0]
var DASH_SEPARATOR_BYTES = []byte("-")
var UNDERSCORE_SEPARATOR_BYTES = []byte("_")
var DATAGRAM_SEPARATOR_BYTES = []byte("|#")
var SPACES_BYTES = []byte("\r\n\t ")

const (
	TAG_METRICS2 = TagMode_METRICS2
	TAG_ALLTAGS  = TagMode_ALL
)
