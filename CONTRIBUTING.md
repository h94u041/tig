# Contributing

Thanks for looking. Issues and pull requests are both welcome, and a question counts as an issue. For anything that could be used against people, see [SECURITY.md](SECURITY.md) instead.

## What fits

tig records and does not judge. A change fits when it keeps these true:

- **tig stands alone.** It knows nothing of any tool on top of it. `by` and `via` are free text, and tig's commands and API do not name any particular tool.
- **Everything is reachable three ways.** What can be written has a command and a Go API, each saying its own `by`, `via` and actor. What can be asked has `-json`. The format in [docs/format.md](docs/format.md) can be read with no tig code.
- **tig decides nothing.** It runs no checks, passes no verdicts, and takes nothing out of what you asked for. It can show where things disagree and keep what someone decided.
- **`init` asks nothing,** and writes no commit and no mark.
- **The format lasts.** A new field or a new kind of record must not break a reader that does not know it, and what has been written stays readable. [compat_test.go](compat_test.go) reads a project written by 0.1.0, which is kept as it was.

Before starting something large, open an issue, so nobody spends a week on something that does not fit.

## Working on it

Go 1.26 or later, and Git. Nothing else.

```sh
go vet ./...
go test ./...
```

The tests run the command line and the MCP server for real, in temporary folders. `TIG_SCALE=2000 go test -run TestScale -v .` measures how long things take as a project grows.

- **Bug fixes come with a test** that fails without the fix.
- **A change to what tig does changes the documents too.** This repository is its own chain, so [docs/](docs/) says what the code does; say it there in the same pull request.
- **English is the source.** A change to `README.md` that changes meaning should touch [i18n/](i18n/) as well. If you can't write one of the languages, say so, and it will be done before the release.
- **Keep the chain closed.** `tigwx check` at the root should say `closed: yes`. Record your change with `tigwx work start`, `tigwx work step -kind edit` and `tigwx work finish`, and link any new file from the document that designs it. The maintainer can do this when merging if you would rather not.
- **`CHANGELOG.md`** gets a line under `Unreleased` for anything a user would notice.

## Pull requests

`main` takes changes through pull requests, reviewed and with the checks passing. Keep one change to one pull request, and say why it is needed as well as what it does. [The template](.github/PULL_REQUEST_TEMPLATE.md) asks for what a review needs.
