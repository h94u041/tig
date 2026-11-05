# The .tig format

Everything tig writes is plain text or a file's own bytes, named by its content, written once and never changed; only the heads move. Any tool can read and write it.

## Layout

```text
.tig/
  config              tig-config 1
  HEAD                the latest snapshot
  WORKS               the latest work
  LINKS               the latest proposed link
  ledger/<hash>       the ledger: intents, lines, links, acceptances
  objects/ab/ab…      objects
  .gitignore          keeps the local files below out of Git
  .gitattributes      * -text: Git must not change the objects' line endings
  WORK                the work in progress (local)
  cache/              file sizes and times, the other sessions' work in progress (local)
  lock, work.lock     (local)
```

A head holds one hash. The config, the heads, the objects and the ledger go into Git with the project, byte for byte: an object whose line endings Git changed would no longer match its name, and readers refuse it.

## Objects

An object is stored at `objects/`, the first two characters of its hash, `/`, and the whole hash: its bytes as they are. The hash is the SHA-256 of those bytes, in lowercase hexadecimal. Readers check that the bytes match the name. Readers also accept an object compressed with zlib at the same path plus `.z`, as older stores wrote them.

## Records

A record is a header line naming its kind and version, then one `key value` line per field. Values are one line; a key may repeat, and the order is kept. Times are RFC 3339 in UTC. Hashes name objects.

| Header | Fields |
|---|---|
| `tig-config 1` | `root` (the main document), `inferred` (the inferred document), `outside` (repeated) |
| `tig-snapshot 1` | `tree`, `parent` (repeated), `at`, `reason`, `actor`, `lineage` |
| `tig-work 1` | `parent` (the work before), `basis` (repeated blocks), `intent`, `before` and `after` (snapshots), `step` (repeated, in order), `child` (repeated: the commits it folds), `result`, `actor`, `approver` (repeated), `started`, `at` |
| `tig-step 1` | `kind`, `basis` (repeated blocks), `before` and `after` (trees), `command`, `exit`, `output` and `detail` (objects), `note`, `runner`, `actor`, `at` |
| `tig-lineage 1` | `m` (`old new kind score`, repeated), `a` (added blocks), `r` (removed blocks) |
| `tig-link 1` | `parent` (the link before), `from`, `to-block` or `to-file`, `to-hash` (the file when linked), `kind`, `note`, `actor`, `at` |

Readers ignore keys they do not know. A later feature adds fields starting with `x-`, and readers keep them; a new version of a kind is only for a change older readers would misread.

## The ledger

Ledger records are named by their content like everything else, kept in `ledger/` and never changed. There is no head: a machine's ledger is the set of its files, so two ledgers merge by taking both, and a record that does not match its name is refused.

| Header | Fields |
|---|---|
| `tig-intent-event 1` | `intent` (the first event's hash; none on the first), `parent` (the event before), `kind`, `revision`, `text`, `span` (`line n from to`, repeated), `from` (repeated: where it came from), `by`, `via`, `actor`, `reason`, `at` |
| `tig-line-event 1` | `line` (the text's hash), `parent` (the line it was), `at` |
| `tig-link-event 1` | `work`, `intent`, `revision`, `kind`, `actor`, `at` |
| `tig-acceptance 1` | `work`, `state` (`accepted` or `rejected`), `by`, `via`, `actor`, `reason`, `at` |
| `tig-ledger-start 1` | `at`: commits older than this count as accepted |

## Trees

```text
tig-tree 1
d <hash> <name>
f <hash> <size> <name>
x <hash> <size> <name>
```

One line per entry, sorted by name: `d` is a folder (its tree), `f` a file, `x` an executable file. A name is never empty, `.` or `..`, and never holds `/`, `\`, a line break or NUL. Readers also accept the header `tealprint-tree 1`.

## Block hashes

A text block's hash is the SHA-256 of

```text
"tig-block 1" NUL path NUL headings NUL text
```

where `path` is the document's path in the project with `/` between folders, `headings` are the titles from the top of the document down to the block joined by U+001F, and `text` is the block's own text without its heading line, trailing spaces and blank lines at either end. The second and later of identical blocks in one document add NUL and their number (1, 2 …).

## Acceptance

The tests of objects, and of reading older trees and unknown fields, in [tig_test.go](../tig_test.go), and of the ledger in [ledger_test.go](../ledger_test.go):

- `go test -run "Objects|OldTrees|Ledger" .`
