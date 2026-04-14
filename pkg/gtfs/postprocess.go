package gtfs

import (
	"router/pkg/storage"

	"github.com/cockroachdb/pebble"
)

func GetRoute(db *pebble.DB, id GTFSRouteID) (*GTFSRoute, error) {
	var route GTFSRoute
	err := storage.GetJSON(db, "route:", string(id), &route)
	return &route, err
}

func GetService(db *pebble.DB, id GTFSServiceID) (*GTFSService, error) {
	var service GTFSService
	err := storage.GetJSON(db, "service:", string(id), &service)
	return &service, err
}

func GetServiceException(db *pebble.DB, serviceId GTFSServiceID, date GTFSDate) (*GTFSServiceException, error) {
	var exception GTFSServiceException
	id := string(serviceId) + ":" + string(date)
	err := storage.GetJSON(db, "service_exception:", id, &exception)
	return &exception, err
}

// replaces iterating over maps[serviceID] for values
// caller MUST invoke iter.Close() when finished
func GetServiceExceptionsIterator(db *pebble.DB, serviceId GTFSServiceID) (*pebble.Iterator, []byte, error) {
	prefix := []byte("service_exception:" + string(serviceId) + ":")

	iter, err := db.NewIter(nil)
	if err != nil {
		return nil, nil, err
	}

	iter.SeekGE(prefix)
	return iter, prefix, nil
}
