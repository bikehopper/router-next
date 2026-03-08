package transit

import "router/pkg/types"

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
	Stops  []Stop
	Routes []Route

	StopIdsByRoute     []types.StopID
	FirstStopIDOfRoute []uint32

	StopEventsByRoute     []StopEvent
	FirstStopEventOfRoute []uint32
	NumTripsInRoute       []uint32

	RouteSegmentsByStop     []RouteSegment
	FirstRouteSegmentOfStop []uint32
}

func buildRaptorTable(gtfsTable GtfsTable) (RaptorTable, error) {
	gtfsStopIdToIdx := make(map[string]types.StopID, len(gtfsTable.Stops))
	for idx, stop := range gtfsTable.Stops {
		gtfsStopIdToIdx[stop.GtfsID] = types.StopID(idx)
	}

	return RaptorTable{
		Stops:                   nil,
		Routes:                  nil,
		StopIdsByRoute:          nil,
		FirstStopIDOfRoute:      nil,
		StopEventsByRoute:       nil,
		FirstStopEventOfRoute:   nil,
		NumTripsInRoute:         nil,
		RouteSegmentsByStop:     nil,
		FirstRouteSegmentOfStop: nil,
	}, nil
}
