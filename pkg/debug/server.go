package debug

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"router/pkg/gtfs"
	"router/pkg/raptor"
	"router/pkg/types"
	"strconv"
	"time"
)

func buildTables(gtfsPath string) (*gtfs.GTFSTable, *raptor.RaptorTable) {
	gtfsTable, err := gtfs.ParseGtfs(gtfsPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	raptorTable, err := raptor.BuildRaptorTable(gtfsTable, gtfs.TimeToGTFSDate(time.Now()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	return gtfsTable, raptorTable
}

func StartServer(port string, gtfsPath string) {
	_, raptorTable := buildTables(gtfsPath)

	// Serve Static Files
	fs := http.FileServer(http.Dir("./static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	// Serve route stop ids
	http.HandleFunc("/routes/{routeId}/stops", func(w http.ResponseWriter, r *http.Request) {
		routeIdStr := r.PathValue("routeId")
		routeId, err := strconv.ParseUint(routeIdStr, 10, 32)
		if err != nil {
			http.Error(w, "invalid route id", http.StatusBadRequest)
			return
		}

		stopIds := raptorTable.StopsForRoute(types.RouteID(routeId))
		json.NewEncoder(w).Encode(stopIds)
	})

	// Serve Index
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" {
			http.ServeFile(w, r, "static/index.html")
			return
		}
		fs.ServeHTTP(w, r)
	})

	fmt.Printf("Starting server on port %s \n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		fmt.Printf("Server failed: %v\n", err)
	}
}
