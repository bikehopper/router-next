package transit

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
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

type RowParser[T any] func(colGetter func(string) string) (T, error)

func toUint(s string) uint32 {
	val, _ := strconv.ParseUint(s, 10, 32)
	return uint32(val)
}

func toFloat(s string) float64 {
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

func parseTimeAsInt(s string) uint32 {
	return 5
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
		results = append(results, parsedValue)
	}
	return results, nil
}

func ParseStops(f *zip.File) ([]GtfsStop, error) {
	return parseCSVFile(f, func(colGetter func(string) string) (GtfsStop, error) {
		return GtfsStop{
			GtfsID: colGetter("stop_id"),
			Name:   colGetter("stop_name"),
			Lat:    toFloat(colGetter("stop_lat")),
			Lon:    toFloat(colGetter("stop_lon")),
		}, nil
	})
}

func ParseStopTimes(f *zip.File) ([]GtfsStopTime, error) {
	return parseCSVFile(f, func(colGetter func(string) string) (GtfsStopTime, error) {
		return GtfsStopTime{
			GtfsStopID:    colGetter("stop_id"),
			GtfsTripID:    colGetter("trip_id"),
			ArrivalTime:   parseTimeAsInt(colGetter("arrival_time")),
			DepartureTime: parseTimeAsInt(colGetter("departure_time")),
			StopSequence:  toUint("stop_sequence"),
		}, nil
	})
}
