package gtfs

func routesById(routes []GTFSRoute) map[GTFSRouteID]*GTFSRoute {
	gtfsRouteMap := make(map[GTFSRouteID]*GTFSRoute, len(routes))
	for i := range routes {
		gtfsRouteMap[routes[i].GtfsId] = &routes[i]
	}

	return gtfsRouteMap
}

func servicesById(services []GTFSService) map[GTFSServiceID]*GTFSService {
	gtfsServiceMap := make(map[GTFSServiceID]*GTFSService, len(services))
	for i := range services {
		gtfsServiceMap[services[i].GtfsId] = &services[i]
	}

	return gtfsServiceMap
}

func serviceExceptionsByDateById(
	serviceExceptions []GTFSServiceException,
) map[GTFSServiceID]map[GTFSDate]GTFSServiceExceptionType {
	gtfsServiceExceptionMap := make(map[GTFSServiceID]map[GTFSDate]GTFSServiceExceptionType, len(serviceExceptions))
	for _, serviceException := range serviceExceptions {
		_, ok := gtfsServiceExceptionMap[serviceException.GtfsServiceId]
		if !ok {
			gtfsServiceExceptionMap[serviceException.GtfsServiceId] = make(map[GTFSDate]GTFSServiceExceptionType)
		}

		gtfsServiceExceptionMap[serviceException.GtfsServiceId][serviceException.Date] = serviceException.ExceptionType
	}

	return gtfsServiceExceptionMap
}
