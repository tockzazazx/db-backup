# db-backup

A CLI tool for backing up databases, written in Go.

## Requirements

- Go 1.26+

## Project Structure

```
.
├── cmd/
│   └── db-backup/      # main entrypoint
│       └── main.go
├── internal/
│   ├── backup/         # core backup logic
│   └── config/         # configuration loading
├── go.mod
├── Makefile
└── README.md
```

## Configuration

Configuration is read from environment variables:

| Variable            | Description                          | Default                                |
|---------------------|--------------------------------------|----------------------------------------|
| `DB_URL`            | Connection string of the database    | `postgres://localhost:5432/postgres`   |
| `BACKUP_OUTPUT_DIR` | Directory where backups are written  | `./backups`                            |

## Usage

```sh
# Build
make build

# Run
make run

# Or directly
go run ./cmd/db-backup
```

## Development

```sh
# Run tests
make test

# Format & vet
make lint
```
