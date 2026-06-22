# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Commands

```sh
go build ./...       # build all packages
go run ./cmd         # run the main program
go test ./...        # run all tests
go vet ./...         # run vet
```

## Architecture

- Module `aicode` (Go 1.26.3).
- `cmd/` — program entry point (`package main`).
- `queue/` — generic `Queue[T any]` data structure backed by a slice. Methods: `Enqueue`, `Dequeue`, `Front`, `Len`, `IsEmpty`.
