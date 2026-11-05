# Chain

How tig reads a project's documents as one chain, what a line means, and when the chain is closed.

## Text blocks

tig splits a Markdown document at its headings (`#` to `######`; headings inside code fences do not count). A block is a heading and its own text, up to the next heading. The text before the first heading is the document's own block. Code: [markdown.go](../markdown.go).

A block is named by a hash of where it sits (the file and the headings above it) and what it says (its own text). Editing a block, or a heading above it, makes a new block; a child's text is not part of its parent's name, so editing a child leaves the parent as it was. Anywhere tig takes a block, a unique prefix of 8 or more characters of its hash works.

## Line marks

A mark binds a physical line of a document to the ledger. It sits at the end of the line, and any Markdown viewer shows nothing for it:

````markdown
The cube turns one layer at a time. <tig l="c3d4e5"/>
````

`l` is the start of the hash of the line's text, at least 6 characters. `n="2"` tells the second line with the same text from the first. Attributes tig does not know are ignored. In a table the mark goes inside the last cell, and before a hard line break on a line that ends with one. Lines in code fences are not tracked, and a mark quoted as inline code is not a mark. Code: [lines.go](../lines.go).

The text a line is hashed by is the line without its mark, trailing spaces and block prefix (indentation, `#`, `>`, list marker, checkbox), so changing a heading's level or a list's marker does not change the meaning. Everything else stays as written, `**` included. The hash is of the UTF-8 bytes, and the text is kept as an object under that hash.

Marks are not design: a block's text, and so its hash, holds the plain line.

## Intents

An intent is what a line, or a part of one, means. It has an identity that outlives its text: `i` and the start of the hash of its first event, at least 4 characters. It holds one or more ranges, each a line hash, which occurrence, and a start and end in characters (code points, from 0, the end not included), and any number of intents may cover the same characters. Its state is worked out from its events:

| Event | What it says |
|---|---|
| `proposed` | someone said this is what it means |
| `confirmed` | a person, or an AI authorised to, agreed |
| `revised`, `touched` | its text changed: a new revision. `touched` means an edit of its line reached its range, so a confirmed intent wants another look |
| `spans` | its text moved and did not change: no new revision |
| `split`, `composed`, `superseded` | made out of others, or replaced by another |
| `deprecated`, `rejected` | it no longer stands, or never did |

Each event records `by` (`human` or `ai`) and `via` (`direct`, `smooth`, `analysis`, `builder`, `planner`, `migrated`), so an AI's decision can always be told from a person's. Code: [ledger.go](../ledger.go).

Commits link to an intent at a revision, as `implements`, `fixes`, `refactors`, `tests`, `reverts` or `unlinked`. A commit is a **candidate** until someone records it accepted or rejected. Commits older than the ledger, the baseline and folded commits count as accepted; a rejected commit's links count for nothing, and undoing its code is a separate act.

## What an edit does

`tigprint sync` (and anything that writes a mark) reads every document. A line whose text is not what its mark holds gets a new hash and a new mark, and the intents on it follow by comparing the old and new text from both ends:

- a range before the change stays;
- a range after it shifts;
- a range that overlaps it, or covers the whole line, is touched: a new revision, and a confirmed intent wants another look.

Two changes in one line count as one region, so an edit may touch more intents than strictly needed, never fewer. Until sync runs the line shows as unsynced. `Analyze` and `check` only read; `sync` writes. Old marks (`**sentence *a1b2***`) are converted by sync: each becomes an intent proposed `via migrated`, not confirmed, because where it came from is not known. Code: [marks.go](../marks.go).

## The inferred document

Sentences nobody has agreed with go in a separate document, `inferred` in `.tig/config`, by default `inferred.md` beside the main document. It is not design: it is outside the chain and not an orphan. An agent's `mark_sentence` lands there unless the main document already says the sentence. Confirming an intent moves its line into the main document; rejecting it takes the line away.

## References

Anything tig records can be named:

| Reference | Means |
|---|---|
| `i7f3a` | an intent, at its current revision |
| `i7f3a#2` | revision 2 of it, which never changes |
| `lc3d4e5:5-14` | characters 5 to 14 of a line's text |
| `lc3d4e5.2` | the second line with that text |
| `i7f3a#2@lc3d4e5:5-14` | revision 2, at that place |
| `a1b2` | a commit |
| `ev91c0` | an event |

Events take `ev` and not a single letter because a commit's hash can start with any hexadecimal digit. `tigprint show <reference>` and the `resolve` tool give the same answer, including where it is now, what followed it, the revision before, and whether it is in sync. Code: [show.go](../show.go).

## Lineage

When a snapshot changes a document, tig pairs the blocks before and after by fixed rules, in this order, and records the result with the snapshot:

1. **edit**: the same headings, the text changed;
2. **move**: the same text, somewhere else;
3. **split** and **merge**: an old block that two or more new, smaller blocks are mostly made of, or the other way round;
4. **rewrite**: both changed, the text still at least half the same (by the Dice similarity of neighbouring characters, which works the same for text with and without spaces);
5. a single new block mostly made of an old one was split from it; an old block mostly found in a new one was merged into it.

The rest was added or removed. There is no model involved: anyone recomputes the same lineage from the same trees. This is how the footprint of a block follows it through edits. Code: [lineage.go](../lineage.go).

## The chain

The chain starts at the main document (`root` in `.tig/config`, `README.md` unless set) and follows every link to another Markdown file in the project: those are its detail documents, and theirs. A link to any other file, or folder, makes it part of the design of the block whose text links it. A Markdown file the chain never reaches is an orphan. Paths listed as `outside` in the config are not part of the chain. Code: [chain.go](../chain.go).

## Closure

`tigprint check`, and the `check` tool over MCP, report:

- **broken**: links to files that do not exist, orphan documents, and marks that name nothing the ledger knows (`broken-mark`);
- **stale**: a file built for a block whose design has changed since while the file did not follow; a built file changed outside the footprint, or gone; a proposed link whose file has changed; a line whose text moved on with no commit following it (`intent-changed`); a file a line's commits built changed outside any commit, or gone (`code-away`); a line still in the old format (`unsynced`); a confirmed intent nobody has looked at since it changed (`review`); a confirmed intent whose line is in no document (`intent-missing`). Code: [links.go](../links.go);
- **no design reaches**: files no block links to and no work built.

The report also lists the commits no line names (**unbound**): work the design has not caught up with. They do not open the chain.

The chain is **closed** when there is none of these. Closed says the documents, the ledger and the code agree with each other. It says nothing of whether the code is good: tig runs no checks and passes no verdicts. A section titled `Acceptance` is ordinary text to tig; this repository uses them to list the commands to run by hand.

## Search

Blocks, works, steps and files containing every word of a query. It is keyword search only: no hit does not mean the project has nothing on it. Code: [search.go](../search.go).

## Acceptance

The tests in [markdown_test.go](../markdown_test.go), [chain_test.go](../chain_test.go), [marks_test.go](../marks_test.go) and [ledger_test.go](../ledger_test.go):

- `go test -run "Blocks|Chain|Stale|Broken|Mark|Line|Ledger|Intent|Ref" .`
