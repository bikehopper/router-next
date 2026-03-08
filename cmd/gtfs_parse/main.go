package main

import (
	"archive/zip"
	"fmt"
	"os"

	"router/pkg/transit"
)

func main() {
	reader, err := zip.OpenReader(os.Args[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	defer reader.Close()

	files := make(map[string]*zip.File)
	for _, file := range reader.File {
		files[file.Name] = file
	}

	stops, err := transit.ParseStops(files["stops.txt"])
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v", err)
		os.Exit(1)
	}
	fmt.Printf("detected %d stops\n", len(stops))
	for idx, stop := range stops[0:10] {
		fmt.Printf(
			"stop %d: id %s name %s coords (%s, %s)\n",
			idx,
			stop.ID,
			stop.Name,
			stop.Lat,
			stop.Lon,
		)
	}
}
