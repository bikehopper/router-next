package transit

import (
	"fmt"
	"sort"
	"strings"

	"router/pkg/types"
	"router/pkg/utils"
)

type RaptorTrip []GtfsStopTime

type RaptorRoute struct {
	stopSequence []types.StopID
	trips        []RaptorTrip
}

type StopEvent struct {
	ArrivalTime   uint32
	DepartureTime uint32
}

type RouteSegment struct {
	RouteId   types.RouteID
	StopIndex uint32
}

type RouteStopOffsets []uint32        // indexed by types.RouteID
type RouteStopEventOffsets []uint32   // indexed by types.RouteID
type RouteTripOffsets []uint32        // indexed by types.RouteID
type StopRouteSegmentOffsets []uint32 // indexed by types.StopID
type StopTransferTimes []uint32       // indexed by types.StopID

type RaptorTable struct {
	Stops  []GtfsStop
	Routes []GtfsRoute

	MinTransferTime StopTransferTimes

	StopIdsByRoute     []types.StopID
	FirstStopIDOfRoute RouteStopOffsets

	TripsByRoute     []GtfsTrip
	FirstTripOfRoute RouteTripOffsets
	NumTripsInRoute  []uint32

	StopEventsByRoute     []StopEvent
	FirstStopEventOfRoute RouteStopEventOffsets

	RouteSegmentsByStop     []RouteSegment
	FirstRouteSegmentOfStop StopRouteSegmentOffsets
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
	routeStart := rt.FirstStopEventOfRoute[route]
	tripStart := routeStart + numStops*trip

	return rt.StopEventsByRoute[tripStart : tripStart+numStops]
}

func (rt *RaptorTable) TripInRoute(route types.RouteID, trip uint32) GtfsTrip {
	routeStart := rt.FirstTripOfRoute[route]

	return rt.TripsByRoute[routeStart+trip]
}

func (rt *RaptorTable) RoutesForStop(stop types.StopID) []RouteSegment {
	start := rt.FirstRouteSegmentOfStop[stop]
	end := rt.FirstRouteSegmentOfStop[stop+1]

	return rt.RouteSegmentsByStop[start:end]
}

// all raptor routes have a parent GTFS route we get name/number from.
func buildGtfsRouteMap(gtfsRoutes []GtfsRoute) map[GtfsRouteID]*GtfsRoute {
	gtfsRouteMap := make(map[GtfsRouteID]*GtfsRoute, len(gtfsRoutes))
	for _, route := range gtfsRoutes {
		gtfsRouteMap[route.GtfsID] = &route
	}

	return gtfsRouteMap
}

func enumerateGtfsStops(gtfsStops []GtfsStop) map[GtfsStopID]types.StopID {
	gtfsStopIdMap := make(map[GtfsStopID]types.StopID, len(gtfsStops))
	for idx, stop := range gtfsStops {
		gtfsStopIdMap[stop.GtfsID] = types.StopID(idx)
	}

	return gtfsStopIdMap
}

func enumerateGtfsTrips(gtfsTrips []GtfsTrip) map[GtfsTripID]uint32 {
	gtfsActiveTripIdMap := make(map[GtfsTripID]uint32, len(gtfsTrips))
	// TODO: filter by service day
	for idx, trip := range gtfsTrips {
		gtfsActiveTripIdMap[trip.GtfsID] = uint32(idx)
	}

	return gtfsActiveTripIdMap
}

func extractSelfTransfers(gtfsTranfers []GtfsTransfer, stopIdMap map[GtfsStopID]types.StopID) StopTransferTimes {
	minTransferTime := make(StopTransferTimes, len(stopIdMap))

	for _, transfer := range gtfsTranfers {
		if transfer.TransferType != 2 {
			continue
		}

		if transfer.FromStopId != transfer.ToStopId {
			continue
		}

		stopId := stopIdMap[transfer.FromStopId]
		minTransferTime[stopId] = transfer.MinTransferTime
	}

	return minTransferTime
}

func groupRaptorTrips(gtfsStopTimes []GtfsStopTime, tripIdMap map[GtfsTripID]uint32) []RaptorTrip {
	raptorTrips := make([]RaptorTrip, len(tripIdMap))

	for _, stopTime := range gtfsStopTimes {
		if tripID, ok := tripIdMap[stopTime.GtfsTripID]; ok {
			raptorTrips[tripID] = append(raptorTrips[tripID], stopTime)
		}
	}

	for _, trip := range raptorTrips {
		sort.Slice(trip, func(i int, j int) bool {
			return trip[i].StopSequence < trip[j].StopSequence
		})
	}

	return raptorTrips
}

func getStopSequence(trip RaptorTrip, stopIdMap map[GtfsStopID]types.StopID) []types.StopID {
	stopSequence := make([]types.StopID, len(trip))
	for i, stopTime := range trip {
		stopID, ok := stopIdMap[stopTime.GtfsStopID]
		if !ok {
			fmt.Printf("WARN: unknown stop_id %q\n", stopTime.GtfsStopID)
		}

		stopSequence[i] = stopID
	}

	return stopSequence
}

func buildStopSequenceKey(stopSequence []types.StopID) string {
	return strings.Join(
		utils.Map(stopSequence, func(stopID types.StopID) string { return fmt.Sprintf("%d", stopID) }),
		",",
	)
}

func groupTripsByStopSequence(raptorTrips []RaptorTrip, stopIdMap map[GtfsStopID]types.StopID) map[string]*RaptorRoute {
	routeMap := make(map[string]*RaptorRoute)

	for _, trip := range raptorTrips {
		if len(trip) == 0 {
			continue
		}

		stopSequence := getStopSequence(trip, stopIdMap)
		tripKey := buildStopSequenceKey(stopSequence)

		route, ok := routeMap[tripKey]
		if !ok {
			routeMap[tripKey] = &RaptorRoute{stopSequence, []RaptorTrip{trip}}
		} else {
			route.trips = append(route.trips, trip)
		}
	}

	return routeMap
}

func sortRoutes(routeMap map[string]*RaptorRoute) []RaptorRoute {
	routes := make([]RaptorRoute, 0, len(routeMap))
	for _, route := range routeMap {
		routes = append(routes, *route)
	}

	sort.Slice(routes, func(i, j int) bool {
		return buildStopSequenceKey(routes[i].stopSequence) < buildStopSequenceKey(routes[j].stopSequence)
	})

	for _, route := range routes {
		sort.Slice(route.trips, func(i, j int) bool {
			return route.trips[i][0].DepartureTime < route.trips[j][0].DepartureTime
		})
	}

	return routes
}

func groupRaptorRoutes(raptorTrips []RaptorTrip, stopIdMap map[GtfsStopID]types.StopID) []RaptorRoute {
	return sortRoutes(groupTripsByStopSequence(raptorTrips, stopIdMap))
}

func iterateOverRaptorRoutes(
	raptorRoutes []RaptorRoute,
	routeMap map[GtfsRouteID]*GtfsRoute,
	getTripFromGtfsId func(gtfsTripID GtfsTripID) GtfsTrip,
	numStops int,
) ([]GtfsRoute, []types.StopID, []GtfsTrip, []StopEvent, RouteStopOffsets,
	RouteTripOffsets, []uint32, RouteStopEventOffsets, []uint32) {
	numRoutes := len(raptorRoutes)

	allRoutes := make([]GtfsRoute, numRoutes)
	firstStopIdOfRoute := make(RouteStopOffsets, numRoutes+1)
	firstTripOfRoute := make(RouteTripOffsets, numRoutes+1)
	numTripsInRoute := make([]uint32, numRoutes)
	firstStopEventOfRoute := make(RouteStopEventOffsets, numRoutes+1)
	numRoutesForStop := make([]uint32, numStops)

	var allStops []types.StopID

	var allStopEvents []StopEvent

	var tripsByRoute []GtfsTrip

	for routeId, route := range raptorRoutes {
		firstTripId := route.trips[0][0].GtfsTripID
		firstTrip := getTripFromGtfsId(firstTripId)
		gtfsRoute := *routeMap[firstTrip.GtfsRouteID]
		allRoutes[routeId] = gtfsRoute

		firstStopIdOfRoute[routeId] = uint32(len(allStops))
		allStops = append(allStops, route.stopSequence...)

		firstTripOfRoute[routeId] = uint32(len(tripsByRoute))
		numTripsInRoute[routeId] = uint32(len(route.trips))

		firstStopEventOfRoute[routeId] = uint32(len(allStopEvents))

		for _, trip := range route.trips {
			gtfsTripId := trip[0].GtfsTripID
			gtfsTrip := getTripFromGtfsId(gtfsTripId)

			tripsByRoute = append(tripsByRoute, gtfsTrip)

			for _, stopTime := range trip {
				allStopEvents = append(allStopEvents, StopEvent{
					ArrivalTime:   stopTime.ArrivalTime,
					DepartureTime: stopTime.DepartureTime,
				})
			}
		}

		for _, stopId := range route.stopSequence {
			numRoutesForStop[stopId]++
		}
	}

	firstStopIdOfRoute[numRoutes] = uint32(len(allStops))
	firstTripOfRoute[numRoutes] = uint32(len(tripsByRoute))
	firstStopEventOfRoute[numRoutes] = uint32(len(allStopEvents))

	return allRoutes, allStops, tripsByRoute, allStopEvents, firstStopIdOfRoute,
		firstTripOfRoute, numTripsInRoute, firstStopEventOfRoute, numRoutesForStop
}

func groupRouteSegments(
	raptorRoutes []RaptorRoute,
	numRoutesForStop []uint32,
	numStops int,
) ([]RouteSegment, StopRouteSegmentOffsets) {
	firstRouteSegmentOfStop := make(StopRouteSegmentOffsets, numStops+1)

	for s := range numStops {
		firstRouteSegmentOfStop[s+1] = firstRouteSegmentOfStop[s] + numRoutesForStop[s]
	}

	totalSegments := firstRouteSegmentOfStop[numStops]
	allRouteSegments := make([]RouteSegment, totalSegments)

	cursor := make(StopRouteSegmentOffsets, numStops)
	copy(cursor, firstRouteSegmentOfStop[:numStops])

	for routeId, route := range raptorRoutes {
		for stopIdx, stopId := range route.stopSequence {
			allRouteSegments[cursor[stopId]] = RouteSegment{
				RouteId:   types.RouteID(routeId),
				StopIndex: uint32(stopIdx),
			}
			cursor[stopId]++
		}
	}

	return allRouteSegments, firstRouteSegmentOfStop
}

func BuildRaptorTable(gtfsTable GtfsTable) (RaptorTable, error) {
	println("Building raptor table")

	numStops := len(gtfsTable.Stops)
	gtfsRouteMap := buildGtfsRouteMap(gtfsTable.Routes)
	gtfsStopIdMap := enumerateGtfsStops(gtfsTable.Stops)
	gtfsActiveTripIdMap := enumerateGtfsTrips(gtfsTable.Trips)
	selfTransfers := extractSelfTransfers(gtfsTable.Transfers, gtfsStopIdMap)

	getTripFromGtfsId := func(gtfsTripID GtfsTripID) GtfsTrip {
		return gtfsTable.Trips[gtfsActiveTripIdMap[gtfsTripID]]
	}

	raptorTrips := groupRaptorTrips(gtfsTable.StopTimes, gtfsActiveTripIdMap)
	raptorRoutes := groupRaptorRoutes(raptorTrips, gtfsStopIdMap)

	allRoutes, allStops, tripsByRoute, allStopEvents, firstStopIdOfRoute,
		firstTripOfRoute, numTripsInRoute, firstStopEventOfRoute, numRoutesForStop :=
		iterateOverRaptorRoutes(raptorRoutes, gtfsRouteMap, getTripFromGtfsId, numStops)

	allRouteSegments, firstRouteSegmentOfStop := groupRouteSegments(raptorRoutes, numRoutesForStop, numStops)

	return RaptorTable{
		Stops:                   gtfsTable.Stops,
		Routes:                  allRoutes,
		MinTransferTime:         selfTransfers,
		StopIdsByRoute:          allStops,
		FirstStopIDOfRoute:      firstStopIdOfRoute,
		StopEventsByRoute:       allStopEvents,
		FirstStopEventOfRoute:   firstStopEventOfRoute,
		NumTripsInRoute:         numTripsInRoute,
		RouteSegmentsByStop:     allRouteSegments,
		FirstRouteSegmentOfStop: firstRouteSegmentOfStop,
		TripsByRoute:            tripsByRoute,
		FirstTripOfRoute:        firstTripOfRoute,
	}, nil
}
