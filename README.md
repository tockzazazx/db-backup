# boxdb (db-backup)

A CLI tool for backing up databases, written in Go. Targets Ubuntu/Linux.

## Install (Ubuntu)

### Via install script (wget)

```sh
wget -qO- https://github.com/tockzazazx/db-backup/releases/latest/download/install.sh | bash
```

### Via .deb package (apt)

```sh
wget https://github.com/tockzazazx/db-backup/releases/latest/download/boxdb_0.1.0_amd64.deb
sudo apt install ./boxdb_0.1.0_amd64.deb
```

(Replace the version/arch in the filename with the latest release — see the
[releases page](https://github.com/tockzazazx/db-backup/releases).)

### Verify

```sh
boxdb --version
```

## Usage

```sh
boxdb --version   # print version
boxdb run         # run a backup
```

## Configuration

Configuration is read from environment variables:

| Variable            | Description                          | Default                                |
|---------------------|--------------------------------------|----------------------------------------|
| `DB_URL`            | Connection string of the database    | `postgres://localhost:5432/postgres`   |
| `BACKUP_OUTPUT_DIR` | Directory where backups are written  | `./backups`                            |

## Project Structure

```
.
├── cmd/
│   └── boxdb/          # main entrypoint
├── internal/
│   ├── backup/         # core backup logic
│   └── config/         # configuration loading
├── scripts/
│   ├── build-deb.sh    # package a .deb from a built binary
│   └── install.sh      # end-user install script
├── .github/workflows/
│   └── release.yml     # builds binaries + .deb on tag push
├── go.mod
├── Makefile
└── README.md
```

## Development

Requires Go 1.26+.

```sh
make build        # build for the current platform (bin/boxdb)
make build-linux  # cross-compile linux amd64 + arm64
make deb          # build .deb packages (requires dpkg-deb, i.e. run on Linux)
make test
make lint
```

## Releasing

Push a tag and GitHub Actions builds the binaries and .deb packages and
attaches them to a GitHub release:

```sh
git tag v0.2.0
git push origin v0.2.0
```

Version info is injected at build time via `-ldflags` (`make build VERSION=v0.2.0`).
