package gtfs

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strconv"
	"sync"
	"time"

	"router/pkg/storage"
	"router/pkg/types"

	"github.com/cockroachdb/pebble"
)

func ParseGtfs(zipFilePath string, db *pebble.DB) (*GTFSTable, error) {
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

	var wg sync.WaitGroup
	errChan := make(chan error, 7)

	runParseTask(&wg, errChan, "routes", db, files["routes.txt"], parseRoutes)
	runParseTask(&wg, errChan, "trips", db, files["trips.txt"], parseTrips)
	runParseTask(&wg, errChan, "calendar", db, files["calendar.txt"], parseCalendar)
	runParseTask(&wg, errChan, "calendar_dates", db, files["calendar_dates.txt"], parseCalendarDates)
	runParseTask(&wg, errChan, "stops", db, files["stops.txt"], parseStops)
	runParseTask(&wg, errChan, "stop_times", db, files["stop_times.txt"], parseStopTimes)
	runParseTask(&wg, errChan, "transfers", db, files["transfers.txt"], parseTransfers)

	wg.Wait()
	close(errChan)

	for err := range errChan {
		return nil, err
	}

	// Returning empty GTFSTable per phase 1 plan, so consuming callers don't panic on nil.
	return &GTFSTable{}, nil
}

func runParseTask(wg *sync.WaitGroup, errChan chan<- error, prefix string, db *pebble.DB, file *zip.File, task func(*zip.File, *pebble.DB) error) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := task(file, db); err != nil {
			errChan <- fmt.Errorf("%s: %w", prefix, err)
		}
	}()
}

type RowParser[T any] func(colGetter func(string) string) (*T, error)

func parseCSVFile[T any](f *zip.File, parser RowParser[T], processor func(T) error) error {
	if f == nil {
		return nil
	}

	rc, err := f.Open()
	if err != nil {
		return fmt.Errorf("failed to open %s: %w", f.Name, err)
	}
	defer rc.Close()

	reader := csv.NewReader(rc)

	header, err := reader.Read()
	if err != nil {
		return fmt.Errorf("failed to read header %s: %w", f.Name, err)
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

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read row %s: %w", f.Name, err)
		}

		colGetter := buildColGetter(row)
		parsedValue, err := parser(colGetter)
		if err != nil {
			// TODO: handle rows that don't parse properly
			continue
		}

		if err := processor(*parsedValue); err != nil {
			return fmt.Errorf("failed to process row: %w", err)
		}
	}

	return nil
}

func parseRoutes(f *zip.File, db *pebble.DB) error {
	println("Parsing routes")
	count := 0

	err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSRoute, error) {
		routeType, err := parseUint32(colGetter(("route_type")))
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
	}, func(route GTFSRoute) error {
		count++
		return storage.PutJSON(db, "route:", string(route.GtfsId), route)
	})

	if err == nil {
		fmt.Printf("Parsed %d routes\n", count)
	}
	return err
}

func parseTrips(f *zip.File, db *pebble.DB) error {
	println("Parsing trips")
	count := 0

	err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSTrip, error) {
		return &GTFSTrip{
			GtfsId:        GTFSTripID(colGetter("trip_id")),
			GtfsRouteId:   GTFSRouteID(colGetter("route_id")),
			GtfsServiceId: GTFSServiceID(colGetter("service_id")),
			Headsign:      colGetter("trip_headsign"),
		}, nil
	}, func(trip GTFSTrip) error {
		count++
		return storage.PutJSON(db, "trip:", string(trip.GtfsId), trip)
	})

	if err == nil {
		fmt.Printf("Parsed %d trips\n", count)
	}
	return err
}

func parseCalendar(f *zip.File, db *pebble.DB) error {
	println("Parsing calendar")
	count := 0

	err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSService, error) {
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
	}, func(service GTFSService) error {
		count++
		return storage.PutJSON(db, "service:", string(service.GtfsId), service)
	})

	if err == nil {
		fmt.Printf("Parsed %d services\n", count)
	}
	return err
}

func parseCalendarDates(f *zip.File, db *pebble.DB) error {
	println("Parsing calendar dates")
	count := 0

	err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSServiceException, error) {
		exceptionType, err := strconv.Atoi(colGetter("exception_type"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse exception_type: %w", err)
		}

		return &GTFSServiceException{
			GtfsServiceId: GTFSServiceID(colGetter("service_id")),
			Date:          GTFSDate(colGetter("date")),
			ExceptionType: GTFSServiceExceptionType(exceptionType),
		}, nil
	}, func(exception GTFSServiceException) error {
		count++
		id := string(exception.GtfsServiceId) + ":" + string(exception.Date)
		return storage.PutJSON(db, "service_exception:", id, exception)
	})

	if err == nil {
		fmt.Printf("Parsed %d service exceptions\n", count)
	}
	return err
}

func parseStops(f *zip.File, db *pebble.DB) error {
	println("Parsing stops")
	count := 0

	err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSStop, error) {
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
	}, func(stop GTFSStop) error {
		count++
		return storage.PutJSON(db, "stop:", string(stop.GtfsId), stop)
	})

	if err == nil {
		fmt.Printf("Parsed %d stops\n", count)
	}
	return err
}

func parseStopTimes(f *zip.File, db *pebble.DB) error {
	println("Parsing stop times")
	count := 0

	err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSStopTime, error) {
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
	}, func(stopTime GTFSStopTime) error {
		count++
		id := string(stopTime.GtfsTripId) + ":" + fmt.Sprint(stopTime.StopSequence)
		return storage.PutJSON(db, "stop_time:", id, stopTime)
	})

	if err == nil {
		fmt.Printf("Parsed %d stop times\n", count)
	}
	return err
}

func parseTransfers(f *zip.File, db *pebble.DB) error {
	println("Parsing transfers")
	count := 0

	err := parseCSVFile(f, func(colGetter func(string) string) (*GTFSTransfer, error) {
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
	}, func(transfer GTFSTransfer) error {
		count++
		id := string(transfer.FromStopId) + ":" + string(transfer.ToStopId)
		return storage.PutJSON(db, "transfer:", id, transfer)
	})

	if err == nil {
		fmt.Printf("Parsed %d transfers\n", count)
	}
	return err
}
