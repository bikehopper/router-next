package transit

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"strconv"
	"strings"

	"router/pkg/types"
	"router/pkg/utils"
)

type RaptorTripID uint32 // Temporary ID during processing. In the final RAPTOR table, trips are only stored per-route.

type RaptorStopTime struct {
	tripId        RaptorTripID
	stopId        types.StopID
	arrivalTime   types.Timestamp
	departureTime types.Timestamp
	stopSequence  uint32
}

type RaptorTrip []RaptorStopTime

type RaptorRoute struct {
	stopSequence []types.StopID
	trips        []RaptorTrip
}

type StopEvent struct {
	ArrivalTime   types.Timestamp
	DepartureTime types.Timestamp
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
	Stops  []GTFSStop
	Routes []GTFSRoute

	MinTransferTime StopTransferTimes

	StopIdsByRoute     []types.StopID
	FirstStopIdOfRoute RouteStopOffsets

	TripsByRoute     []GTFSTrip
	FirstTripOfRoute RouteTripOffsets
	NumTripsInRoute  []uint32 // indexed by types.RouteID

	StopEventsByRoute     []StopEvent
	FirstStopEventOfRoute RouteStopEventOffsets

	RouteSegmentsByStop     []RouteSegment
	FirstRouteSegmentOfStop StopRouteSegmentOffsets
}

func (rt *RaptorTable) NumStops() int  { return len(rt.Stops) }
func (rt *RaptorTable) NumRoutes() int { return len(rt.NumTripsInRoute) }

func (rt *RaptorTable) StopsForRoute(route types.RouteID) []types.StopID {
	start := rt.FirstStopIdOfRoute[route]
	end := rt.FirstStopIdOfRoute[route+1]

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

func (rt *RaptorTable) TripInRoute(route types.RouteID, trip uint32) GTFSTrip {
	routeStart := rt.FirstTripOfRoute[route]

	return rt.TripsByRoute[routeStart+trip]
}

func (rt *RaptorTable) RoutesForStop(stop types.StopID) []RouteSegment {
	start := rt.FirstRouteSegmentOfStop[stop]
	end := rt.FirstRouteSegmentOfStop[stop+1]

	return rt.RouteSegmentsByStop[start:end]
}

func buildGtfsRouteMap(gtfsRoutes []GTFSRoute) map[GTFSRouteID]*GTFSRoute {
	gtfsRouteMap := make(map[GTFSRouteID]*GTFSRoute, len(gtfsRoutes))
	for i := range gtfsRoutes {
		gtfsRouteMap[gtfsRoutes[i].GtfsId] = &gtfsRoutes[i]
	}

	return gtfsRouteMap
}

func enumerateGtfsStops(gtfsStops []GTFSStop) map[GTFSStopID]types.StopID {
	gtfsStopIdMap := make(map[GTFSStopID]types.StopID, len(gtfsStops))
	for idx, stop := range gtfsStops {
		gtfsStopIdMap[stop.GtfsId] = types.StopID(idx)
	}

	return gtfsStopIdMap
}

func enumerateGtfsTrips(gtfsTrips []GTFSTrip) map[GTFSTripID]RaptorTripID {
	gtfsActiveTripIdMap := make(map[GTFSTripID]RaptorTripID, len(gtfsTrips))
	// TODO: filter by service day
	for idx, trip := range gtfsTrips {
		gtfsActiveTripIdMap[trip.GtfsId] = RaptorTripID(idx)
	}

	return gtfsActiveTripIdMap
}

func extractSelfTransfers(gtfsTranfers []GTFSTransfer, stopIdMap map[GTFSStopID]types.StopID) StopTransferTimes {
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

func groupRaptorTrips(
	gtfsStopTimes []GTFSStopTime,
	stopIdMap map[GTFSStopID]types.StopID,
	tripIdMap map[GTFSTripID]RaptorTripID,
) []RaptorTrip {
	raptorTrips := make([]RaptorTrip, len(tripIdMap))

	for _, stopTime := range gtfsStopTimes {
		if tripId, ok := tripIdMap[stopTime.GtfsTripId]; ok {
			stopId, ok := stopIdMap[stopTime.GtfsStopId]
			if !ok {
				fmt.Printf("WARN: unknown stop_id %q\n", stopTime.GtfsStopId)
			}

			raptorTrips[tripId] = append(raptorTrips[tripId], RaptorStopTime{
				tripId:        tripId,
				stopId:        stopId,
				arrivalTime:   stopTime.ArrivalTime,
				departureTime: stopTime.DepartureTime,
				stopSequence:  stopTime.StopSequence,
			})
		}
	}

	for _, trip := range raptorTrips {
		slices.SortFunc(trip, func(stopTimeA, stopTimeB RaptorStopTime) int {
			return cmp.Compare(stopTimeA.stopSequence, stopTimeB.stopSequence)
		})
	}

	return raptorTrips
}

func getStopSequence(trip RaptorTrip) []types.StopID {
	return utils.Map(trip, func(stopTime RaptorStopTime) types.StopID {
		return stopTime.stopId
	})
}

func buildStopSequenceKey(stopSequence []types.StopID) string {
	return strings.Join(
		utils.Map(stopSequence, func(stopId types.StopID) string { return strconv.FormatUint(uint64(stopId), 10) }),
		",",
	)
}

func groupTripsByStopSequence(raptorTrips []RaptorTrip) map[string]*RaptorRoute {
	routeMap := make(map[string]*RaptorRoute)

	for _, trip := range raptorTrips {
		if len(trip) == 0 {
			continue
		}

		stopSequence := getStopSequence(trip)
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
	sortedSequenceKeys := slices.Sorted(maps.Keys(routeMap))

	routes := make([]RaptorRoute, len(routeMap))
	for i, key := range sortedSequenceKeys {
		routes[i] = *routeMap[key]
	}

	for _, route := range routes {
		slices.SortFunc(route.trips, func(tripA, tripB RaptorTrip) int {
			return cmp.Compare(tripA[0].departureTime, tripB[0].departureTime)
		})
	}

	return routes
}

func groupRaptorRoutes(raptorTrips []RaptorTrip) []RaptorRoute {
	return sortRoutes(groupTripsByStopSequence(raptorTrips))
}

func iterateOverRaptorRoutes(
	raptorRoutes []RaptorRoute,
	routeMap map[GTFSRouteID]*GTFSRoute,
	gtfsTrips []GTFSTrip,
	numStops int,
) ([]GTFSRoute, []types.StopID, []GTFSTrip, []StopEvent, RouteStopOffsets,
	RouteTripOffsets, []uint32, RouteStopEventOffsets, []uint32) {
	numRoutes := len(raptorRoutes)

	routes := make([]GTFSRoute, numRoutes)
	firstStopIdOfRoute := make(RouteStopOffsets, numRoutes+1)
	firstTripOfRoute := make(RouteTripOffsets, numRoutes+1)
	numTripsInRoute := make([]uint32, numRoutes)
	firstStopEventOfRoute := make(RouteStopEventOffsets, numRoutes+1)
	numRoutesForStop := make([]uint32, numStops)

	var stopIdsByRoute []types.StopID

	var stopEventsByRoute []StopEvent

	var tripsByRoute []GTFSTrip

	for routeId, route := range raptorRoutes {
		firstTripId := route.trips[0][0].tripId
		firstTrip := gtfsTrips[firstTripId]
		gtfsRoute := *routeMap[firstTrip.GtfsRouteId]
		routes[routeId] = gtfsRoute

		firstStopIdOfRoute[routeId] = uint32(len(stopIdsByRoute))
		stopIdsByRoute = append(stopIdsByRoute, route.stopSequence...)

		firstTripOfRoute[routeId] = uint32(len(tripsByRoute))
		numTripsInRoute[routeId] = uint32(len(route.trips))

		firstStopEventOfRoute[routeId] = uint32(len(stopEventsByRoute))

		for _, trip := range route.trips {
			gtfsTrip := gtfsTrips[trip[0].tripId]

			tripsByRoute = append(tripsByRoute, gtfsTrip)

			for _, stopTime := range trip {
				stopEventsByRoute = append(stopEventsByRoute, StopEvent{
					ArrivalTime:   stopTime.arrivalTime,
					DepartureTime: stopTime.departureTime,
				})
			}
		}

		for _, stopId := range route.stopSequence {
			numRoutesForStop[stopId]++
		}
	}

	firstStopIdOfRoute[numRoutes] = uint32(len(stopIdsByRoute))
	firstTripOfRoute[numRoutes] = uint32(len(tripsByRoute))
	firstStopEventOfRoute[numRoutes] = uint32(len(stopEventsByRoute))

	return routes, stopIdsByRoute, tripsByRoute, stopEventsByRoute, firstStopIdOfRoute,
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
	routeSegmentsByStop := make([]RouteSegment, totalSegments)

	cursor := make(StopRouteSegmentOffsets, numStops)
	copy(cursor, firstRouteSegmentOfStop[:numStops])

	for routeId, route := range raptorRoutes {
		for stopIdx, stopId := range route.stopSequence {
			routeSegmentsByStop[cursor[stopId]] = RouteSegment{
				RouteId:   types.RouteID(routeId),
				StopIndex: uint32(stopIdx),
			}
			cursor[stopId]++
		}
	}

	return routeSegmentsByStop, firstRouteSegmentOfStop
}

func BuildRaptorTable(gtfsTable GTFSTable) (RaptorTable, error) {
	println("Building raptor table")

	gtfsRouteMap := buildGtfsRouteMap(gtfsTable.Routes)
	gtfsStopIdMap := enumerateGtfsStops(gtfsTable.Stops)
	numStops := len(gtfsTable.Stops)
	gtfsActiveTripIdMap := enumerateGtfsTrips(gtfsTable.Trips)
	selfTransfers := extractSelfTransfers(gtfsTable.Transfers, gtfsStopIdMap)

	raptorTrips := groupRaptorTrips(gtfsTable.StopTimes, gtfsStopIdMap, gtfsActiveTripIdMap)
	raptorRoutes := groupRaptorRoutes(raptorTrips)

	routes, stopIdsByRoute, tripsByRoute, stopEventsByRoute, firstStopIdOfRoute,
		firstTripOfRoute, numTripsInRoute, firstStopEventOfRoute, numRoutesForStop :=
		iterateOverRaptorRoutes(raptorRoutes, gtfsRouteMap, gtfsTable.Trips, numStops)

	routeSegmentsByStop, firstRouteSegmentOfStop := groupRouteSegments(raptorRoutes, numRoutesForStop, numStops)

	return RaptorTable{
		Stops:                   gtfsTable.Stops,
		Routes:                  routes,
		MinTransferTime:         selfTransfers,
		StopIdsByRoute:          stopIdsByRoute,
		FirstStopIdOfRoute:      firstStopIdOfRoute,
		StopEventsByRoute:       stopEventsByRoute,
		FirstStopEventOfRoute:   firstStopEventOfRoute,
		NumTripsInRoute:         numTripsInRoute,
		RouteSegmentsByStop:     routeSegmentsByStop,
		FirstRouteSegmentOfStop: firstRouteSegmentOfStop,
		TripsByRoute:            tripsByRoute,
		FirstTripOfRoute:        firstTripOfRoute,
	}, nil
}
