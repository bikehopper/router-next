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

func Reduce[T any, U any](slice []T, init U, f func(U, T) U) U {
	acc := init
	for _, v := range slice {
		acc = f(acc, v)
	}

	return acc
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
	Stops []Stop

	StopIdsByRoute     []types.StopID
	FirstStopIDOfRoute []types.StopID

	StopEventsByRoute     []StopEvent
	FirstStopEventOfRoute []uint32
	NumTripsInRoute       []uint32

	RouteSegmentsByStop     []RouteSegment
	FirstRouteSegmentOfStop []uint32
}

func (rt *RaptorTable) NumStops() int  { return len(rt.Stops) }
func (rt *RaptorTable) NumRoutes() int { return len(rt.NumTripsInRoute) }

func (rt *RaptorTable) StopsForRoute(route types.RouteID) []types.StopID {
	start := rt.FirstStopIDOfRoute[route]
	end := rt.FirstStopIDOfRoute[route+1]

	return rt.StopIdsByRoute[start:end]
}

func (rt *RaptorTable) NumStopsInRoute(route types.RouteID) uint32 {
	return uint32(len(rt.StopsForRoute(route)))
}

func (rt *RaptorTable) StopEventsForTrip(route types.RouteID, trip uint32) []StopEvent {
	numStops := rt.NumStopsInRoute(route)
	base := rt.FirstStopEventOfRoute[route]
	tripStart := base + numStops*trip

	return rt.StopEventsByRoute[tripStart : tripStart+numStops]
}

func (rt *RaptorTable) RoutesForStop(stop types.StopID) []RouteSegment {
	start := rt.FirstRouteSegmentOfStop[stop]
	end := rt.FirstRouteSegmentOfStop[stop+1]

	fmt.Printf("[%d:%d]\n", start, end)

	return rt.RouteSegmentsByStop[start:end]
}

func BuildRaptorTable(gtfsTable GtfsTable) (RaptorTable, error) {
	println("Building raptor table")

	stops := make([]Stop, len(gtfsTable.Stops))

	gtfsStopIdToIdx := make(map[GtfsStopID]types.StopID, len(gtfsTable.Stops))
	for idx, stop := range gtfsTable.Stops {
		gtfsStopIdToIdx[stop.GtfsID] = types.StopID(idx)
		stops[idx] = Stop{
			Name: stop.Name,
			Lat:  stop.Lat,
			Lon:  stop.Lon,
		}
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

	numRoutes := len(routes)
	numStops := len(gtfsTable.Stops)

	firstStopIdOfRoute := make([]types.StopID, numRoutes+1)
	firstStopEventOfRoute := make([]uint32, numRoutes+1)
	numTripsInRoute := make([]uint32, numRoutes)
	segmentCountByStop := make([]uint32, numStops)
	firstRouteSegmentOfStop := make([]uint32, numStops+1)

	var allStopIds []types.StopID

	var allStopEvents []StopEvent

	for routeId, route := range routes {
		firstStopIdOfRoute[routeId] = types.StopID(len(allStopIds))
		allStopIds = append(allStopIds, route.stopSequence...)

		firstStopEventOfRoute[routeId] = uint32(len(allStopEvents))
		numTripsInRoute[routeId] = uint32(len(route.trips))

		for _, trip := range route.trips {
			for _, stopTime := range trip {
				allStopEvents = append(allStopEvents, StopEvent{
					ArrivalTime:   stopTime.ArrivalTime,
					DepartureTime: stopTime.DepartureTime,
				})
			}
		}

		for _, stopId := range route.stopSequence {
			segmentCountByStop[stopId]++
		}
	}

	firstStopIdOfRoute[numRoutes] = types.StopID(len(allStopIds))
	firstStopEventOfRoute[numRoutes] = uint32(len(allStopEvents))

	for s := 0; s < numStops; s++ {
		firstRouteSegmentOfStop[s+1] = firstRouteSegmentOfStop[s] + segmentCountByStop[s]
	}

	totalSegments := firstRouteSegmentOfStop[numStops]
	allRouteSegments := make([]RouteSegment, totalSegments)

	cursor := make([]uint32, numStops)
	copy(cursor, firstRouteSegmentOfStop[:numStops])

	for routeIdx, route := range routes {
		for stopIdx, stopId := range route.stopSequence {
			allRouteSegments[cursor[stopId]] = RouteSegment{
				RouteId:   types.RouteID(routeIdx),
				StopIndex: types.StopIndex(stopIdx),
			}
			cursor[stopId]++
		}
	}

	return RaptorTable{
		Stops:                   stops,
		StopIdsByRoute:          allStopIds,
		FirstStopIDOfRoute:      firstStopIdOfRoute,
		StopEventsByRoute:       allStopEvents,
		FirstStopEventOfRoute:   firstStopEventOfRoute,
		NumTripsInRoute:         numTripsInRoute,
		RouteSegmentsByStop:     allRouteSegments,
		FirstRouteSegmentOfStop: firstRouteSegmentOfStop,
	}, nil
}
