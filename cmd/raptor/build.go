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

	rt, err := transit.BuildRaptorTable(*gtfsTable)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}

	fmt.Printf("Extracted %d routes\n", rt.NumRoutes())
	fmt.Printf("Total size: %.2fMB\n", float64(rt.Sizeof())/1024/1024)
}
