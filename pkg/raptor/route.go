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

type Journey struct {
	NumLegs     int
	ArrivalTime types.Timestamp
	Legs        []Label
}

const MAX_ROUNDS = 10

func (rt *RaptorTable) Route(start types.StopID, end types.StopID, startTime types.Timestamp) []Journey {
	numStops := rt.NumStops()

	parents := make([][]Label, MAX_ROUNDS+1)

	rounds := make([][]types.Timestamp, MAX_ROUNDS+1)
	for k := range rounds {
		parents[k] = make([]Label, numStops)

		rounds[k] = make([]types.Timestamp, numStops)
		for s := range rounds[k] {
			rounds[k][s] = types.INFINITY
		}
	}

	rounds[0][start] = startTime

	best := make([]types.Timestamp, numStops)
	for i := range best {
		best[i] = types.INFINITY
	}

	best[start] = startTime

	stopsUpdated := []types.StopID{start}

	for round := 1; round <= MAX_ROUNDS; round++ {
		if len(stopsUpdated) == 0 {
			break
		}

		copy(rounds[round], rounds[round-1])

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
					arrivalTime := rt.StopEventsForTrip(routeId, uint32(currentTripIdx))[currStopIdx].ArrivalTime
					if arrivalTime < rounds[round][currStopId] {
						rounds[round][currStopId] = arrivalTime

						parents[round][currStopId] = Label{
							RouteId:       routeId,
							TripIdx:       uint32(currentTripIdx),
							BoardStopId:   boardStopId,
							BoardStopIdx:  StopIndex(boardStopIdx),
							AlightStopIdx: StopIndex(currStopIdx),
						}
						if arrivalTime < best[currStopId] {
							best[currStopId] = arrivalTime
							nextStopsUpdated = append(nextStopsUpdated, currStopId)
						}
					}
				}

				prevArrival := rounds[round-1][currStopId]
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

	var pareto []Journey

	for round := 1; round <= MAX_ROUNDS; round++ {
		if round == 0 {
			continue
		}

		if rounds[round][end] < rounds[round-1][end] {
			pareto = append(pareto, Journey{
				NumLegs:     round,
				ArrivalTime: rounds[round][end],
				Legs:        reconstructLegs(start, end, round, parents),
			})
		}
	}

	return pareto
}

func reconstructLegs(start, end types.StopID, round int, parents [][]Label) []Label {
	var legs []Label

	stop := end
	for round > 0 && stop != start {
		label := parents[round][stop]
		legs = append(legs, label)
		stop = label.BoardStopId
		round--
	}

	slices.Reverse(legs)

	return legs
}
