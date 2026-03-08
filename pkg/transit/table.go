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

	StopIds            []types.StopID
	FirstStopIDOfRoute []uint32

	StopEvents            []StopEvent
	FirstStopEventOfRoute []uint32
	NumTripsInRoute       []uint32

	RouteSegments           []RouteSegment
	FirstRouteSegmentOfStop []uint32
}
