package transit

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"router/pkg/types"
	"strconv"
	"strings"
	"time"
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
	Routes            []GTFSRoute
	Trips             []GTFSTrip
	services          map[GTFSServiceID]*GTFSService
	serviceExceptions map[GTFSServiceID]map[GTFSDate]GTFSServiceExceptionType
	Stops             []GTFSStop
	StopTimes         []GTFSStopTime
	Transfers         []GTFSTransfer
}

func (gt *GTFSTable) RoutesById() map[GTFSRouteID]*GTFSRoute {
	gtfsRouteMap := make(map[GTFSRouteID]*GTFSRoute, len(gt.Routes))
	for i := range gt.Routes {
		gtfsRouteMap[gt.Routes[i].GtfsId] = &gt.Routes[i]
	}

	return gtfsRouteMap
}

func (gt *GTFSTable) TripsForDate(date GTFSDate) []GTFSTrip {
	var activeTrips []GTFSTrip

	for _, trip := range gt.Trips {
		if gt.ServiceActiveOnDate(trip.GtfsServiceId, date) {
			activeTrips = append(activeTrips, trip)
		}
	}

	fmt.Printf("Found %d trips for date %s\n", len(activeTrips), date)

	return activeTrips
}

func (gt *GTFSTable) ServiceActiveOnDate(serviceId GTFSServiceID, date GTFSDate) bool {
	activeByCalendar := gt.serviceActiveByCalendar(serviceId, date)
	serviceException := gt.serviceException(serviceId, date)

	return (activeByCalendar && !(serviceException == REMOVED)) || (serviceException == ADDED)
}

func servicesById(services []GTFSService) map[GTFSServiceID]*GTFSService {
	gtfsServiceMap := make(map[GTFSServiceID]*GTFSService, len(services))
	for i := range services {
		gtfsServiceMap[services[i].GtfsId] = &services[i]
	}

	return gtfsServiceMap
}

func serviceExceptionsByDateById(
	serviceExceptions []GTFSServiceException,
) map[GTFSServiceID]map[GTFSDate]GTFSServiceExceptionType {
	gtfsServiceExceptionMap := make(map[GTFSServiceID]map[GTFSDate]GTFSServiceExceptionType, len(serviceExceptions))
	for _, serviceException := range serviceExceptions {
		_, ok := gtfsServiceExceptionMap[serviceException.GtfsServiceId]
		if !ok {
			gtfsServiceExceptionMap[serviceException.GtfsServiceId] = make(map[GTFSDate]GTFSServiceExceptionType)
		}

		gtfsServiceExceptionMap[serviceException.GtfsServiceId][serviceException.Date] = serviceException.ExceptionType
	}

	return gtfsServiceExceptionMap
}

func (gt *GTFSTable) serviceActiveByCalendar(serviceId GTFSServiceID, date GTFSDate) bool {
	service, ok := gt.services[serviceId]
	if !ok {
		fmt.Printf("WARN: unknown service_id %q\n", serviceId)
	}

	if !(service.StartDate <= date && date <= service.EndDate) {
		return false
	}

	datetime, err := toTime(date)
	if err != nil {
		fmt.Printf("WARN: invalid date %s", date)
		return false
	}

	return service.ActiveOnDay[datetime.Weekday()]
}

func (gt *GTFSTable) serviceException(serviceId GTFSServiceID, date GTFSDate) GTFSServiceExceptionType {
	exceptionsByDate, ok := gt.serviceExceptions[serviceId]
	if !ok {
		return NONE
	}

	exception, ok := exceptionsByDate[date]
	if !ok {
		return NONE
	}

	return exception
}

func toUint32(s string) (uint32, error) {
	val, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid uint %s: %w", s, err)
	}

	return uint32(val), nil
}

func toFloat64(s string) (float64, error) {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float: %w", err)
	}

	return val, nil
}

func toBool(s string) bool {
	return s == "1"
}

func toTime(date GTFSDate) (time.Time, error) {
	s := string(date)
	year, err1 := strconv.Atoi(s[:4])
	month, err2 := strconv.Atoi(s[4:6])
	day, err3 := strconv.Atoi(s[6:])

	if err1 != nil || err2 != nil || err3 != nil {
		return time.Time{}, fmt.Errorf("invalid date: %s", s)
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}

func ToGTFSDate(time time.Time) GTFSDate {
	year := time.Year()
	month := time.Month()
	day := time.Day()

	return GTFSDate(fmt.Sprintf("%04d%02d%02d", year, month, day))
}

func gtfsTimeToSeconds(gtfsTime string) (uint32, error) {
	parts := strings.Split(gtfsTime, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid GTFS time format: %s", gtfsTime)
	}

	hours, err := toUint32(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hours in GTFS time: %w", err)
	}

	minutes, err := toUint32(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid minutes in GTFS time: %w", err)
	}

	seconds, err := toUint32(parts[2])
	if err != nil {
		return 0, fmt.Errorf("invalid seconds in GTFS time: %w", err)
	}

	// Calculate total seconds, allowing hours > 24
	totalSeconds := hours*3600 + minutes*60 + seconds

	return totalSeconds, nil
}

type RowParser[T any] func(colGetter func(string) string) (*T, error)

func parseCSVFile[T any](f *zip.File, parser RowParser[T]) ([]T, error) {
	if f == nil {
		return nil, nil
	}

	rc, err := f.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open %s: %w", f.Name, err)
	}
	defer rc.Close()

	reader := csv.NewReader(rc)

	header, err := reader.Read()
	if err != nil {
		return nil, fmt.Errorf("failed to read header %s: %w", f.Name, err)
	}

	colIdx := make(map[string]int, len(header))
	for i, h := range header {
		colIdx[h] = i
	}

	buildColGetter := func(row []string) func(string) string {
		return func(colName string) string {
			i, ok := colIdx[colName]
			if ok {
				return row[i]
			}

			return ""
		}
	}

	var results []T

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("failed to read row %s: %w", f.Name, err)
		}

		colGetter := buildColGetter(row)

		parsedValue, err := parser(colGetter)
		if err != nil {
			// TODO: handle rows that don't parse properly
			continue
		}

		results = append(results, *parsedValue)
	}

	return results, nil
}

func parseRoutes(f *zip.File) ([]GTFSRoute, error) {
	println("Parsing routes")

	routes, err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSRoute, error) {
		routeType, err := toUint32(colGetter(("route_type")))
		if err != nil {
			return nil, fmt.Errorf("failed to parse route_type: %w", err)
		}

		return &GTFSRoute{
			GtfsId:       GTFSRouteID(colGetter("route_id")),
			GtfsAgencyId: GTFSAgencyID(colGetter("agency_id")),
			ShortName:    colGetter("route_short_name"),
			LongName:     colGetter("route_long_name"),
			RouteType:    routeType,
			Color:        colGetter("route_color"),
		}, nil
	})
	if err == nil {
		fmt.Printf("Found %d routes\n", len(routes))
	}

	return routes, err
}

func parseTrips(f *zip.File) ([]GTFSTrip, error) {
	println("Parsing trips")

	trips, err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSTrip, error) {
		return &GTFSTrip{
			GtfsId:        GTFSTripID(colGetter("trip_id")),
			GtfsRouteId:   GTFSRouteID(colGetter("route_id")),
			GtfsServiceId: GTFSServiceID(colGetter("service_id")),
			Headsign:      colGetter("trip_headsign"),
		}, nil
	})
	if err == nil {
		fmt.Printf("Found %d trips\n", len(trips))
	}

	return trips, err
}

func parseCalendar(f *zip.File) ([]GTFSService, error) {
	println("Parsing calendar")

	services, err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSService, error) {
		activeOnDay := make([]bool, 7)
		activeOnDay[time.Monday] = toBool(colGetter("monday"))
		activeOnDay[time.Tuesday] = toBool(colGetter("tuesday"))
		activeOnDay[time.Wednesday] = toBool(colGetter("wednesday"))
		activeOnDay[time.Thursday] = toBool(colGetter("thursday"))
		activeOnDay[time.Friday] = toBool(colGetter("friday"))
		activeOnDay[time.Saturday] = toBool(colGetter("saturday"))
		activeOnDay[time.Sunday] = toBool(colGetter("sunday"))

		return &GTFSService{
			GtfsId:      GTFSServiceID(colGetter("service_id")),
			ActiveOnDay: activeOnDay,
			StartDate:   GTFSDate(colGetter("start_date")),
			EndDate:     GTFSDate(colGetter("end_date")),
		}, nil
	})
	if err == nil {
		fmt.Printf("Found %d services\n", len(services))
	}

	return services, err
}

func parseCalendarDates(f *zip.File) ([]GTFSServiceException, error) {
	println("Parsing calendar dates")

	serviceExceptions, err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSServiceException, error) {
		exceptionType, err := strconv.Atoi(colGetter("exception_type"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse exception_type: %w", err)
		}

		return &GTFSServiceException{
			GtfsServiceId: GTFSServiceID(colGetter("service_id")),
			Date:          GTFSDate(colGetter("date")),
			ExceptionType: GTFSServiceExceptionType(exceptionType),
		}, nil
	})
	if err == nil {
		fmt.Printf("Found %d service exceptions\n", len(serviceExceptions))
	}

	return serviceExceptions, err
}

func parseStops(f *zip.File) ([]GTFSStop, error) {
	println("Parsing stops")

	stops, err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSStop, error) {
		lat, err := toFloat64(colGetter("stop_lat"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse stop_lat: %w", err)
		}

		lon, err := toFloat64(colGetter("stop_lon"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse stop_lon: %w", err)
		}

		return &GTFSStop{
			GtfsId: GTFSStopID(colGetter("stop_id")),
			Name:   colGetter("stop_name"),
			Lat:    lat,
			Lon:    lon,
		}, nil
	})
	if err == nil {
		fmt.Printf("Found %d stops\n", len(stops))
	}

	return stops, err
}

func parseStopTimes(f *zip.File) ([]GTFSStopTime, error) {
	println("Parsing stop times")

	stopTimes, err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSStopTime, error) {
		arrivalTime, err := gtfsTimeToSeconds(colGetter("arrival_time"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse arrival_time: %w", err)
		}

		departureTime, err := gtfsTimeToSeconds(colGetter("departure_time"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse departure_time: %w", err)
		}

		stopSequence, err := toUint32(colGetter("stop_sequence"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse stop_sequence: %w", err)
		}

		return &GTFSStopTime{
			GtfsStopId:    GTFSStopID(colGetter("stop_id")),
			GtfsTripId:    GTFSTripID(colGetter("trip_id")),
			ArrivalTime:   types.Timestamp(arrivalTime),
			DepartureTime: types.Timestamp(departureTime),
			StopSequence:  stopSequence,
		}, nil
	})
	if err == nil {
		fmt.Printf("Found %d stop times\n", len(stopTimes))
	}

	return stopTimes, err
}

func parseTransfers(f *zip.File) ([]GTFSTransfer, error) {
	println("Parsing transfers")

	transfers, err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSTransfer, error) {
		transferType, err := toUint32(colGetter("transfer_type"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse transfer_type: %w", err)
		}

		minTransferTime, err := toUint32(colGetter("min_transfer_time"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse min_transfer_time: %w", err)
		}

		return &GTFSTransfer{
			FromStopId: GTFSStopID(colGetter("from_stop_id")),
			ToStopId:   GTFSStopID(colGetter("to_stop_id")),

			TransferType:    transferType,
			MinTransferTime: minTransferTime,
		}, nil
	})
	if err == nil {
		fmt.Printf("Found %d transfers\n", len(transfers))
	}

	return transfers, err
}

func ParseGtfs(zipFilePath string) (*GTFSTable, error) {
	reader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	defer reader.Close()

	files := make(map[string]*zip.File)
	for _, file := range reader.File {
		files[file.Name] = file
	}

	routes, err := parseRoutes(files["routes.txt"])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse routes: %w", err)
	}

	trips, err := parseTrips(files["trips.txt"])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse trips: %w", err)
	}

	services, err := parseCalendar(files["calendar.txt"])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse calendar: %w", err)
	}

	serviceExceptions, err := parseCalendarDates(files["calendar_dates.txt"])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse calendar dates: %w", err)
	}

	stops, err := parseStops(files["stops.txt"])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse stops: %w", err)
	}

	stopTimes, err := parseStopTimes(files["stop_times.txt"])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse stop times: %w", err)
	}

	transfers, err := parseTransfers(files["transfers.txt"])
	if err != nil {
		return nil, fmt.Errorf("Failed to parse transfers: %w", err)
	}

	return &GTFSTable{
		Routes:            routes,
		Stops:             stops,
		Trips:             trips,
		services:          servicesById(services),
		serviceExceptions: serviceExceptionsByDateById(serviceExceptions),
		StopTimes:         stopTimes,
		Transfers:         transfers,
	}, nil
}
