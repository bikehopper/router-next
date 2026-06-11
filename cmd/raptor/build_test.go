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

func assertSnapshotMatches(t *testing.T, rt *raptor.RaptorTable, snapshotId string) {
	snapshot := rt.SnapshotString()
	fileName := fmt.Sprintf("./snapshots/%s.txt", snapshotId)
	bytes, err := os.ReadFile(fileName)

	if (err == nil && string(bytes) != snapshot) || errors.Is(err, os.ErrNotExist) {
		t.Errorf("%s.txt snapshot mismatch, regenrating", snapshotId)
		os.WriteFile(fileName, []byte(snapshot), 0644)
	} else if err != nil {
		log.Fatalln(err)
	}
}

func TestRaptorBuild(t *testing.T) {
	gtfsTable, err := gtfs.ParseGtfs("./testdata/gtfs_04162026.zip")
	if err != nil {
		t.Errorf("GTFS parsing failed")
	}

	d1 := time.Date(2026, time.April, 9, 0, 0, 0, 0, time.UTC)
	raptorTable1, err := raptor.BuildRaptorTable(gtfsTable, gtfs.TimeToGTFSDate(d1))
	if err != nil {
		t.Errorf("Rator Table generation failed")
	}

	assertSnapshotMatches(t, raptorTable1, d1.Format(time.DateOnly))

	d2 := time.Date(2026, time.April, 20, 0, 0, 0, 0, time.UTC)
	raptorTable2, err := raptor.BuildRaptorTable(gtfsTable, gtfs.TimeToGTFSDate(d2))
	if err != nil {
		t.Errorf("Rator Table generation failed")
	}

	assertSnapshotMatches(t, raptorTable2, d2.Format((time.DateOnly)))
}
