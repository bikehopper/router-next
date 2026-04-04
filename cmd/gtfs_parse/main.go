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

	routeSegments := rt.RoutesForStop(types.StopID(1))
	stop := rt.Stops[10]
	fmt.Printf("stop %s (%f, %f)\n", stop.Name, stop.Lat, stop.Lon)

	for _, rs := range routeSegments {
		fmt.Printf("route %d: stop %d\n", rs.RouteId, rs.StopIndex)
	}
}
