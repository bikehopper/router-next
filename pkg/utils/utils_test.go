package utils

import (
	"testing"
)

func TestSnapshotStr(t *testing.T) {
	// Test case 1
	lessThanLineWidth := make([]uint32, 6)

	formatted1 := SnapshotStr(lessThanLineWidth)
	expected1 := "0,0,0,0,0,0"
	if formatted1 != expected1 {
		t.Errorf("got:\n%s\nexpected:%s", formatted1, expected1)
	}

	// Test case 2
	moreThanLineWidth := make([]uint32, 44)

	formatted2 := SnapshotStr(moreThanLineWidth[:])
	expected2 := "0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0,0\n0,0,0,0"
	if formatted2 != expected2 {
		t.Errorf("got:\n%s\nexpected:%s", formatted2, expected2)
	}
}
