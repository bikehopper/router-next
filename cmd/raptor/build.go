package main

import (
	"fmt"
	"os"
	"time"

	"router/pkg/gtfs"
	"router/pkg/raptor"
	"router/pkg/storage"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: %s <gtfs_zip> <pebble_dir>\n", os.Args[0])
		os.Exit(1)
	}

	db, err := storage.Open(os.Args[2])
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to open pebble DB: %v\n", err)
		os.Exit(1)
	}
	defer db.Close() // Flushes all buffered writes to disk

	gtfsTable, err := gtfs.ParseGtfs(os.Args[1], db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Parse GTFS error: %v\n", err)
		os.Exit(1)
	}

	raptorTable, err := raptor.BuildRaptorTable(gtfsTable, gtfs.TimeToGTFSDate(time.Now()))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Build Raptor Table error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Total size: %.2fMB\n", float64(raptorTable.Sizeof())/1024/1024)
}
