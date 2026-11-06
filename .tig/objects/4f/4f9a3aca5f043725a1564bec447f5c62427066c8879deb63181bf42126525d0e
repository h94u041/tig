# tig

**Markdown as source.** tig is a ledger of intent and footprint: it makes a project's Markdown documents the source of truth for its code, hosts them as an MCP server that any AI agent can build from, and records what each line means, who agreed with it, and every construction that followed it.

[![Go Reference](https://pkg.go.dev/badge/github.com/h94u041/tig.svg)](https://pkg.go.dev/github.com/h94u041/tig)
[![test](https://github.com/h94u041/tig/actions/workflows/test.yml/badge.svg)](https://github.com/h94u041/tig/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT)

**English** · [繁體中文](i18n/README.zh-TW.md) · [Deutsch](i18n/README.de.md) · [Français](i18n/README.fr.md) · [Русский](i18n/README.ru.md) · [日本語](i18n/README.ja.md)

> This tig is not [tig, the text-mode interface for Git](https://github.com/jonas/tig). That is why its command is called `tigwx`.

## Why tig

Agents now write code faster than anyone can read it. What gets lost is not the code but everything around it:

- **Intent.** Git records which lines changed, not which part of the design they were for. Six months later nobody can say why a file looks the way it does.
- **Agreement.** A design document says what should be true, but not who said so: a person, or an agent guessing. Docs drift from code silently, and an edit to one sentence looks the same as a change of mind about everything around it.
- **Handoff.** Every agent session starts cold. It re-reads the project, guesses which documents are current, and repeats work another agent already did, or undoes it.

tig closes these gaps with one idea: the design is written as Markdown, each line carries a small mark that leads to what it means, and every piece of work is recorded against that meaning. tig records; it does not judge. Whether a piece of work is good enough is decided by you or by the tool you run on top of it, and tig keeps the decision.

## What tig can do

**It can answer these, from the command line, as JSON, or to an agent over MCP:**

| You want to know | Ask |
|---|---|
| What does this line mean, who said so, and has anyone agreed? | `tigwx show i7f3a` |
| What happened to this intent, and why did it change? | `tigwx log i7f3a` |
| What did this requirement say when we built against it? | `tigwx set -i i7f3a#1` |
| Which works changed this file, and what were they for? | `tigwx trail src/cart.go` |
| What do I need to read to work on this, and no more? | `tigwx set -i i7f3a` |
| Do the design and the code still agree? | `tigwx check` (exit status 1 when not) |
| Which lines seem to be about which files, in code nobody connected? | `tigwx reverse` |
| What is waiting for a decision? | `tigwx status` |

**It can keep these apart, which Git and a document alone cannot:**

- A person's decision and an agent's guess. An agent's sentence waits in a separate inferred document until someone agrees, and over MCP an agent can only propose.
- Whether a line is right, and whether this code goes with it. They are two questions, each with its own answer.
- An edit someone meant, and an edit nobody explained. Only the intents an edit reached need another look.
- Code that shares a file, and intents that have something to do with each other. You can say so: `depends_on`, `parallel`, or your own word.

**It can do this for you:**

- **Serve your project to any agent.** `tigwx mcp` gives Claude Code, Cursor or any MCP client the design, the footprint, the closure check, and tools to record new work.
- **Keep a faithful footprint.** Each work records the design it was for, the project before and after, and every step: each edit with exactly the files it changed, each command, check and decision. A second agent given only the documents and the footprint can build the same thing again, and tig's own tests do exactly that.
- **Connect a project that already has code,** with no model: `tigwx reverse` reads what the lines point at and proposes it, and you decide.
- **Merge without a server.** Records never change and are named by their hash, so two machines merge by putting their files together. When they decided differently, tig says so instead of quietly picking one.
- **Publish the design without the code.** `tigwx push` sends the documents and the chain, and never the code, to a hub people can read.
- **Stay out of your way.** Everything lives in `.tig`, in plain text, in Git with your project. `init` asks nothing and changes no document. tig runs nothing.

**What it does not do.** It runs no checks and passes no verdicts, and a check an agent records is only what the agent said. Closed means the documents, the ledger and the code agree with each other, not that the code is good. Search is keyword search. `by` is what the writer claimed, and nothing is signed, so tig tells you what was said, and your Git host tells you who could have said it.

## Install

Get the setup program for your system from [Releases](https://github.com/h94u041/tig/releases) and run it; it needs nothing else. On Windows it goes to `%LOCALAPPDATA%\Programs\tigwx`, joins your PATH and shows up in your list of installed programs. Elsewhere it goes to `~/.local/bin` and tells you the line to add if that is not on your PATH. `-dir`, `-yes`, `-portable` and `-uninstall` do what they say. Archives holding only the program, and `SHA256SUMS`, are in the same place. The setup program is not signed yet, so Windows and macOS may warn you before running it.

Or with Go 1.26 or later:

```sh
go install github.com/h94u041/tig/cmd/tigwx@latest
```

`@latest` is the latest tagged release; this page follows `main`, and the [changelog](CHANGELOG.md) says what is new since. The hub is a separate command: `go install github.com/h94u041/tig/cmd/tighub@latest`.

## Try it in five minutes

Make a folder holding two files. Your intent IDs will differ: they come from a hash.

`README.md`:

```markdown
# Shop

Carts are kept for a day. Code: [cart](src/cart.go).

Prices come from the table.
```

`src/cart.go`:

```go
package shop

const keepFor = "24h"
```

**Start, and say what the lines mean.** `init` asks nothing and changes no document. `mark` writes a small mark at the end of the line, invisible in any Markdown viewer, and answers with the intent's ID. `-confirm` says this is what you meant.

```console
$ tigwx init
main document README.md, snapshot d53109490e1f
$ tigwx mark -confirm "Carts are kept for a day."
i90e7  README.md:3    Carts are kept for a day. Code: [cart](src/cart.go).
$ tigwx mark "Prices come from the table."
i70fd  README.md:5    Prices come from the table.
$ cat README.md
# Shop

Carts are kept for a day. Code: [cart](src/cart.go). <tig l="fe6d80"/>

Prices come from the table. <tig l="265d0f"/>
```

**Change your mind.** Edit `README.md` so that it says *a week*, and leave the mark where it is. The intent keeps its identity and gets a new revision, and because nobody said why, someone should look again.

```console
$ tigwx sync
1 documents rewritten, 1 intents touched, 0 moved, 0 old marks converted
touched i90e7: confirm it again
$ tigwx show i90e7
i90e7#2  confirmed (human via direct)
  text   Carts are kept for a week.
  place  README.md:3  i90e7#2@l90b35c:0-26
  before i90e7#1 "Carts are kept for a day."
  sync   review
$ tigwx check
closed: no

stale:
  README.md:3 (Shop): the sentence changed and nobody has confirmed it since: Carts are kept for a week.
```

**Do the work, and leave a footprint.** Agents do this over MCP; here it is by hand. Between `start` and `step`, change `src/cart.go` to `168h`.

```console
$ tigwx work start -m "keep carts for a week"
$ tigwx work step -kind edit -m "how long a cart is kept"
step 0ffcaa93ac0c (edit)
  M src/cart.go
$ tigwx work finish
work 4086b2cc85c5: done, 1 steps
$ tigwx commits
4086   2026-11-06 21:24  done       candidate keep carts for a week
```

**Tie them together, and decide.** This commit follows that line, and the line is what you meant. `-source` says where the decision came from.

```console
$ tigwx mark -id 4086 "Carts are kept for a week."
i90e7  README.md:3  4086  Carts are kept for a week. Code: [cart](src/cart.go).
$ tigwx confirm -m "the shop stays open longer now" -source "ticket 12" i90e7
i90e7#2  confirmed  Carts are kept for a week.
$ tigwx check
closed: yes
```

**Now ask it things.**

```console
$ tigwx log i90e7
i90e7#2  confirmed
2026-11-06 21:24  ev90e795  #1 proposed  (human direct Pat)
  text    Carts are kept for a day.
2026-11-06 21:24  ev1557fd  #1 confirmed  (human direct Pat)
2026-11-06 21:24  evae1096  #2 touched  (human direct Pat)
  text    Carts are kept for a week.
2026-11-06 21:24  evec9302  #2 confirmed  (human direct Pat)
  why     the shop stays open longer now
  source  ticket 12
$ tigwx set -i i90e7#1
pinned   i90e7#1 Carts are kept for a day. (it has moved on: #2, confirmed)
sentence README.md:3 Carts are kept for a week. Code: [cart](src/cart.go). 4086
commit   4086 keep carts for a week
file     src/cart.go
digest   796afdbbd1dd1002f5b7d84f6583f798ce06c6d12fdea6dabffaddecaca13aee
$ tigwx trail src/cart.go
  4086b2cc85c5  done        2026-11-06 21:24  Pat  keep carts for a week
```

**Say what two intents have to do with each other,** when they share code but not meaning, or one needs the other. tig keeps it and takes nothing out of what you ask for.

```console
$ tigwx relate -kind parallel -m "nothing in common" i90e7 i70fd
i90e7 parallel i70fd: confirmed
```

Add `.tig` to Git with the rest of the project, and it travels with it.

## Using it with an agent

For Claude Code:

```sh
claude mcp add tig -- tigwx mcp
```

Or in any MCP client's configuration:

```json
{ "mcpServers": { "tig": { "command": "tigwx", "args": ["mcp"] } } }
```

`tigwx mcp` serves the project it is started in (or `-C dir`). The agent reads the design, starts a work for the part it serves, records each step, and proposes which sentences its commit followed; you confirm or reject with `tigwx confirm`, `tigwx reject`, `tigwx link` and `tigwx accept`. [The tools, and what keeps them safe](docs/mcp.md).

For a project that already has code and documents that nothing connects:

```sh
tigwx reverse           # which lines seem to be about which files; it writes nothing
tigwx reverse -write    # propose them; nothing binds until you agree
tigwx init -force       # or say it all at once: the code there is one commit, and every line goes with it
```

## Commands

`tigwx --help` lists every command and its flags. Every command but `mcp` takes `-json`. On the command line, with no `-by` and no `user.by`, a terminal records `human` and anything else records `unknown`: a script should say who it speaks for.

| Command | What it does |
|---|---|
| `init` | start a tig in this folder (`-root` the main document); it asks nothing and records no commit. `-force`: the code there is one commit and every line goes with it |
| `status` | where this project stands, and what is waiting for a decision |
| `config` | read or set your name, your `by`, the main document and the paths outside the chain |
| `snap`, `log`, `diff`, `restore` | snapshots: take one, list them, compare, put files back. `log <intent>`: everything that happened to an intent |
| `blocks <file.md>` | a document's text blocks and their hashes |
| `chain` | every block of the chain |
| `show <block or reference>` | one block, or what a reference means: `i7f3a`, `i7f3a#2`, `lc3d4e5:5-14`, `a1b2`, `ev91c0` |
| `search <words>` | blocks, works, steps and files containing every word |
| `check` | the closure report |
| `trail <block or file>` | the works done for it; `-steps` for each step and the files it changed |
| `work start`, `work step`, `work finish` | leave a footprint by hand |
| `commits`, `commit <id>` | the commits, newest first; one commit with its files and sentences |
| `mark`, `sync`, `unmark` | say what a line means; bring the ledger up to edited lines (`-m why`: the change was meant); withdraw undone commits |
| `reverse`, `describe`, `link` | find which lines are about which files, with no model; say a line is about files already there; decide whether code goes with an intent |
| `relate` | what two intents have to do with each other: `depends_on`, `part_of`, `parallel`, or your own word |
| `set`, `groups` | what a piece of work needs; what belongs together |
| `confirm`, `reject`, `retire`, `accept` | record a decision on an intent or a commit |
| `mcp` | serve this tig over MCP on stdin and stdout |
| `push [-token t] [-m note] [-insecure] <url>` | send the documents and the chain to a hub |

A block is named by its content hash; any unique prefix of 8 or more characters works. Go programs can use tig directly: `import "github.com/h94u041/tig"`. The code is in [cmd/tigwx](cmd/tigwx/) and [cmd/tigwx-setup](cmd/tigwx-setup/); releases are built by [scripts](scripts/) and [.github](.github/).

## Where its potential lies

Code is becoming cheap; knowing what it is for, and that it holds, is not. That is where tig is aimed:

- **A contract between people and agents.** Confirmed intents are what an agent may build from; its own guesses stay marked as guesses. "Is every line of this change agreed with?" can gate a merge the way "do the tests pass?" does today.
- **Memory that outlives a session.** The footprint is shared state for every agent and person on a project: what was done, for which intent, by whom, and whether anyone accepted it. The next agent starts where the last one stopped, not from zero.
- **Traceability without bureaucracy.** Safety-critical and regulated fields demand that every requirement trace to its implementation and its test evidence, and usually pay for it with heavyweight tools. tig gives the same trace in plain text, in Git, kept up to date as a side effect of the work.
- **Reviewing design, not diffs.** Because every intent has an identity that survives edits, a change can be reviewed as "this meaning changed, and these commits followed it, or did not."
- **Publishing what a project promises, without its code.** Open specifications for closed implementations, course material where each student's agent builds against the same scenarios, and audits.

## How it works

Everything lives in the project's `.tig` folder: plain, content-addressed records that go into Git with the project. A tig is portable: hand someone the documents and the footprint, and their agent can build the same thing again.

- **[Footprint](docs/footprint.md).** Snapshots of the project as content-addressed trees, the works done on it, and what keeps keys out of them.
- **[Chain](docs/chain.md).** Text blocks named by content hash, line marks, intents and references, the relations between intents, the chain from the main document to the files built for it, and the closure rules.
- **[MCP](docs/mcp.md).** The tools and resources `tigwx mcp` gives agents, how an agent works with them, and what keeps them safe.
- **[Hub](docs/hub.md).** `tigwx push` sends a project's documents and chain, and nothing else, to a hub, `tighub`, where people read them as they read a repository on a Git host.
- **[Format](docs/format.md).** The `.tig` folder holds plain text objects and records, versioned, so other tools can read and write it too. Every release reads what earlier ones wrote, and the old read what new ones write.
- **[Building on tig](docs/tools.md).** Using it from your own tool, as a command, in Go, or through the files.

**How far it goes.** Measured on Windows: with 5,000 intents, `check` takes under a second and marking one more line about half a second. On a project of 12,000 files (the Go standard library), the first snapshot takes under 20 seconds, and then `check` takes about 0.3 seconds and `status` about 3. On Windows, reading files just written is slower while antivirus looks at them. `.tig` stays about the size of the project on disk, and Git compresses it well. `TIG_SCALE=2000 go test -run TestScale -v .` ([scale_test.go](scale_test.go)) measures it on yours.

## Security

tig runs nothing, keeps paths inside the project, checks records against their hashes, and keeps files that look like keys out of its snapshots. It is also clear about what it cannot do: read [SECURITY.md](SECURITY.md) before relying on it, or to report a problem. **Please upgrade from v0.1.0**: it says why.

## Contributing

Issues and pull requests are welcome. [CONTRIBUTING.md](CONTRIBUTING.md) says what fits and how. What changed between versions is in the [changelog](CHANGELOG.md).

This repository is its own closed chain: its `.tig` holds the footprint of how it was built, and its acceptance sections list the commands to run by hand.

This English README is the source. The translations follow it and are kept out of the chain (`outside` in `.tig/config`); where they differ, this one holds.

## License

MIT, see [LICENSE](LICENSE). Written in Go ([go.mod](go.mod), [go.sum](go.sum)).

## Acceptance

- `go vet ./...`
- `go test ./...`
- `sh scripts/compat.sh <new tigwx> <tigwx 0.1.0>`
