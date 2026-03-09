package transit

import (
	"fmt"
	"router/pkg/types"
	"sort"
)

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
	StopIdsByRoute     []types.StopID
	FirstStopIDOfRoute []uint32

	StopEventsByRoute     []StopEvent
	FirstStopEventOfRoute []uint32
	NumTripsInRoute       []uint32

	RouteSegmentsByStop     []RouteSegment
	FirstRouteSegmentOfStop []uint32
}

func BuildRaptorTable(gtfsTable GtfsTable) (RaptorTable, error) {
	println("Building raptor table")

	gtfsStopIdToIdx := make(map[GtfsStopID]types.StopID, len(gtfsTable.Stops))
	for idx, stop := range gtfsTable.Stops {
		gtfsStopIdToIdx[stop.GtfsID] = types.StopID(idx)
	}

	activeGtfsTripIds := make(map[GtfsTripID]types.TripID, len(gtfsTable.Trips))
	for idx, trip := range gtfsTable.Trips {
		// TODO: filter by service day
		activeGtfsTripIds[trip.GtfsID] = types.TripID(idx)
	}

	stopTimesByTrip := make(map[GtfsTripID][]GtfsStopTime)

	for _, st := range gtfsTable.StopTimes {
		if _, ok := activeGtfsTripIds[st.GtfsTripID]; ok {
			stopTimesByTrip[st.GtfsTripID] = append(stopTimesByTrip[st.GtfsTripID], st)
		}
	}

	for _, stopTimes := range stopTimesByTrip {
		sort.Slice(stopTimes, func(i int, j int) bool {
			return stopTimes[i].StopSequence < stopTimes[j].StopSequence
		})
	}

	for _, trip := range gtfsTable.Trips[0:10] {
		for _, st := range stopTimesByTrip[trip.GtfsID] {
			fmt.Printf("trip %s seq %d stop %s arr %d dep %d\n",
				st.GtfsTripID,
				st.StopSequence,
				st.GtfsStopID,
				st.ArrivalTime,
				st.DepartureTime,
			)
		}
	}

	return RaptorTable{
		StopIdsByRoute:          nil,
		FirstStopIDOfRoute:      nil,
		StopEventsByRoute:       nil,
		FirstStopEventOfRoute:   nil,
		NumTripsInRoute:         nil,
		RouteSegmentsByStop:     nil,
		FirstRouteSegmentOfStop: nil,
	}, nil
}
