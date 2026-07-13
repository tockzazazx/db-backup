# boxdb (db-backup)

**English** | [ไทย](README.th.md)

A CLI tool for backing up databases, written in Go. Targets Ubuntu/Linux.

## Install (Ubuntu)

### Recommended: .deb package (apt)

```sh
wget https://github.com/tockzazazx/db-backup/releases/latest/download/boxdb_amd64.deb
sudo apt install ./boxdb_amd64.deb
```

To install a specific (or prerelease) version instead, the tag and the
versioned filename must match — mixing `latest` with a versioned filename
404s:

```sh
VERSION=0.3.0   # <- without the "v"
wget "https://github.com/tockzazazx/db-backup/releases/download/v${VERSION}/boxdb_${VERSION}_amd64.deb"
sudo apt install "./boxdb_${VERSION}_amd64.deb"
```

Upgrading works the same way: download the newer .deb and `apt install` it
again. apt tracks the package, so you can inspect or remove it later with:

```sh
apt list --installed | grep boxdb
sudo apt remove boxdb
```

### Alternative: install script (wget)

For machines where you can't use apt. Installs the latest release binary to
`/usr/local/bin`:

```sh
wget -qO- https://github.com/tockzazazx/db-backup/releases/latest/download/install.sh | bash
```

> **Warning:** pick one method and stick with it. The script installs to
> `/usr/local/bin` while the .deb installs to `/usr/bin`, and PATH prefers
> `/usr/local/bin` — so a leftover script install shadows any .deb install
> and `boxdb --version` keeps showing the old version. If you switch from
> the script to the .deb, remove the old binary first:
>
> ```sh
> sudo rm /usr/local/bin/boxdb && hash -r
> ```

### Verify

```sh
boxdb --version
which -a boxdb   # should list exactly one path
```

## Usage

```sh
boxdb --version   # print version
boxdb config      # show saved S3 config
boxdb test        # test the S3 connection
boxdb upload      # upload new files from the configured paths
boxdb list        # list date folders on S3
boxdb list <date> # list files in one date folder
boxdb download <date>[/<file>] <dest-dir>   # download from S3
boxdb schedule    # show / install / remove the auto-upload schedule
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

Inspect what's stored with `boxdb list`:

```sh
boxdb list                # show available date folders
boxdb list 2026-07-08     # show files in that folder
# ubuntu-server-01/2026-07-08:
#   3.pg    300.0 KB  2026-07-08 14:46:25
# total: 1 files, 300.0 KB
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

## Download

Fetch one file or a whole date folder back from S3:

```sh
boxdb download 2026-07-08/db1.pg /root/restore   # one file
boxdb download 2026-07-08 /root/restore          # every file in that date folder
```

- The destination must be a directory; it is created if missing.
- Existing files are never overwritten — a counter goes in before the
  extension instead: `db1.pg` → `db1 (1).pg` (compound extensions are kept
  together: `backup.tar.gz` → `backup (1).tar.gz`).
- Files are downloaded to a temporary `.part` name and renamed only when
  complete, so a file with its final name is never half-written.

## Scheduled uploads

Run `boxdb upload` automatically via a systemd timer — daily, weekly, or
monthly (one schedule per machine):

```sh
sudo boxdb schedule --daily 03:00                    # every day at 03:00
sudo boxdb schedule --weekly saturday --at 03:00     # every Saturday
sudo boxdb schedule --weekly sat,sun --at 03:00      # several days a week
sudo boxdb schedule --monthly 1 --at 03:00           # the 1st of every month
sudo boxdb schedule --monthly last --at 03:00        # the last day of every month
boxdb schedule                                       # show status, next run, last result
sudo boxdb schedule --remove                         # uninstall
```

`--monthly` accepts 1-28 or `last`. Days 29-31 are rejected on purpose:
systemd would silently skip months without that day (no backup in
February!) — `last` always means the final day, whether that's the 28th,
29th, 30th, or 31st.

Notes:

- The upload runs as the user who invoked `sudo` (via `$SUDO_USER`), so it
  reads that user's `~/.config/boxdb/config.json` — run `boxdb config` and
  `boxdb test` as that user first. Installing refuses to proceed when that
  user has no config.
- In a root shell reached via `sudo su` / `sudo -i`, `$SUDO_USER` still
  points at the original user — pass `--user` to pick explicitly, e.g.
  `boxdb schedule --daily 03:00 --user root`.
- `Persistent=true` is set: if the machine was off at the scheduled time,
  the upload runs right after boot instead of being skipped.
- Run logs: `journalctl -u boxdb-upload.service`.
- One schedule per machine; installing again overwrites the previous time.

## Project Structure

```
.
├── cmd/
│   └── boxdb/          # main entrypoint
├── internal/
│   ├── config/         # config file handling (~/.config/boxdb/config.json)
│   ├── s3/             # MinIO client wrapper (check, upload, list)
│   └── schedule/       # systemd timer install/status/remove
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
