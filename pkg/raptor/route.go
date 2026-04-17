package raptor

import "router/pkg/types"

type RouteLeg struct {
	routeId      types.RouteID
	tripIdx      uint32
	startStopIdx uint32
	endStopIdx   uint32
}

type Journey struct {
	arrivalTime types.Timestamp
	legs        []RouteLeg
}

func (rt *RaptorTable) Route(start types.StopID, end types.StopID, startTime types.Timestamp) {
	currentBest := make([]Journey, rt.NumStops())

	var round = 0

	var stopsToConsider = []types.StopID{start}

	for {
		var nextStopsToConsider []types.StopID

		for _, stopId := range stopsToConsider {
			currentJourney := currentBest[stopId]
			routesForStop := rt.RoutesForStop(stopId)

			for _, routeSegment := range routesForStop {
				routeId := routeSegment.RouteId
				startStopIdx := routeSegment.StopIndex

				stopsForRoute := rt.StopsForRoute(routeId)
				for tripIdx := range rt.NumTripsInRoute[routeId] {
					stopEvents := rt.StopEventsForTrip(routeId, tripIdx)
					if stopEvents[startStopIdx].DepartureTime > currentJourney.arrivalTime {
						for endStopIdx := startStopIdx; endStopIdx < uint32(len(stopEvents)); endStopIdx++ {
							stopEvent := stopEvents[endStopIdx]

							stopId := stopsForRoute[endStopIdx]
							if stopEvent.ArrivalTime < currentBest[stopId].arrivalTime {
								currentBest[stopId] = Journey{
									arrivalTime: stopEvent.ArrivalTime,
									legs: append(
										currentJourney.legs,
										RouteLeg{
											routeId:      routeId,
											tripIdx:      tripIdx,
											startStopIdx: startStopIdx,
											endStopIdx:   endStopIdx,
										},
									)}
								nextStopsToConsider = append(nextStopsToConsider, stopId)
							}
						}

						break
					}
				}
			}
		}

		stopsToConsider = nextStopsToConsider
		round++
	}
}
