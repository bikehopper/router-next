package main

import (
	"fmt"
	"os"
	"time"

	"router/pkg/gtfs"
	"router/pkg/raptor"
)

func main() {
	gtfsTable, err := gtfs.ParseGtfs(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	raptorTable, err := raptor.BuildRaptorTable(gtfsTable, gtfs.TimeToGTFSDate(time.Now()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	fmt.Printf("Total size: %.2fMB\n", float64(raptorTable.Sizeof())/1024/1024)
}
