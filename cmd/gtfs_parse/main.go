package main

import (
	"fmt"
	"os"

	"router/pkg/transit"
)

func main() {
	gtfsTable, err := transit.ParseGtfs(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	_, err = transit.BuildRaptorTable(*gtfsTable)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
}
