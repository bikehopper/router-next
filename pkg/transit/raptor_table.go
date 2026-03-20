package transit

import (
	"fmt"
	"router/pkg/types"
	"sort"
	"strings"
)

func Map[T any, U any](slice []T, f func(T) U) []U {
	result := make([]U, len(slice))
	for i, item := range slice {
		result[i] = f(item)
	}

	return result
}

type Stop struct {
	Name     string
	Lat, Lon float64
}

type Route struct {
	Name string
}

type StopEvent struct {
	ArrivalTime   uint32
	DepartureTime uint32
}

type RouteSegment struct {
	RouteId   types.RouteID
	StopIndex types.StopIndex
}

type RaptorTable struct {
	StopIdsByRoute     []types.StopID
	FirstStopIDOfRoute []uint32

	StopEventsByRoute     []StopEvent
	FirstStopEventOfRoute []uint32
	NumTripsInRoute       []uint32

	RouteSegmentsByStop     []RouteSegment
	FirstRouteSegmentOfStop []uint32
}

func BuildRaptorTable(gtfsTable GtfsTable) (RaptorTable, error) {
	println("Building raptor table")

	gtfsStopIdToIdx := make(map[GtfsStopID]types.StopID, len(gtfsTable.Stops))
	for idx, stop := range gtfsTable.Stops {
		gtfsStopIdToIdx[stop.GtfsID] = types.StopID(idx)
	}

	activeGtfsTripIds := make(map[GtfsTripID]types.TripID, len(gtfsTable.Trips))
	for idx, trip := range gtfsTable.Trips {
		// TODO: filter by service day
		activeGtfsTripIds[trip.GtfsID] = types.TripID(idx)
	}

	stopTimesByTrip := make([][]GtfsStopTime, len(activeGtfsTripIds))

	for _, st := range gtfsTable.StopTimes {
		if tripID, ok := activeGtfsTripIds[st.GtfsTripID]; ok {
			stopTimesByTrip[tripID] = append(stopTimesByTrip[tripID], st)
		}
	}

	for _, stopTimes := range stopTimesByTrip {
		sort.Slice(stopTimes, func(i int, j int) bool {
			return stopTimes[i].StopSequence < stopTimes[j].StopSequence
		})
	}

	type routeEntry struct {
		stopSequence []types.StopID
		trips        [][]GtfsStopTime
	}

	stopSequenceKey := func(stopSequence []types.StopID) string {
		return strings.Join(Map(stopSequence, func(stopID types.StopID) string { return fmt.Sprintf("%d", stopID) }), ",")
	}

	routeMap := make(map[string]*routeEntry)

	for _, stopTimes := range stopTimesByTrip {
		stopSequence := make([]types.StopID, len(stopTimes))
		for i, stopTime := range stopTimes {
			stopID := gtfsStopIdToIdx[stopTime.GtfsStopID]
			stopSequence[i] = stopID
		}

		tripKey := stopSequenceKey(stopSequence)

		route, ok := routeMap[tripKey]
		if !ok {
			routeMap[tripKey] = &routeEntry{stopSequence, [][]GtfsStopTime{stopTimes}}
		} else {
			route.trips = append(route.trips, stopTimes)
		}
	}

	routes := make([]routeEntry, 0, len(routeMap))
	for _, route := range routeMap {
		routes = append(routes, *route)
	}

	for _, route := range routes {
		sort.Slice(route.trips, func(i, j int) bool {
			return route.trips[i][0].DepartureTime < route.trips[j][0].DepartureTime
		})
	}

	for routeID, route := range routes[0:5] {
		fmt.Printf("route %d\n", routeID)

		for tripNum, trip := range route.trips[0:min(len(route.trips), 2)] {
			fmt.Printf("trip %d\n", tripNum)

			for _, st := range trip {
				fmt.Printf("trip %s seq %d stop %s arr %d dep %d\n",
					st.GtfsTripID,
					st.StopSequence,
					st.GtfsStopID,
					st.ArrivalTime,
					st.DepartureTime,
				)
			}
		}
	}

	return RaptorTable{
		StopIdsByRoute:          nil,
		FirstStopIDOfRoute:      nil,
		StopEventsByRoute:       nil,
		FirstStopEventOfRoute:   nil,
		NumTripsInRoute:         nil,
		RouteSegmentsByStop:     nil,
		FirstRouteSegmentOfStop: nil,
	}, nil
}
