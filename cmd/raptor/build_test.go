package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"router/pkg/gtfs"
	"router/pkg/raptor"
	"testing"
)

var snapshotId int = 1

func assertSnapshotMatches(t *testing.T, rt *raptor.RaptorTable) {
	snapshot := rt.SnapshotString()
	fileName := fmt.Sprintf("./snapshots/%d.txt", snapshotId)
	bytes, err := os.ReadFile(fileName)

	if (err == nil && string(bytes) != snapshot) || errors.Is(err, os.ErrNotExist) {
		t.Errorf("%d.txt snapshot mismatch, regenrating", snapshotId)
		os.WriteFile(fileName, []byte(snapshot), 0644)
	} else if err != nil {
		log.Fatalln(err)
	}

	snapshotId++
}

func TestRaptorBuild(t *testing.T) {
	gtfsTable, err := gtfs.ParseGtfs("./testdata/gtfs_04162026.zip")
	if err != nil {
		t.Errorf("GTFS parsing failed")
	}

	raptorTable1, err := raptor.BuildRaptorTable(gtfsTable, gtfs.TimeToGTFSDate(time.Date(2026, time.April, 9, 0, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Errorf("Rator Table generation failed")
	}

	assertSnapshotMatches(t, raptorTable1)

	raptorTable2, err := raptor.BuildRaptorTable(gtfsTable, gtfs.TimeToGTFSDate(time.Date(2026, time.April, 20, 0, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Errorf("Rator Table generation failed")
	}

	assertSnapshotMatches(t, raptorTable2)
}
