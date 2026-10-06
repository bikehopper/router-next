# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project

Public transit routing engine in Go. Parses a GTFS zip, builds a RAPTOR timetable for a single service date, and answers earliest-arrival queries. The Go module is named `router`, so imports are `router/pkg/...`. Go 1.26.1 (`.go-version`).

## Commands

```sh
make setup                                   # install lefthook git hooks
make test                                    # go test ./...
go test ./cmd/raptor -run TestRaptorBuild    # snapshot test only
go test ./pkg/utils -run TestSnapshotStr
go vet ./...
golangci-lint run ./...                      # v2 config in .golangci.yaml
make build                                   # -> ./router-next (gitignored)

go run ./cmd/raptor build -gtfs <gtfs.zip>             # parse + build, print table size
go run ./cmd/raptor serve -gtfs <gtfs.zip> -port 3456  # debug web server; run from repo root
```

Both subcommands default `-gtfs` to `./gtfs.zip`. The README's `go run ./cmd/raptor path/to/gtfs.zip` form is stale: a subcommand is required.

## Lint, hooks, CI

- The lefthook pre-commit hook runs `golangci-lint run --fix ./...`, re-stages the files, and runs `go vet ./...`. Lint failures don't block a commit (the hook joins lint and `git add` with `;`). Vet failures do.
- Linters are `default: fast` plus `wsl_v5` (blank-line and statement-cuddling rules, the most common finding) and `exhaustruct` (struct literals must set every field; the existing code follows this).
- The tree already has about 13 lint findings (mostly `wsl_v5`, plus `gocognit` on `Route`). Don't assume they came from your change.
- CI (`.github/workflows/test.yml`) runs only `go build ./...` and `go test ./...`. It doesn't lint.

## Architecture

Pipeline: `gtfs.ParseGtfs(zipPath)` → `*gtfs.GTFSTable` → `raptor.BuildRaptorTable(gtfsTable, date)` → `*raptor.RaptorTable` → `rt.Route(start, end, startTime)` → `[]raptor.Journey`.

**`pkg/types`** holds the shared dense IDs: `StopID`, `RouteID` (uint32), and `Timestamp`, which counts seconds since service-day midnight and can go past 24h as GTFS allows. `INFINITY` means unreachable.

**`pkg/gtfs`** has a generic CSV-in-zip reader (`parseCSVFile` plus a `RowParser[T]` for each file).
- It reads only `routes`, `trips`, `calendar`, `calendar_dates`, `stops`, `stop_times` and `transfers`.
- A missing file gives nil without an error. A row that fails to parse is silently skipped (marked TODO).
- `location_type` and `parent_station` are ignored, so stations and their platforms each get their own stop.
- `TripsForDate` combines `calendar` with the `calendar_dates` exceptions.
- GTFS string IDs have their own named types (`GTFSStopID`, `GTFSTripID`, …). Keep them separate from the dense `types.*` IDs.

**`pkg/raptor/build.go`** turns the GTFS table into a RAPTOR table for one date.
- `types.StopID` is the stop's index in `GTFSTable.Stops`, which follows `stops.txt` order.
- A RAPTOR "route" is not a GTFS route. Trips are grouped by identical stop sequence. Routes are sorted by their stop-sequence key string, which keeps `RouteID`s deterministic for the snapshots. Trips within a route are sorted by first departure.
- `RaptorTable.Routes[r]` is the GTFS route of route r's first trip.
- `RaptorTripID` is a temporary index into `TripsForDate(date)`, the filtered list of active trips. It is not an index into `GTFSTable.Trips`.
- Transfers: only same-stop rows with `transfer_type == 2` are used, and they become the per-stop `MinTransferTime`. Footpaths between different stops aren't implemented (see the TODO in `route.go`).

**`pkg/raptor/table.go`**: `RaptorTable` is a flat CSR layout.
- Each `...ByRoute` or `...ByStop` array is paired with a `First...Of...` offsets array of length N+1, with a trailing sentinel. Element i's data is `[off[i]:off[i+1]]`.
- Route r's stop events form a row-major block of `numTrips × numStopsInRoute`.
- Prefer the accessors (`StopsForRoute`, `StopEventsForTrip`, `TripInRoute`, `RoutesForStop`) over indexing the arrays directly.
- `NumTripsInRoute` has no sentinel. `NumRoutes()` is defined as its length.
- When you add a field, update `BuildRaptorTable` and `Sizeof()`. If it should be regression-tested, also update `SnapshotString()`.

**`pkg/raptor/route.go`** is round-based RAPTOR with `MAX_ROUNDS = 10`.
- It prunes with a cross-round `best[]` and with the target's arrival time.
- It returns the Pareto set: one `Journey` for each round count that improves arrival at `end`.
- Boarding scans the trips in order and assumes trips on a route never overtake each other.
- It isn't wired into the CLI or the debug server yet.

**`pkg/debug` and `static/index.html`** make up a debug HTTP server.
- It builds tables for today's date (`time.Now()`) and exposes `GET /routes/{routeId}/stops`.
- It serves a MapLibre page, and that page hardcodes `localhost:3456`.
- `./static` is a relative path, so start the server from the repo root.

**`pkg/utils`** has generic `Map` and `Reduce`, `SizeOf[T]`, and `SnapshotStr`.

For logging, the code prints progress and timings with `fmt.Printf` or `println`. There's no logger.

## Testing

- `cmd/raptor/build_test.go` builds tables from `cmd/raptor/testdata/gtfs_04162026.zip` for 2026-04-09 and 2026-04-20. It compares `SnapshotString()` with `cmd/raptor/snapshots/<date>.txt`.
- **If the snapshot doesn't match or is missing, the test fails and overwrites the snapshot file.**
  - To update a snapshot on purpose: run the test, review `git diff cmd/raptor/snapshots`, then re-run it to confirm it passes.
  - After an unexpected failure the tree holds a rewritten snapshot. Revert it if the change wasn't intended.
- Snapshots cover `MinTransferTime`, `StopIdsByRoute` and the offset arrays. They don't cover `StopEventsByRoute`, `TripsByRoute`, `Routes` or `Route()` results.
- The bundled feed is the 511 SF Bay regional feed: 40 agencies and about 320 MB uncompressed. It isn't Muni-only, despite what the README says. Most IDs carry an agency prefix (`SF:`, `BA:`, `mtc:`…), but some don't.
- Tests should pass explicit dates. Anything that uses `time.Now()` depends on the feed's calendar ranges.
