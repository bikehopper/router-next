package storage

import (
	"encoding/json"

	"github.com/cockroachdb/pebble"
)

func Open(path string) (*pebble.DB, error) {
	db, err := pebble.Open(path, &pebble.Options{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func PutJSON(db *pebble.DB, prefix, id string, data any) error {
	key := []byte(prefix + id)

	val, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return db.Set(key, val, pebble.NoSync)
}
