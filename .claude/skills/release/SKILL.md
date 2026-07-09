---
name: release
description: Release a new boxdb version — tags v<version> and pushes it so GitHub Actions builds linux binaries + .deb packages and publishes the GitHub release. Use when the user asks to release, cut a version, or tag a release.
argument-hint: <version, e.g. 0.7.0 or 0.7.0-beta.1>
---

# Release boxdb

Release the version given in `$ARGUMENTS`. The release itself is fully
automated by `.github/workflows/release.yml` — pushing a tag `v*` builds
linux amd64/arm64 binaries and .deb packages and attaches them to a GitHub
release. This skill's job is the checks around that push.

## Steps

1. **Normalize the version.** Accept `0.7.0` or `v0.7.0` → tag is always
   `v0.7.0`. Valid forms: `vX.Y.Z` or `vX.Y.Z-<pre>` (e.g. `v0.7.0-beta.1`).
   A hyphenated tag is automatically published as a **prerelease** by the
   workflow (it won't move `releases/latest`) — tell the user this when
   they release one. If no version was given, ask for it; suggest the next
   patch/minor after `git tag --sort=-v:refname | head -5`.

2. **Check the tag doesn't already exist:** `git tag -l <tag>` and
   `git ls-remote --tags origin <tag>`. If it exists, stop and tell the user.

3. **Branch check.** Run `git branch --show-current`:
   - On `main`: continue.
   - On any other branch: **ask the user before doing anything** (use
     AskUserQuestion): "อยู่บน branch `<name>` จะเอาเข้า main ก่อน release
     ไหม" with options:
     - **Merge เข้า main ก่อน (แนะนำ)** — checkout main, pull,
       `git merge --no-ff <branch>`, push, then tag from main.
     - **Release จาก branch นี้เลย** — tag the branch HEAD directly;
       recommend a prerelease version (`-beta.N`) if they picked a plain
       X.Y.Z, but don't force it.
     - Cancel.

4. **Working tree must be clean** (`git status --porcelain`). Uncommitted
   changes → stop and ask whether to commit them first (they won't be in
   the release binary otherwise).

5. **Sync + verify:** `git pull`, then `go vet ./... && make build` and a
   smoke test: `./bin/boxdb --version` prints the expected version
   (Makefile injects it via git describe/ldflags after tagging — at this
   point just confirm it builds and runs).

6. **Tag and push:**
   ```sh
   git tag <tag>
   git push origin <tag>
   ```

7. **Watch the workflow** (takes ~1 min, occasionally queues for several):
   ```sh
   gh run list --repo tockzazazx/db-backup --workflow Release --branch <tag> --limit 1
   gh run watch <run-id> --repo tockzazazx/db-backup --exit-status
   ```
   If it fails, show the failing step's log (`gh run view <run-id> --log-failed`)
   and stop — do not retry by re-tagging without a fix.

8. **Verify the release assets** — must contain all 7:
   `boxdb_<ver>_amd64.deb`, `boxdb_<ver>_arm64.deb`, `boxdb_amd64.deb`,
   `boxdb_arm64.deb`, `boxdb_linux_amd64`, `boxdb_linux_arm64`, `install.sh`:
   ```sh
   gh release view <tag> --repo tockzazazx/db-backup --json assets,isPrerelease
   ```

9. **Report to the user** (in Thai): the released version, and the Ubuntu
   install/upgrade command:
   ```sh
   wget https://github.com/tockzazazx/db-backup/releases/latest/download/boxdb_amd64.deb
   sudo apt install ./boxdb_amd64.deb
   ```
   For a prerelease, give the exact versioned URL instead
   (`releases/download/<tag>/boxdb_<ver>_amd64.deb`) since `latest` skips
   prereleases.
