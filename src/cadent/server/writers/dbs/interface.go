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
   little DB abstractor
*/

package dbs

// just a dummy interface .. casting will be required for real life usage
type DBConn interface{}

/****************** Data writers *********************/
type DB interface {
	Config(map[string]interface{}) error
	Connection() DBConn
}

type DBRegistry map[string]DB

// singleton
var DB_REGISTRY DBRegistry

func NewDB(dbtype string, dbkey string, config map[string]interface{}) (DB, error) {

	hook_key := dbtype + dbkey
	gots := DB_REGISTRY[hook_key]
	if gots != nil {
		return gots, nil
	}

	var db DB
	switch dbtype {
	case "mysql":
		db = NewMySQLDB()
		err := db.Config(config)
		if err != nil {
			return nil, err
		}
	case "cassandra":
		db = NewCassandraDB()
		err := db.Config(config)
		if err != nil {
			return nil, err
		}
	case "kafka":
		db = NewKafkaDB()
		err := db.Config(config)
		if err != nil {
			return nil, err
		}
	case "leveldb":
		db = NewLevelDB()
		err := db.Config(config)
		if err != nil {
			return nil, err
		}
	}

	DB_REGISTRY[hook_key] = db
	return db, nil
}

// make it on startup
func init() {
	DB_REGISTRY = make(map[string]DB)
}
