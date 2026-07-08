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
boxdb config      # show saved S3 config
boxdb test        # test the S3 connection
boxdb upload      # upload new files from the configured paths
boxdb run         # run a backup
```

## Upload

`boxdb upload` sweeps every directory in `paths` and uploads files into the
bucket under `<folder>/<upload-date>/`, e.g. `ubuntu-server-01/2026-07-08/db1.pg`.

- Only files that have never been uploaded are sent — a file counts as
  uploaded when an object with the same name already exists under `folder`,
  in any date subfolder.
- Files removed locally are never deleted from the bucket.
- Subdirectories and dotfiles are skipped.

```sh
boxdb upload
# upload: /var/backups/3.pg -> ubuntu-server-01/2026-07-08/3.pg (300.0 KB)
# skip:   2.pg (already uploaded)
# done: 1 uploaded, 1 skipped
```

## S3 Configuration

Save S3 credentials (stored per-user at `~/.config/boxdb/config.json`,
mode 0600 — no sudo needed):

```sh
boxdb config \
  --endpoint https://s3.example.com \
  --access AKIA... \
  --secret secret... \
  --bucket my-backups \
  --folder ubuntu-server-01 \
  --paths /var/backups,/opt/data
```

- `--endpoint` accepts `host:port` or a URL; an `https://` scheme implies TLS
  (or pass `--ssl` explicitly).
- `--folder` is the folder (object prefix) inside the bucket where this
  machine's files are uploaded. It is created automatically if missing.
- `--paths` is a comma-separated list of local directories whose files will
  be uploaded into that folder.

Run `boxdb config` with no flags to view the saved config (secret masked),
and pass only the flags you want to change to update individual fields.

Test connectivity (uses the [MinIO Go client](https://github.com/minio/minio-go)):

```sh
boxdb test
# connecting to s3.example.com (bucket "my-backups", ssl=true)...
# OK: connection successful, bucket is accessible
# OK: folder "ubuntu-server-01" created in bucket
# OK: local path /var/backups
```

`boxdb test` also creates the configured folder when missing and warns about
local paths that don't exist. Errors are reported for missing config,
unreachable endpoints, bad credentials, and missing buckets.

## Backup Configuration

Read from environment variables:

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
