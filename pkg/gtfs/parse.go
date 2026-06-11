package gtfs

import (
	"archive/zip"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strconv"
	"time"

	"router/pkg/types"
)

func ParseGtfs(zipFilePath string) (*GTFSTable, error) {
	start := time.Now()

	println("Parsing GTFS")

	reader, err := zip.OpenReader(zipFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open gtfs zip: %w", err)
	}
	defer reader.Close()

	files := make(map[string]*zip.File)
	for _, file := range reader.File {
		files[file.Name] = file
	}

	var errs []error

	routes, err := parseRoutes(files["routes.txt"])
	errs = append(errs, err)
	trips, err := parseTrips(files["trips.txt"])
	errs = append(errs, err)
	services, err := parseCalendar(files["calendar.txt"])
	errs = append(errs, err)
	serviceExceptions, err := parseCalendarDates(files["calendar_dates.txt"])
	errs = append(errs, err)
	stops, err := parseStops(files["stops.txt"])
	errs = append(errs, err)
	stopTimes, err := parseStopTimes(files["stop_times.txt"])
	errs = append(errs, err)
	transfers, err := parseTransfers(files["transfers.txt"])
	errs = append(errs, err)

	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("GTFS parse errors: %w", err)
	}

	table := GTFSTable{
		routes:            routes,
		Stops:             stops,
		Trips:             trips,
		Services:          services,
		ServiceExceptions: serviceExceptions,
		StopTimes:         stopTimes,
		Transfers:         transfers,

		RoutesById:                  routesById(routes),
		servicesById:                servicesById(services),
		serviceExceptionsByDateById: serviceExceptionsByDateById(serviceExceptions),
	}

	fmt.Printf("GTFS parsing done in %s\n", time.Since(start))

	return &table, nil
}

type RowParser[T any] func(colGetter func(string) string) (*T, error)

func parseCSVFile[T any](f *zip.File, parser RowParser[T]) ([]T, time.Duration, error) {
	start := time.Now()

	if f == nil {
		return nil, 0, nil
	}

	rc, err := f.Open()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to open %s: %w", f.Name, err)
	}
	defer rc.Close()

	reader := csv.NewReader(rc)

	header, err := reader.Read()
	if err != nil {
		return nil, 0, fmt.Errorf("failed to read header %s: %w", f.Name, err)
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
			return nil, 0, fmt.Errorf("failed to read row %s: %w", f.Name, err)
		}

		colGetter := buildColGetter(row)

		parsedValue, err := parser(colGetter)
		if err != nil {
			// TODO: handle rows that don't parse properly
			continue
		}

		results = append(results, *parsedValue)
	}

	return results, time.Since(start), nil
}

func parseRoutes(f *zip.File) ([]GTFSRoute, error) {
	routes, dur, err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSRoute, error) {
		routeType, err := parseUint32(colGetter("route_type"))
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
		fmt.Printf("Found %d routes. %s\n", len(routes), dur)
	}

	return routes, err
}

func parseTrips(f *zip.File) ([]GTFSTrip, error) {
	trips, dur, err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSTrip, error) {
		return &GTFSTrip{
			GtfsId:        GTFSTripID(colGetter("trip_id")),
			GtfsRouteId:   GTFSRouteID(colGetter("route_id")),
			GtfsServiceId: GTFSServiceID(colGetter("service_id")),
			Headsign:      colGetter("trip_headsign"),
		}, nil
	})
	if err == nil {
		fmt.Printf("Found %d trips. %s\n", len(trips), dur)
	}

	return trips, err
}

func parseCalendar(f *zip.File) ([]GTFSService, error) {
	services, dur, err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSService, error) {
		activeOnDay := make([]bool, 7)
		activeOnDay[time.Monday] = parseBool(colGetter("monday"))
		activeOnDay[time.Tuesday] = parseBool(colGetter("tuesday"))
		activeOnDay[time.Wednesday] = parseBool(colGetter("wednesday"))
		activeOnDay[time.Thursday] = parseBool(colGetter("thursday"))
		activeOnDay[time.Friday] = parseBool(colGetter("friday"))
		activeOnDay[time.Saturday] = parseBool(colGetter("saturday"))
		activeOnDay[time.Sunday] = parseBool(colGetter("sunday"))

		return &GTFSService{
			GtfsId:      GTFSServiceID(colGetter("service_id")),
			ActiveOnDay: activeOnDay,
			StartDate:   GTFSDate(colGetter("start_date")),
			EndDate:     GTFSDate(colGetter("end_date")),
		}, nil
	})
	if err == nil {
		fmt.Printf("Found %d services. %s\n", len(services), dur)
	}

	return services, err
}

func parseCalendarDates(f *zip.File) ([]GTFSServiceException, error) {
	serviceExceptions, dur, err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSServiceException, error) {
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
		fmt.Printf("Found %d service exceptions. %s\n", len(serviceExceptions), dur)
	}

	return serviceExceptions, err
}

func parseStops(f *zip.File) ([]GTFSStop, error) {
	stops, dur, err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSStop, error) {
		lat, err := parseFloat64(colGetter("stop_lat"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse stop_lat: %w", err)
		}

		lon, err := parseFloat64(colGetter("stop_lon"))
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
		fmt.Printf("Found %d stops. %s\n", len(stops), dur)
	}

	return stops, err
}

func parseStopTimes(f *zip.File) ([]GTFSStopTime, error) {
	stopTimes, dur, err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSStopTime, error) {
		arrivalTime, err := gtfsTimeToSeconds(colGetter("arrival_time"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse arrival_time: %w", err)
		}

		departureTime, err := gtfsTimeToSeconds(colGetter("departure_time"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse departure_time: %w", err)
		}

		stopSequence, err := parseUint32(colGetter("stop_sequence"))
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
		fmt.Printf("Found %d stop times. %s\n", len(stopTimes), dur)
	}

	return stopTimes, err
}

func parseTransfers(f *zip.File) ([]GTFSTransfer, error) {
	transfers, dur, err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSTransfer, error) {
		transferType, err := parseUint32(colGetter("transfer_type"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse transfer_type: %w", err)
		}

		minTransferTime, err := parseUint32(colGetter("min_transfer_time"))
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
		fmt.Printf("Found %d transfers. %s\n", len(transfers), dur)
	}

	return transfers, err
}
