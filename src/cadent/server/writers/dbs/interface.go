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
	}

	DB_REGISTRY[hook_key] = db
	return db, nil
}

// make it on startup
func init() {
	DB_REGISTRY = make(map[string]DB)
}
