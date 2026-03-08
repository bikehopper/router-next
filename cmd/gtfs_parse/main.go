package main

import (
	"fmt"
	"os"

	"router/pkg/transit"
)

func main() {
	gtfsTable, err := transit.ParseGtfs(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	fmt.Printf("detected %d stops\n", len(gtfsTable.Stops))
	fmt.Printf("detected %d trips\n", len(gtfsTable.Trips))
	fmt.Printf("detected %d stop_times\n", len(gtfsTable.StopTimes))

	for idx, stop := range gtfsTable.Stops[0:10] {
		fmt.Printf(
			"stop %d: id %s name %s coords (%f, %f)\n",
			idx,
			stop.GtfsID,
			stop.Name,
			stop.Lat,
			stop.Lon,
		)
	}

	for idx, trip := range gtfsTable.Trips[0:10] {
		fmt.Printf(
			"trip %d: id %s route %s service %s\n",
			idx,
			trip.GtfsID,
			trip.GtfsRouteID,
			trip.GtfsServiceID,
		)
	}

	for idx, stopTime := range gtfsTable.StopTimes[0:10] {
		fmt.Printf(
			"stop time %d: stop id %s trip id %s arrival %d departure %d seq %d \n",
			idx,
			stopTime.GtfsStopID,
			stopTime.GtfsTripID,
			stopTime.ArrivalTime,
			stopTime.DepartureTime,
			stopTime.StopSequence,
		)
	}
}
