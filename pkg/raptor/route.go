package raptor

import (
	"router/pkg/types"
	"slices"
)

type Label struct {
	RouteId       types.RouteID
	TripIdx       uint32
	BoardStopId   types.StopID
	BoardStopIdx  StopIndex
	AlightStopIdx StopIndex
}

const MAX_ROUNDS = 10

func (rt *RaptorTable) Route(start types.StopID, end types.StopID, startTime types.Timestamp) []Label {
	parent := make([]Label, rt.NumStops())

	best := make([]types.Timestamp, rt.NumStops())
	for i := range best {
		best[i] = types.INFINITY
	}

	best[start] = startTime

	var prevBest []types.Timestamp

	stopsUpdated := []types.StopID{start}

	for len(stopsUpdated) > 0 {
		prevBest = make([]types.Timestamp, rt.NumStops())
		copy(prevBest, best)

		routeEarliestStop := make(map[types.RouteID]StopIndex)

		for _, stopId := range stopsUpdated {
			for _, segment := range rt.RoutesForStop(stopId) {
				if existing, ok := routeEarliestStop[segment.RouteId]; !ok || segment.StopIndex < existing {
					routeEarliestStop[segment.RouteId] = segment.StopIndex
				}
			}
		}

		var nextStopsUpdated []types.StopID

		for routeId, firstStopIdx := range routeEarliestStop {
			stopsForRoute := rt.StopsForRoute(routeId)

			currentTripIdx := -1
			boardStopId := types.StopID(0)
			boardStopIdx := uint32(0)

			for currStopIdx := int(firstStopIdx); currStopIdx < len(stopsForRoute); currStopIdx++ {
				currStopId := stopsForRoute[currStopIdx]

				if currentTripIdx >= 0 {
					events := rt.StopEventsForTrip(routeId, uint32(currentTripIdx))
					if events[currStopIdx].ArrivalTime < best[currStopId] {
						best[currStopId] = events[currStopIdx].ArrivalTime
						parent[currStopId] = Label{
							RouteId:       routeId,
							TripIdx:       uint32(currentTripIdx),
							BoardStopId:   boardStopId,
							BoardStopIdx:  StopIndex(boardStopIdx),
							AlightStopIdx: StopIndex(currStopIdx),
						}
						nextStopsUpdated = append(nextStopsUpdated, currStopId)
					}
				}

				prevArrival := prevBest[currStopId]
				if prevArrival == types.INFINITY {
					continue
				}

				earliestBoard := prevArrival + types.Timestamp(rt.MinTransferTime[currStopId])

				numTrips := rt.NumTripsInRoute[routeId]
				for potentialTripIdx := range numTrips {
					stopEvents := rt.StopEventsForTrip(routeId, potentialTripIdx)
					if stopEvents[currStopIdx].DepartureTime >= earliestBoard {
						if currentTripIdx < 0 || potentialTripIdx < uint32(currentTripIdx) {
							currentTripIdx = int(potentialTripIdx)
							boardStopId = currStopId
							boardStopIdx = uint32(currStopIdx)
						}

						break
					}
				}
			}
		}

		stopsUpdated = nextStopsUpdated
	}

	if best[end] == types.INFINITY {
		return nil
	}

	var legs []Label

	for stop := end; stop != start; {
		label := parent[stop]
		legs = append(legs, label)
		stop = label.BoardStopId
	}

	slices.Reverse(legs)

	return legs
}
