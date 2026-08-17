# MeshTurn

MeshTurn is a small command-line ledger for screen-printing reclamation batches. A batch keeps its screen manifest in order while each screen is marked `clean`, `rework`, or `retire`. A complete batch closes into a report that cannot be changed afterward.

The application uses only the Go standard library and stores data in `meshturn.json` by default. Pass `--ledger PATH` before a command to use another ledger.

## Commands

Open a batch with a unique ID and ordered, comma-separated screens:

```text
go run ./cmd/meshturn open --id april-01 --ink-family water --screens frame-01,frame-02 --note "first washout"
```

The `--note` flag is optional. Supplying `--note ""` records an explicit empty note, while omitting it leaves the note absent.

Record one disposition for each screen, then close the batch:

```text
go run ./cmd/meshturn record --id april-01 --screen frame-01 --outcome clean
go run ./cmd/meshturn record --id april-01 --screen frame-02 --outcome rework
go run ./cmd/meshturn close --id april-01
```

Inspect one batch or find active and attention-needed batches:

```text
go run ./cmd/meshturn show --id april-01
go run ./cmd/meshturn attention
```

Every successful command writes one JSON value to standard output. The bounded smoke workflow uses a temporary ledger and does not modify the default ledger:

```text
go run ./cmd/meshturn smoke
```

Run the test suite with:

```text
go test ./...
```
