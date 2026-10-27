# Hub

A hub keeps projects' documents and chains where people can read them, the way a Git host keeps repositories. `tigprint push` sends a project to a hub; the hub, `tighub`, takes only what the chain is made of and shows it. Code: [hub](../hub/), [cmd/tighub](../cmd/tighub/).

## What is pushed

A push carries the documents and the chain, and nothing else:

- the **documents**: the main document, every detail document the chain reaches, and `README.md` when there is one, as they were in the latest snapshot;
- the **chain**: every block of those documents with its status (guideline, direction or open), the acceptance section that covers it, the detail documents it hands on to, and the paths of the files it links to and the footprint built for it;
- the **closure report**, as `tigprint check` gives it.

The project's other files are never sent: the hub knows their paths, not their contents. Neither are the snapshots, works and steps of the footprint; the chain carries what the footprint concluded. The documents come from the latest snapshot, so `tigprint push` says so when documents changed after it.

## Publications

A push adds a publication: a record that names the documents and the chain, and the publication before it.

```text
tig-publication 1
docs <tree>          the documents, a tree that holds Markdown files only
chain <object>       the chain, as JSON
snapshot <hash>      the snapshot it was read from (not sent)
parent <hash>        the publication before
at <time>
actor <name>
note <text>
```

The tree and the objects are those of [the format](format.md): content-addressed and checked against their names. A project's publications on a hub form one line, newest at its head, as commits do on a branch.

## Pushing

`tigprint push <url>` takes the hub's address of the project, `http://host/owner/name`, or `$TIG_HUB`; the token is `-token` or `$TIG_TOKEN`. Like `git push`:

1. it asks the hub for the project's head;
2. it builds the publication on top of that head, and stops when the documents and the chain are what the head already has;
3. it asks which objects the hub lacks and sends only those, with the head it built on;
4. the hub moves its head only if it still is the one the push built on, so two pushes never overwrite each other.

## What the hub accepts

The hub checks a push before it keeps any of it, and refuses it whole when:

- an object does not match its hash, or the push is too large;
- the documents tree holds anything but Markdown files, or a folder of them;
- the chain names a document the tree does not have, or a block the documents, split again by the hub, do not have;
- the publication's parent is not the project's head;
- the token is not the owner's.

A project is made by its owner's first push. Tokens are given to the hub in a file, one `owner token` per line.

## Reading

Anyone can read what a hub keeps:

| Address | What it shows |
|---|---|
| `/` | the projects |
| `/owner/name` | the project: whether its chain is closed, the counts, its documents and the closure report |
| `/owner/name/doc/<path>` | one document, block by block, each with its status and what it links to and was built |
| `/owner/name/log` | the publications, newest first |
| `/api/v1/projects/owner/name` | the head and the chain, as JSON |

Documents are rendered from Markdown without their raw HTML. A link to another document of the chain stays a link; a link to a file the hub does not have becomes its name.

## Acceptance

The tests in [hub_test.go](../hub/hub_test.go): a push of a project to a hub in the test, what the hub refuses, and its pages.

- `go test ./hub/`
