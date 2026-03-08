package transit

import (
	"archive/zip"
	"encoding/csv"
	"fmt"
	"io"
)

type GtfsStop struct {
	ID   string
	Name string
	Lat  string
	Lon  string
}

type RowParser[T any] func(colGetter func(string) string) (T, error)

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
		results = append(results, parsedValue)
	}
	return results, nil
}

func ParseStops(f *zip.File) ([]GtfsStop, error) {
	return parseCSVFile(f, func(colGetter func(string) string) (GtfsStop, error) {
		return GtfsStop{
			ID:   colGetter("stop_id"),
			Name: colGetter("stop_name"),
			Lat:  colGetter("stop_lat"),
			Lon:  colGetter("stop_lon"),
		}, nil
	})
}
