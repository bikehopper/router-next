package transit

import "router/pkg/types"

type Stop struct {
	Lat, Lon float64
	MinTransferTime uint32
	Name string
}

type StopEvent struct {
	ArrivalTime uint32
	DepartureTime uint32
}

type RouteSegment struct {
	RouteId types.RouteId
	StopIndex types.StopIndex
}

type RaptorTable struct {
	Stops []Stop
	RouteNames []string

	StopIds []types.StopId
	StopIdOffsetForRoute []uint32
	
	StopEvents []StopEvent
	StopEventOffsetForRoute []uint32
	NumTripsInRoute []uint32

	RouteSegmentsForStop [][]RouteSegment
}