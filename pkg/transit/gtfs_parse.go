package transit

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type GtfsStop struct {
	GtfsID string
	Name   string
	Lat    float64
	Lon    float64
}

type GtfsStopTime struct {
	GtfsTripID    string
	GtfsStopID    string
	ArrivalTime   uint32
	DepartureTime uint32
	StopSequence  uint32
}

type GtfsTrip struct {
	GtfsID        string
	GtfsRouteID   string
	GtfsServiceID string
}

type RowParser[T any] func(colGetter func(string) string) (*T, error)

func toUint(s string) (uint32, error) {
	val, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid uint: %w", err)
	}

	return uint32(val), nil
}

func toFloat(s string) (float64, error) {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float: %w", err)
	}

	return val, nil
}

func gtfsTimeToSeconds(gtfsTime string) (uint32, error) {
	parts := strings.Split(gtfsTime, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid GTFS time format: %s", gtfsTime)
	}

	hours, err := toUint(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hours in GTFS time: %w", err)
	}

	minutes, err := toUint(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid minutes in GTFS time: %w", err)
	}

	seconds, err := toUint(parts[2])
	if err != nil {
		return 0, fmt.Errorf("invalid seconds in GTFS time: %w", err)
	}

	// Calculate total seconds, allowing hours > 24
	totalSeconds := hours*3600 + minutes*60 + seconds

	return totalSeconds, nil
}

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
			return nil, fmt.Errorf("failed to parse row: %w", err)
		}

		results = append(results, *parsedValue)
	}

	return results, nil
}

func ParseStops(f *zip.File) ([]GtfsStop, error) {
	return parseCSVFile(f, func(colGetter func(string) string) (*GtfsStop, error) {
		lat, err := toFloat(colGetter("stop_lat"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse stop_lat: %w", err)
		}

		lon, err := toFloat(colGetter("stop_lon"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse stop_lon: %w", err)
		}

		return &GtfsStop{
			GtfsID: colGetter("stop_id"),
			Name:   colGetter("stop_name"),
			Lat:    lat,
			Lon:    lon,
		}, nil
	})
}

func ParseStopTimes(f *zip.File) ([]GtfsStopTime, error) {
	return parseCSVFile(f, func(colGetter func(string) string) (*GtfsStopTime, error) {
		arrivalTime, err := gtfsTimeToSeconds(colGetter("arrival_time"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse arrival_time: %w", err)
		}

		departure_time, err := gtfsTimeToSeconds(colGetter("departure_time"))
		if err != nil {
			return nil, fmt.Errorf("failed to parse departure_time: %w", err)
		}

		stopSequence, err := toUint("stop_sequence")
		if err != nil {
			return nil, fmt.Errorf("failed to parse stop_sequence: %w", err)
		}

		return &GtfsStopTime{
			GtfsStopID:    colGetter("stop_id"),
			GtfsTripID:    colGetter("trip_id"),
			ArrivalTime:   arrivalTime,
			DepartureTime: departure_time,
			StopSequence:  stopSequence,
		}, nil
	})
}
