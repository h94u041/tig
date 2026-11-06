# Security

## Reporting a problem

Please use GitHub's private reporting: [Report a vulnerability](https://github.com/h94u041/tig/security/advisories/new). Do not open a public issue for something that could be used against people before it is fixed.

Say what you found, how to reproduce it, and which version (`tigwx --version`). tig is maintained by one person: expect an answer within a week. A confirmed problem gets a fix in the next release and a note in the [changelog](CHANGELOG.md). Fixes go to the latest release only.

## What tig promises

- **tig runs nothing.** Reading a project or a document never runs a program from it. `tigwx-<name>` extensions run only when you type that command, and only from PATH.
- **Paths stay inside the project.** Trees can come from someone else's repository: putting files back refuses any path that could land outside the project or inside `.git` or `.tig`, Windows names included, and never writes through a symbolic link.
- **Records are checked.** Objects and ledger records are named by their hash and checked on every read.
- **Agents can only propose.** Over MCP an agent cannot confirm an intent, confirm a link or accept a commit: those tools are not served.
- **A hub keeps documents only.** A push carries the documents and the chain, never the code, and a hub shows others' documents with no scripts allowed.

## What tig does not promise

These are limits, not bugs. Please do not report them as vulnerabilities, but do tell us if you see a way to narrow them.

- **`by` and `via` are claims.** tig keeps what it is told. Anyone who can write to `.tig` can write `by human`. Records are not signed, so the ledger tells you what was said, and Git history and code review tell you who could have said it.
- **Secrets are kept out by name, not by content.** A file named `.env` or `id_rsa` stays out of snapshots unless Git tracks it; a key inside `config.yaml` goes in, and `.tig` goes into Git. Treat a key that reached `.tig` as leaked and replace it. See [the footprint](docs/footprint.md).
- **Documents are data, and may be written to mislead.** Text in a repository reaches an agent as it is, instructions included. See [the MCP server](docs/mcp.md).
- **A check an agent recorded is what the agent said.** tig does not re-run it.
- **`tighub` is not an internet-facing server by itself.** It speaks plain `http` and has no rate limits. See [the hub](docs/hub.md).

## Checking what you downloaded

Every release has a `SHA256SUMS` file. Releases from v0.2.0 on also have a [provenance attestation](https://docs.github.com/actions/security-for-github-actions/using-artifact-attestations): `gh attestation verify <file> --repo h94u041/tig` shows that this repository's release workflow built the file. The setup programs are not code-signed yet, so Windows and macOS may warn you before running them.

**Upgrade from v0.1.0.** Its programs were built with Go 1.25.0, and `govulncheck` lists standard-library vulnerabilities in code it can reach. Its hub also uses a version of goldmark with a known cross-site scripting problem, and it did not stop `.env` and private keys from entering snapshots. If a project ran v0.1.0 with such a file not ignored by Git, look in `.tig/objects`, and treat that key as leaked.
