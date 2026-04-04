package main

import (
	"fmt"
	"os"

	"router/pkg/transit"
	"router/pkg/types"
)

func main() {
	gtfsTable, err := transit.ParseGtfs(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	rt, err := transit.BuildRaptorTable(*gtfsTable)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	routeSegments := rt.RoutesForStop(types.StopID(transit.StopOfInterest))
	stop := rt.Stops[transit.StopOfInterest]
	fmt.Printf("stop %s (%f, %f)\n", stop.Name, stop.Lat, stop.Lon)

	for _, rs := range routeSegments {
		fmt.Printf("route %d: stop %d\n", rs.RouteId, rs.StopIndex)
	}

	routeOfInterest := types.RouteID(324)

	numTrips := rt.NumTripsInRoute[routeOfInterest]
	tm := rt.TripInRoute(routeOfInterest, numTrips-1)
	fmt.Printf("route: %d, trip: %d, gtfs id: %s, headsign: %s\n", routeOfInterest, numTrips-1, tm.GtfsID, tm.Headsign)

	route := rt.Routes[routeOfInterest]
	fmt.Printf("route: %d, short name: %q, long name: %q, type: %d, color %s", routeOfInterest,
		route.ShortName, route.LongName, route.RouteType, route.Color)
}
