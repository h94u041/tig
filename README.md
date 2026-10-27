# tig

**Markdown as source.** tig makes a project's Markdown documents the source of truth for its code, hosts them as an MCP server that any AI agent can build from, and keeps a faithful footprint of every construction, so the work can be checked, controlled and reproduced.

[![Go Reference](https://pkg.go.dev/badge/github.com/h94u041/tig.svg)](https://pkg.go.dev/github.com/h94u041/tig)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT)

**English** · [繁體中文](i18n/README.zh-TW.md) · [Deutsch](i18n/README.de.md) · [Français](i18n/README.fr.md) · [Русский](i18n/README.ru.md) · [日本語](i18n/README.ja.md)

> This tig is not [tig, the text-mode interface for Git](https://github.com/jonas/tig). That is why its command is called `tigprint`.

## Why tig

Agents now write code faster than anyone can read it. What gets lost is not the code but everything around it:

- **Intent.** Git records which lines changed, not which part of the design they were for. Six months later nobody can say why a file looks the way it does.
- **Evidence.** A design document says what should be true; nothing says whether it still is. Docs drift from code silently, and tests check the code, not whether every part of the design is tested at all.
- **Handoff.** Every agent session starts cold. It re-reads the project, guesses which documents are current, and repeats work another agent already did, or undoes it.

tig closes these gaps with one idea: the design is written as Markdown, every part of it says how it is accepted, and every piece of work is recorded against the part of the design it serves.

## What is tig

- **You write the design** as ordinary Markdown: a main document that links to detail documents, and in each part an *Acceptance* section listing the commands that prove it works.
- **Agents build from it** over MCP. They read and follow the design, see how it was built before, and record every step of their own work against the part of the design it serves.
- **tig checks the result.** It tells you whether every piece of design is covered by an acceptance (the chain is *closed*), whether it passed, and where code and design have drifted apart.

Everything lives in the project's `.tig` folder: plain, content-addressed records that go into Git with the project. A tig is portable: hand someone the documents and the footprint, and their agent can build the same thing again.

## Where its potential lies

Code is becoming cheap; knowing what it is for, and that it holds, is not. That is where tig is aimed:

- **A contract between people and agents.** A closed chain is a specification an agent can execute and a person can verify: directions say what to build, guidelines what was built and holds. "Is the chain closed?" can gate a merge the way "do the tests pass?" does today.
- **Memory that outlives a session.** The footprint is shared state for every agent and person on a project: what was done, for which design, by whom, and whether it passed. The next agent starts where the last one stopped, not from zero.
- **Traceability without bureaucracy.** Safety-critical and regulated fields demand that every requirement trace to its implementation and its test evidence, and usually pay for it with heavyweight tools. tig gives the same trace in plain text, in Git, kept up to date as a side effect of the work.
- **Reviewing design, not diffs.** Because every block of design is named by its content and followed through edits, a change can be reviewed as "this part of the design changed, and these files followed it, or did not."
- **Publishing what a project promises and proves, without its code.** With the hub, a team can show its design and which parts are verified while the implementation stays private: open specifications for closed implementations, course material, and audits.

What tig does not do: an acceptance is only as strong as its commands, and in `loose` mode tig believes the checks agents report (use `strict` when that matters). Search is keyword search. tig shows where design and code disagree; deciding which one is right stays with you.

## Features

- **An MCP server for your project.** `tigprint mcp` serves the documents and the footprint to any MCP client (Claude Code, Cursor and others): an overview, keyword search, reading and following blocks of design, the footprint of each block or file, the closure check, and tools to record new work.
- **Documents as one chain.** The main document, its detail documents and the files they link to are read as one whole. Every text block is named by its content hash, and tig follows each block through edits, moves, splits and merges by fixed, recomputable rules.
- **A faithful footprint.** Each *work* (one construction, like a commit) records the design it was for, its intent, the project before and after, and its *steps*: every edit with exactly the files it changed, every command, check and decision.
- **Agent-level reproduction.** The footprint is complete enough that a second agent, given only the documents and the footprint, can rebuild what the first one built. tig's own tests do exactly that.
- **Closure checks.** Broken links, design no acceptance covers, code changed outside the footprint or left behind by a design change, and files no design reaches. Covered design is a *direction* until its acceptance passes, then a *guideline*.
- **Control over what runs.** tig runs only the commands written in acceptance sections, and over MCP only when you start the server with `-run`. In `strict` mode only the checks tig ran itself count; an agent cannot simply report success.
- **A hub to publish to.** `tigprint push` sends the documents and the chain, never the code, to a `tighub`, which checks every push and shows each document block by block with its status.
- **Safe and portable.** Objects are verified against their hashes on every read, paths from other people's trees cannot escape the project, and concurrent writers never lose a record. The format is plain text, versioned and meant to last.

## Use cases

- **Education and assignments.** Host a closed design as an MCP server and hand it out. Every student's agent builds against the same scenarios, and each submission is a footprint whose acceptance can be checked, by tig itself in strict mode.
- **Tracking and controlling agent work.** Every agent action is recorded against the design it was meant to serve: what was done, by whom, why, and whether it passed.
- **Multi-agent coworking.** Agents share one design and one footprint, so each sees what the others built and picks up where they left off.
- **Design that stays true.** When the design changes and the code does not follow, or the code changes outside any recorded work, tig says so.
- **Showing the design without the code.** Push to a hub so reviewers, clients or students can read the design and see what is verified, without access to the source.

## Install

With Go 1.25 or later:

```sh
go install github.com/h94u041/tig/cmd/tigprint@latest
```

The hub is a separate command: `go install github.com/h94u041/tig/cmd/tighub@latest`.

## Quick start

In a project whose main document is `README.md`:

```sh
tigprint init              # start .tig here
tigprint chain             # every text block of the chain and its status
tigprint check             # is the chain closed? (exit status 1 when not)
tigprint check -run-all    # have tig run every acceptance section itself
```

A section titled `Acceptance` lists the commands that accept the part of the document it sits in:

````markdown
## Turning

How a layer of the cube turns. Code: [turn.js](src/turn.js).

### Acceptance

- `npm test -- turn`
````

Then connect your agents. For Claude Code:

```sh
claude mcp add tig -- tigprint mcp
```

Or in any MCP client's configuration:

```json
{ "mcpServers": { "tig": { "command": "tigprint", "args": ["mcp"] } } }
```

`tigprint mcp` serves the project it is started in (or `-C dir`). Add `-run` to let agents have tig run acceptance commands.

## How it works

tig has three parts, and a hub to publish to, each with its own document.

### Footprint

Snapshots of the project as content-addressed trees, and the works done on it, each made of steps. See [the footprint](docs/footprint.md).

### Chain

Text blocks named by content hash, their lineage from snapshot to snapshot, the chain from the main document to the files built for it, and the closure rules. See [the chain](docs/chain.md).

### MCP

The tools and resources `tigprint mcp` gives agents, how an agent works with them, and what keeps them safe. See [the MCP server](docs/mcp.md).

### Hub

`tigprint push` sends a project's documents and chain, and nothing else, to a hub, `tighub`, where people read them as they read a repository on a Git host. See [the hub](docs/hub.md).

## Commands

`tigprint` alone lists every command and its flags. The code is in [cmd/tigprint](cmd/tigprint/).

| Command | What it does |
|---|---|
| `init` | start a tig in this folder (`-root` the main document, `-strict` to count only checks tig runs) |
| `snap`, `log`, `diff`, `restore` | snapshots: take one, list them, compare, put files back |
| `blocks <file.md>` | a document's text blocks and their hashes |
| `chain` | every block of the chain and its status |
| `show <block>` | one block: its text, status, details, what was built for it and its works |
| `search <words>` | blocks, works, steps and files containing every word |
| `check` | the closure report; `-run <block>` or `-run-all` to run acceptance first; `-json` |
| `trail <block or file>` | the works done for it; `-steps` for each step and the files it changed |
| `work start`, `work step`, `work finish` | leave a footprint by hand |
| `mcp` | serve this tig over MCP on stdin and stdout |
| `push [-token t] [-m note] <url>` | send the documents and the chain to a hub |

A block is named by its content hash; any unique prefix of 8 or more characters works. Go programs can use tig directly: `import "github.com/h94u041/tig"`.

## Format

The `.tig` folder holds plain text objects and records, versioned, so other tools can read and write it too. See [the format](docs/format.md).

## Status

First release, 0.1. The `.tig` format is versioned and meant to last; the commands and the MCP tools may still change. This repository is its own closed chain: its `.tig` holds the footprint of how it was built, and every acceptance section passes when tig runs it.

This English README is the source. The translations follow it and are kept out of the chain (`outside` in `.tig/config`); where they differ, this one holds.

## License

MIT, see [LICENSE](LICENSE). Written in Go ([go.mod](go.mod), [go.sum](go.sum)).

## Acceptance

- `go vet ./...`
- `go test ./...`
