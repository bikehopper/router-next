# router-next

A public transit routing engine written in Go. It parses GTFS feeds and
builds a RAPTOR timetable for fast, multi-round transit journey queries.

## Setup

Requires Go 1.26+.

```sh
git clone git@github.com:bikehopper/router-next.git
cd router-next
make setup  # installs git hooks via lefthook
```

## Development

Run the tests:

```sh
make test
```

Build a RAPTOR table from a GTFS zip (prints the table's in-memory size):

```sh
go run ./cmd/raptor path/to/gtfs.zip
```

A sample San Francisco Muni feed is bundled at
`cmd/raptor/testdata/gtfs_04162026.zip` for testing.
