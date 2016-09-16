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
   Injectors

   These bypass the normal "line" protocals and consume from "something else"

   and feed them directly into the writer paths

   For now we have a "kafka" injector that consumes the "produced" cadent writer items
*/

package injectors

import (
	"cadent/server/repr"
	"cadent/server/utils/options"
	"cadent/server/writers"
)

/****************** Data writers *********************/
type Injector interface {
	Config(options.Options) error
	Start() error
	Stop() error
	SetWriter(writer writers.Writer) error
}
