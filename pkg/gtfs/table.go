package gtfs

import (
	"fmt"

	"router/pkg/types"
)

type GTFSStopID string
type GTFSTripID string
type GTFSRouteID string
type GTFSAgencyID string
type GTFSServiceID string

type GTFSDate string // YYYYMMDD

type GTFSRoute struct {
	GtfsId       GTFSRouteID
	GtfsAgencyId GTFSAgencyID
	ShortName    string
	LongName     string
	RouteType    uint32
	Color        string
}

type GTFSTrip struct {
	GtfsId        GTFSTripID
	GtfsRouteId   GTFSRouteID
	GtfsServiceId GTFSServiceID
	Headsign      string
}

type GTFSService struct {
	GtfsId      GTFSServiceID
	ActiveOnDay []bool // indexed by time.Weekday
	StartDate   GTFSDate
	EndDate     GTFSDate
}

type GTFSServiceExceptionType int

const (
	NONE GTFSServiceExceptionType = iota
	ADDED
	REMOVED
)

type GTFSServiceException struct {
	GtfsServiceId GTFSServiceID
	Date          GTFSDate
	ExceptionType GTFSServiceExceptionType
}

type GTFSStop struct {
	GtfsId GTFSStopID
	Name   string
	Lat    float64
	Lon    float64
}

type GTFSStopTime struct {
	GtfsTripId    GTFSTripID
	GtfsStopId    GTFSStopID
	ArrivalTime   types.Timestamp
	DepartureTime types.Timestamp
	StopSequence  uint32
}

type GTFSTransfer struct {
	FromStopId      GTFSStopID
	ToStopId        GTFSStopID
	TransferType    uint32
	MinTransferTime uint32
}

type GTFSTable struct {
	routes            []GTFSRoute
	Trips             []GTFSTrip
	Services          []GTFSService
	ServiceExceptions []GTFSServiceException
	Stops             []GTFSStop
	StopTimes         []GTFSStopTime
	Transfers         []GTFSTransfer

	RoutesById                  map[GTFSRouteID]*GTFSRoute
	servicesById                map[GTFSServiceID]*GTFSService
	serviceExceptionsByDateById map[GTFSServiceID]map[GTFSDate]GTFSServiceExceptionType
}

func (gt *GTFSTable) TripsForDate(date GTFSDate) []GTFSTrip {
	var activeTrips []GTFSTrip

	for _, trip := range gt.Trips {
		if gt.serviceActiveOnDate(trip.GtfsServiceId, date) {
			activeTrips = append(activeTrips, trip)
		}
	}

	fmt.Printf("Found %d trips for date %s\n", len(activeTrips), date)

	return activeTrips
}

func (gt *GTFSTable) serviceActiveOnDate(serviceId GTFSServiceID, date GTFSDate) bool {
	activeByCalendar := gt.serviceActiveByCalendar(serviceId, date)
	serviceException := gt.serviceException(serviceId, date)

	return (activeByCalendar && !(serviceException == REMOVED)) || (serviceException == ADDED)
}

func (gt *GTFSTable) serviceActiveByCalendar(serviceId GTFSServiceID, date GTFSDate) bool {
	service, ok := gt.servicesById[serviceId]
	if !ok {
		fmt.Printf("WARN: unknown service_id %q\n", serviceId)
	}

	if !(service.StartDate <= date && date <= service.EndDate) {
		return false
	}

	datetime, err := parseTime(date)
	if err != nil {
		fmt.Printf("WARN: invalid date %s", date)
		return false
	}

	return service.ActiveOnDay[datetime.Weekday()]
}

func (gt *GTFSTable) serviceException(serviceId GTFSServiceID, date GTFSDate) GTFSServiceExceptionType {
	exceptionsByDate, ok := gt.serviceExceptionsByDateById[serviceId]
	if !ok {
		return NONE
	}

	exception, ok := exceptionsByDate[date]
	if !ok {
		return NONE
	}

	return exception
}
