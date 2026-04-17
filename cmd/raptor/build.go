package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"router/pkg/debug"
	"router/pkg/gtfs"
	"router/pkg/raptor"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	switch os.Args[1] {
	case "serve":
		serverPort := flag.String("port", "3456", "port to serve on")
		flag.Parse()
		debug.StartServer(*serverPort)
	case "build":
		gtfsPath := flag.String("gtfs", "./gtfs.zip", "path to GTFS zip file")
		flag.Parse()
		gtfsTable, err := gtfs.ParseGtfs(*gtfsPath)
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
	default:
		printUsage()
		os.Exit(1)
	}

}

func printUsage() {
	fmt.Println("Usage: router-next <command> [flags]")
	fmt.Println("\nCommands:")
	fmt.Println("  serve      Start the debug web server")
	fmt.Println("  build      Build RAPTOR table")
}
