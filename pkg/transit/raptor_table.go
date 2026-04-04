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

type RaptorRoute struct {
	stopSequence []types.StopID
	trips        [][]GtfsStopTime
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
	Stops  []GtfsStop
	Routes []GtfsRoute

	MinTransferTime []uint32

	StopIdsByRoute     []types.StopID
	FirstStopIDOfRoute []uint32

	StopEventsByRoute     []StopEvent
	FirstStopEventOfRoute []uint32
	NumTripsInRoute       []uint32

	RouteSegmentsByStop     []RouteSegment
	FirstRouteSegmentOfStop []uint32

	TripsByRoute     []GtfsTrip
	FirstTripOfRoute []uint32
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

func (rt *RaptorTable) TripInRoute(route types.RouteID, trip uint32) GtfsTrip {
	base := rt.FirstTripOfRoute[route]

	return rt.TripsByRoute[base+trip]
}

func (rt *RaptorTable) RoutesForStop(stop types.StopID) []RouteSegment {
	start := rt.FirstRouteSegmentOfStop[stop]
	end := rt.FirstRouteSegmentOfStop[stop+1]

	return rt.RouteSegmentsByStop[start:end]
}

const StopOfInterest = 1093

func BuildRaptorTable(gtfsTable GtfsTable) (RaptorTable, error) {
	println("Building raptor table")

	gtfsStopIdToIdx := make(map[GtfsStopID]types.StopID, len(gtfsTable.Stops))
	for idx, stop := range gtfsTable.Stops {
		gtfsStopIdToIdx[stop.GtfsID] = types.StopID(idx)
	}

	gtfsRouteIdToIdx := make(map[GtfsRouteID]uint32, len(gtfsTable.Routes))
	for idx, route := range gtfsTable.Routes {
		gtfsRouteIdToIdx[route.GtfsID] = uint32(idx)
	}

	activeGtfsTripIds := make(map[GtfsTripID]uint32, len(gtfsTable.Trips))
	// TODO: filter by service day
	for idx, trip := range gtfsTable.Trips {
		activeGtfsTripIds[trip.GtfsID] = uint32(idx)
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

	stopSequenceKey := func(stopSequence []types.StopID) string {
		return strings.Join(Map(stopSequence, func(stopID types.StopID) string { return fmt.Sprintf("%d", stopID) }), ",")
	}

	routeMap := make(map[string]*RaptorRoute)

	for _, stopTimes := range stopTimesByTrip {
		if len(stopTimes) == 0 {
			continue
		}

		stopSequence := make([]types.StopID, len(stopTimes))
		for i, stopTime := range stopTimes {
			stopID, ok := gtfsStopIdToIdx[stopTime.GtfsStopID]
			if !ok {
				fmt.Printf("WARN: unknown stop_id %q\n", stopTime.GtfsStopID)
			}

			stopSequence[i] = stopID
		}

		tripKey := stopSequenceKey(stopSequence)

		route, ok := routeMap[tripKey]
		if !ok {
			routeMap[tripKey] = &RaptorRoute{stopSequence, [][]GtfsStopTime{stopTimes}}
		} else {
			route.trips = append(route.trips, stopTimes)
		}
	}

	routes := make([]RaptorRoute, 0, len(routeMap))
	for _, route := range routeMap {
		routes = append(routes, *route)
	}

	sort.Slice(routes, func(i, j int) bool {
		return stopSequenceKey(routes[i].stopSequence) < stopSequenceKey(routes[j].stopSequence)
	})

	for _, route := range routes {
		sort.Slice(route.trips, func(i, j int) bool {
			return route.trips[i][0].DepartureTime < route.trips[j][0].DepartureTime
		})
	}

	numRoutes := len(routes)
	numStops := len(gtfsTable.Stops)

	firstStopIdOfRoute := make([]uint32, numRoutes+1)
	firstStopEventOfRoute := make([]uint32, numRoutes+1)
	numTripsInRoute := make([]uint32, numRoutes)
	segmentCountByStop := make([]uint32, numStops)
	firstRouteSegmentOfStop := make([]uint32, numStops+1)
	firstTripOfRoute := make([]uint32, numRoutes+1)

	var allStopIds []types.StopID

	var allStopEvents []StopEvent

	var allTrips []GtfsTrip

	allRoutes := make([]GtfsRoute, len(routes))

	for routeId, route := range routes {
		firstTripId := route.trips[0][0].GtfsTripID
		firstTripIdx := activeGtfsTripIds[firstTripId]
		gtfsRouteID := gtfsTable.Trips[firstTripIdx].GtfsRouteID
		gtfsRouteIdx := gtfsRouteIdToIdx[gtfsRouteID]
		gtfsRoute := gtfsTable.Routes[gtfsRouteIdx]
		allRoutes[routeId] = gtfsRoute

		firstStopIdOfRoute[routeId] = uint32(len(allStopIds))
		allStopIds = append(allStopIds, route.stopSequence...)

		firstStopEventOfRoute[routeId] = uint32(len(allStopEvents))
		numTripsInRoute[routeId] = uint32(len(route.trips))

		firstTripOfRoute[routeId] = uint32(len(allTrips))

		for _, trip := range route.trips {
			gtfsTripId := trip[0].GtfsTripID
			tripIdx := activeGtfsTripIds[gtfsTripId]
			gtfsTrip := gtfsTable.Trips[tripIdx]

			allTrips = append(allTrips, gtfsTrip)

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

	firstStopIdOfRoute[numRoutes] = uint32(len(allStopIds))
	firstStopEventOfRoute[numRoutes] = uint32(len(allStopEvents))
	firstTripOfRoute[numRoutes] = uint32(len(allTrips))

	for s := 0; s < numStops; s++ {
		firstRouteSegmentOfStop[s+1] = firstRouteSegmentOfStop[s] + segmentCountByStop[s]
	}

	totalSegments := firstRouteSegmentOfStop[numStops]
	allRouteSegments := make([]RouteSegment, totalSegments)

	cursor := make([]uint32, numStops)
	copy(cursor, firstRouteSegmentOfStop[:numStops])

	for routeId, route := range routes {
		for stopIdx, stopId := range route.stopSequence {
			allRouteSegments[cursor[stopId]] = RouteSegment{
				RouteId:   types.RouteID(routeId),
				StopIndex: types.StopIndex(stopIdx),
			}
			cursor[stopId]++
		}
	}

	minTransferTime := make([]uint32, numStops)

	for _, transfer := range gtfsTable.Transfers {
		if transfer.TransferType != 2 {
			continue
		}

		if transfer.FromStopId != transfer.ToStopId {
			continue
		}

		stopId := gtfsStopIdToIdx[transfer.FromStopId]
		minTransferTime[stopId] = transfer.MinTransferTime
	}

	return RaptorTable{
		Stops:                   gtfsTable.Stops,
		Routes:                  allRoutes,
		MinTransferTime:         minTransferTime,
		StopIdsByRoute:          allStopIds,
		FirstStopIDOfRoute:      firstStopIdOfRoute,
		StopEventsByRoute:       allStopEvents,
		FirstStopEventOfRoute:   firstStopEventOfRoute,
		NumTripsInRoute:         numTripsInRoute,
		RouteSegmentsByStop:     allRouteSegments,
		FirstRouteSegmentOfStop: firstRouteSegmentOfStop,
		TripsByRoute:            allTrips,
		FirstTripOfRoute:        firstTripOfRoute,
	}, nil
}
