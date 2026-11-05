# Building on tig

tig records and does not judge, so whatever decides, runs or asks sits on top of it: a person at the command line, a CI job, a review board, an editor, an agent harness. Anything on top can use all of tig three ways, and none of them needs any other:

| | How | For |
|---|---|---|
| Command line | `tigwx <command> -json` | scripts, CI, any language |
| Go | `github.com/h94u041/tig` | programs in Go |
| Files | the [format](format.md) | reading or writing `.tig` with no tig code at all |

Agents get a fourth way, the [MCP server](mcp.md), which can propose but not decide.

## Calling it

`tigwx` is called the way `git` is, so what you already do with git carries over.

- It finds the project by walking up from the current folder to the nearest `.tig`. `-C dir` starts from another folder.
- Flags may go before or after the other arguments. `--` ends them: `tigwx search -- -json` searches for the word.
- `--version`, `--help`, `help <command>` and `<command> -h` answer on stdout and end with 0. `help -a` lists every command, and `help -json` does so for a program.
- A mistyped command gets one line and the nearest name, not the whole usage.

| Exit status | Means |
|---|---|
| 0 | done |
| 1 | the answer is no: `check` is not closed, a `config` key is not set, no such command |
| 128 | it could not be done: `fatal: …` on stderr, for example no project here |
| 129 | it was asked for wrongly: usage on stderr |

### Where a project stands

`tigwx status` lists facts and changes nothing: the project and its main document, who you are acting as and where that came from, the last snapshot, the work in progress, what changed since the snapshot, lines waiting for a sync, proposed intents and links, candidate commits, and whether the chain is closed. It always ends with 0 when it can read the project. To branch on the chain, use `check`'s exit status. `status -json` has the same fields.

### Who is acting

```sh
tigwx config -global user.name "Pat Lee"   # every project of yours
tigwx config user.name shopbot             # this project, ahead of the global one
tigwx config user.by ai                    # what -by says when nobody says otherwise
tigwx config -list
tigwx config -unset user.by
```

| | First | Then | Then | Then | Default |
|---|---|---|---|---|---|
| Actor | `-actor` | `$TIG_ACTOR` | project `user.name` | global `user.name` | `cli` |
| By | `-by` | project `user.by` | global `user.by` | | `human` |

The project also keeps `root`, `inferred` and `outside` (comma-separated). They have no global value. The session comes from `-session` or `$TIG_SESSION`.

### Your own commands

Any program on PATH named `tigwx-<name>` runs as `tigwx <name>`, in any language. The name is lowercase letters, digits and hyphens. Built-in commands come first, and a program in the current folder only counts if PATH names that folder absolutely.

The program gets its arguments as given, the working folder from `-C`, and these variables: `TIGWX` (the path of `tigwx`, to call it back), `TIG_ACTOR` when `-actor` was given, and `TIG_SESSION`. Its exit status becomes `tigwx`'s. `tigwx help <name>` runs it with `--help`.

## Releasing

`go run ./scripts/release <version>` builds the setup programs, the archives and `SHA256SUMS` for every system into `dist/`, with Go and nothing else. On GitHub, `.github/workflows/test.yml` runs the tests on Linux, Windows and macOS for every push to `main`, and `.github/workflows/release.yml` publishes a release for every `v*` tag. Tag only after the tests pass: the Go module proxy keeps a tagged version forever.

## Your words, not tig's

Every decision records who made it and how it came about: `by` and `via`. tig gives these no meaning. `by` is `human`, `ai`, `tool` or a word of your own; `via` is free text. Say your own ways of working here and read them back later:

```sh
tigwx accept -by alice -via review-board 7f3a
tigwx mark -inferred -by shopbot -via nightly-scan -id 7f3a "Receipts are printed."
```

The only meaning tig attaches: a link marked as a person's word (`by human`, the default on the command line) starts confirmed, and everything else starts proposed. Pass `-propose` to say otherwise. A tool that wants its own users' names as `by` should also pass `-propose`, and confirm through `tigwx link`.

## The flow

```sh
tigwx init                          # asks nothing, records no commit, leaves the documents alone
tigwx reverse -json                 # which lines seem to be about which files
tigwx reverse -write -via my-tool   # propose them; nothing binds yet
tigwx set -json -s "A sentence."    # what's on a line: intents, links, states
tigwx link -by alice <intent> <id>  # this code goes with this intent
tigwx link -reject <intent> <id>    # it does not, and it won't be proposed again
```

Work is recorded the same way, with whatever does it:

```sh
tigwx work start -m "print receipts"
# ... change files ...
tigwx work step -kind edit -m "prints"
tigwx work finish -result done
tigwx mark -inferred -by my-agent -via my-tool -id <id> "Receipts are printed."
tigwx accept -by alice <id>         # accepting a commit carries the links it proposed
tigwx check -json                   # exit status 1 when the chain is not closed
```

Editing a line and running `tigwx sync` moves what the edit did not reach and asks again about what it did. To say it all at once instead, `tigwx init -force` makes the code there one commit and has every line go with it, proposed.

## Two questions, kept apart

- **Is this what the line means?** The intent's state: `proposed`, `confirmed`, and so on. `confirm`, `reject`, `retire`.
- **Does this code go with it?** The link's state: `proposed`, `confirmed`, or rejected and then gone. `link`, or `accept` on the commit.

A tool often wants to ask them together: when someone agrees with a correspondence, also confirm the intent if nobody has. That is the tool's choice, and tig records both.

## What binds

A commit that changed files stands on a line as soon as something says so, and waits as a candidate. A commit from `describe` or `reverse -write` changed nothing: it is only a reading of the code, so until someone agrees it is not in a sentence's `ids`, does not put files in a group and does not close the chain. Sentences show it in `intents[].links[]`, with its state and reason, whichever it is.

## Reading the answers

`-json` answers are objects and lists with stable names, and new names may appear, so ignore what you don't know. Flags go before other arguments. Only `check` uses exit status 1, for a chain that is not closed; anything that went wrong is 2.
