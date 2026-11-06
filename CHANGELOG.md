# Changelog

What a user of tig would notice, newest first. The `.tig` format is versioned, and every release reads what earlier ones wrote.

## Unreleased

### Security

- Releases are built with the latest Go instead of the oldest `go.mod` allows. v0.1.0's programs were built with Go 1.25.0: please upgrade. See [SECURITY.md](SECURITY.md).
- goldmark upgraded past a cross-site scripting problem (GO-2026-5320), and every hub answer now carries a `Content-Security-Policy` allowing no scripts.
- Files that look like keys or credentials (`.env`, `id_rsa`, `*.pem` and others) stay out of snapshots unless Git tracks them. `status` lists them. [Which ones](docs/footprint.md).
- `push` refuses to send a token over plain `http` to another machine unless you say `-insecure`, and follows no redirect.
- `tighub`: timeouts for slow clients; asking which objects it lacks now takes the owner's token and is limited to 4 MB; tokens under 16 characters are refused; errors go to its log and not to the client.
- Text output leaves out control characters, so someone else's document cannot write to your terminal. `-json` still gives the data as it is.
- Workflows pin their actions by hash, run `govulncheck` and CodeQL, and releases carry a provenance attestation.

### Added

- `tigwx relate`, and the `relate_intents` MCP tool: say what two intents have to do with each other (`depends_on`, `part_of`, `parallel`, or your own word). A confirmed `depends_on` brings the other intent into a set, one step. A confirmed `parallel` is noted on the sentence and takes nothing out. [How](docs/chain.md).
- `tigwx log <intent>`, and the `intent_history` MCP tool: everything that happened to an intent.
- `tigwx sync -m why`: an edit someone meant is recorded as `revised`, with the reason, not as `touched`.
- `-source` on `mark`, `describe`, `confirm`, `reject`, `retire` and `sync`: what a decision was made from.
- `tigwx set -i i7f3a#2` says what that revision said and whether the intent has moved on, instead of silently answering with the latest words. A revision it never had is missing.
- A set has a `digest` naming what it holds.
- Decisions taken apart on two machines are shown: `intent-disputed` in `check`, a count in `status`, and `forks` in `show` and `log`.
- `mark` and `describe` answer with the intent's short ID.

### Changed

- **With no `-by` and no `user.by`, a terminal records `human` and anything else records `unknown`.** It used to be `human` everywhere, so a script's decisions looked like a person's, and its links started confirmed. A script that speaks for a person should say `-by human`, or set `user.by`.
- `mark` and `describe` print the intent first. Scripts should use `-json`.
- `blocks` reads its file relative to `-C`, as if started there.
- Requires Go 1.26 to build.

### Fixed

- `set -i` with more of the hash than the short ID now finds the intent.
- A file a work deleted is no longer reported as not following its design.

### Removed

- Unused code that could start a shell. It was never called, and it contradicted "tig runs nothing".

## 0.1.0

First release.
