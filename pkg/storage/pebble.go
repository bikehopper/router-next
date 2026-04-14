package storage

import (
	"encoding/json"

	"github.com/cockroachdb/pebble"
)

// Open initializes and returns a new Pebble DB instance.
func Open(path string) (*pebble.DB, error) {
	db, err := pebble.Open(path, &pebble.Options{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

// PutJSON is a generic wrapper that JSON encodes the provided data and saves it
// under the constructed key (prefix + id).
func PutJSON(db *pebble.DB, prefix, id string, data any) error {
	key := []byte(prefix + id)

	val, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return db.Set(key, val, pebble.NoSync)
}

// GetJSON retrieves a key from Pebble and unmarshals it into the target struct natively.
func GetJSON(db *pebble.DB, prefix, id string, target any) error {
	key := []byte(prefix + id)

	val, closer, err := db.Get(key)
	if err != nil {
		return err
	}
	defer closer.Close()

	return json.Unmarshal(val, target)
}
