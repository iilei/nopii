# Contributing

Run the standard checks before submitting a change:

```sh
go test ./...
go vet ./...
gofmt -w .
```

Keep the CLI pipe-first and avoid adding network dependencies to the default
processing path. Any incompatible structured input change must use a new wire
format version rather than mutating `nopii-v1`.
