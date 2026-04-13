package gtfs

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func parseUint32(s string) (uint32, error) {
	val, err := strconv.ParseUint(s, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid uint %s: %w", s, err)
	}

	return uint32(val), nil
}

func parseFloat64(s string) (float64, error) {
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid float: %w", err)
	}

	return val, nil
}

func parseBool(s string) bool {
	return s == "1"
}

func parseTime(date GTFSDate) (time.Time, error) {
	s := string(date)
	year, err1 := strconv.Atoi(s[:4])
	month, err2 := strconv.Atoi(s[4:6])
	day, err3 := strconv.Atoi(s[6:])

	if err1 != nil || err2 != nil || err3 != nil {
		return time.Time{}, fmt.Errorf("invalid date: %s", s)
	}

	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC), nil
}

func TimeToGTFSDate(time time.Time) GTFSDate {
	return GTFSDate(fmt.Sprintf("%04d%02d%02d", time.Year(), time.Month(), time.Day()))
}

func gtfsTimeToSeconds(gtfsTime string) (uint32, error) {
	parts := strings.Split(gtfsTime, ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("invalid GTFS time format: %s", gtfsTime)
	}

	hours, err := parseUint32(parts[0])
	if err != nil {
		return 0, fmt.Errorf("invalid hours in GTFS time: %w", err)
	}

	minutes, err := parseUint32(parts[1])
	if err != nil {
		return 0, fmt.Errorf("invalid minutes in GTFS time: %w", err)
	}

	seconds, err := parseUint32(parts[2])
	if err != nil {
		return 0, fmt.Errorf("invalid seconds in GTFS time: %w", err)
	}

	// Calculate total seconds, allowing hours > 24
	totalSeconds := hours*3600 + minutes*60 + seconds

	return totalSeconds, nil
}
