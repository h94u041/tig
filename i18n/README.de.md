# tig

**Markdown als Quelle.** tig ist ein Hauptbuch für Absichten und Spuren: Es macht die Markdown-Dokumente eines Projekts zur maßgeblichen Quelle für seinen Code, stellt sie als MCP-Server bereit, nach dem jeder KI-Agent bauen kann, und hält fest, was jede Zeile bedeutet, wer ihr zugestimmt hat und welche Arbeit ihr gefolgt ist.

[![Go Reference](https://pkg.go.dev/badge/github.com/h94u041/tig.svg)](https://pkg.go.dev/github.com/h94u041/tig)
[![test](https://github.com/h94u041/tig/actions/workflows/test.yml/badge.svg)](https://github.com/h94u041/tig/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT)

[English](../README.md) · [繁體中文](README.zh-TW.md) · **Deutsch** · [Français](README.fr.md) · [Русский](README.ru.md) · [日本語](README.ja.md)

> Dieses tig ist nicht [tig, die Textoberfläche für Git](https://github.com/jonas/tig). Deshalb heißt der Befehl `tigwx`.

## Warum tig

Agenten schreiben Code heute schneller, als ihn jemand lesen kann. Verloren geht nicht der Code, sondern alles darum herum:

- **Absicht.** Git hält fest, welche Zeilen sich geändert haben, nicht für welchen Teil des Entwurfs. Ein halbes Jahr später kann niemand sagen, warum eine Datei so aussieht.
- **Zustimmung.** Ein Entwurf sagt, was gelten soll, aber nicht, wer das gesagt hat: ein Mensch oder ein ratender Agent. Dokumente entfernen sich still vom Code, und die Änderung eines Satzes sieht genauso aus wie ein Sinneswandel über alles drumherum.
- **Übergabe.** Jede Agentensitzung beginnt bei null. Sie liest das Projekt neu, rät, welche Dokumente aktuell sind, und wiederholt die Arbeit eines anderen Agenten oder macht sie rückgängig.

tig schließt diese Lücken mit einer Idee: Der Entwurf wird in Markdown geschrieben, jede Zeile trägt eine kleine Markierung, die zu ihrer Bedeutung führt, und jede Arbeit wird gegen diese Bedeutung festgehalten. tig zeichnet auf und urteilt nicht. Ob eine Arbeit gut genug ist, entscheiden Sie oder das Werkzeug darüber, und tig bewahrt die Entscheidung.

## Was tig kann

**Es beantwortet diese Fragen, auf der Kommandozeile, als JSON oder einem Agenten über MCP:**

| Sie möchten wissen | Fragen Sie |
|---|---|
| Was bedeutet diese Zeile, wer sagt das, hat jemand zugestimmt? | `tigwx show i7f3a` |
| Was ist mit dieser Absicht geschehen, und warum änderte sie sich? | `tigwx log i7f3a` |
| Was sagte diese Anforderung, als danach gebaut wurde? | `tigwx set -i i7f3a#1` |
| Welche Arbeiten änderten diese Datei, und wofür? | `tigwx trail src/cart.go` |
| Was muss ich lesen, um daran zu arbeiten, und nicht mehr? | `tigwx set -i i7f3a` |
| Stimmen Entwurf und Code noch überein? | `tigwx check` (Exit-Status 1, wenn nicht) |
| Welche Zeilen scheinen welche Dateien zu betreffen, in Code, den niemand verbunden hat? | `tigwx reverse` |
| Was wartet auf eine Entscheidung? | `tigwx status` |

**Es hält auseinander, was Git und ein Dokument allein nicht können:**

- Die Entscheidung eines Menschen und die Vermutung eines Agenten. Der Satz eines Agenten wartet in einem eigenen abgeleiteten Dokument, bis jemand zustimmt, und über MCP kann ein Agent nur vorschlagen.
- Ob eine Zeile stimmt und ob dieser Code zu ihr gehört. Das sind zwei Fragen mit je eigener Antwort.
- Eine gewollte Änderung und eine, die niemand erklärt hat. Nur die Absichten, die eine Änderung erreicht hat, brauchen einen neuen Blick.
- Code, der sich eine Datei teilt, und Absichten, die wirklich zusammenhängen. Das können Sie sagen: `depends_on`, `parallel` oder Ihr eigenes Wort.

**Es erledigt das für Sie:**

- **Ihr Projekt jedem Agenten bereitstellen.** `tigwx mcp` gibt Claude Code, Cursor oder jedem MCP-Client den Entwurf, die Spur, die Abschlussprüfung und Werkzeuge, um neue Arbeit festzuhalten.
- **Eine getreue Spur führen.** Jede Arbeit hält den Entwurf fest, dem sie diente, das Projekt davor und danach und jeden Schritt: jede Änderung mit genau den betroffenen Dateien, jeden Befehl, jede Prüfung und Entscheidung. Ein zweiter Agent, der nur die Dokumente und die Spur erhält, kann dasselbe erneut bauen, und tigs eigene Tests tun genau das.
- **Ein Projekt mit vorhandenem Code verbinden,** ohne Modell: `tigwx reverse` liest, worauf die Zeilen zeigen, und schlägt es vor, und Sie entscheiden.
- **Ohne Server zusammenführen.** Datensätze ändern sich nie und heißen nach ihrem Hash, also führen zwei Rechner zusammen, indem sie ihre Dateien vereinen. Haben sie unterschiedlich entschieden, sagt tig das, statt still eines zu wählen.
- **Den Entwurf ohne den Code veröffentlichen.** `tigwx push` sendet die Dokumente und die Kette, nie den Code, an einen Hub, den andere lesen können.
- **Nicht im Weg stehen.** Alles liegt in `.tig`, als Klartext, mit Ihrem Projekt in Git. `init` fragt nichts und ändert kein Dokument. tig führt nichts aus.

**Was es nicht tut.** Es führt keine Prüfungen aus und fällt keine Urteile, und eine von einem Agenten festgehaltene Prüfung ist nur, was der Agent gesagt hat. Geschlossen heißt, dass Dokumente, Hauptbuch und Code übereinstimmen, nicht dass der Code gut ist. Die Suche ist eine Stichwortsuche. `by` ist, was der Schreibende behauptet hat, und nichts ist signiert: tig sagt Ihnen, was gesagt wurde, und Ihr Git-Host, wer es hätte sagen können.

## Installation

Laden Sie das Setup-Programm für Ihr System von [Releases](https://github.com/h94u041/tig/releases) und starten Sie es; mehr braucht es nicht. Unter Windows landet es in `%LOCALAPPDATA%\Programs\tigwx`, kommt in den PATH und erscheint in der Liste der installierten Programme. Anderswo landet es in `~/.local/bin` und nennt Ihnen die nötige Zeile, falls das nicht im PATH liegt. `-dir`, `-yes`, `-portable` und `-uninstall` tun, was sie sagen. Archive nur mit dem Programm und `SHA256SUMS` liegen am selben Ort. Das Setup-Programm ist noch nicht signiert, daher warnen Windows und macOS eventuell vor dem Start.

Oder mit Go 1.26 oder neuer:

```sh
go install github.com/h94u041/tig/cmd/tigwx@latest
```

`@latest` ist die neueste getaggte Version; diese Seite folgt `main`, und das [Änderungsprotokoll](../CHANGELOG.md) nennt, was seither neu ist. Der Hub ist ein eigener Befehl: `go install github.com/h94u041/tig/cmd/tighub@latest`.

## In fünf Minuten ausprobieren

Legen Sie einen Ordner mit zwei Dateien an. Ihre Absichts-IDs werden abweichen: Sie stammen aus einem Hash.

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

**Beginnen und sagen, was die Zeilen bedeuten.** `init` fragt nichts und ändert kein Dokument. `mark` schreibt eine kleine Markierung ans Zeilenende, in jeder Markdown-Ansicht unsichtbar, und antwortet mit der ID der Absicht. `-confirm` sagt, dass es so gemeint ist.

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

**Die Meinung ändern.** Ändern Sie `README.md` zu *a week* und lassen Sie die Markierung stehen. Die Absicht behält ihre Identität und bekommt eine neue Revision, und weil niemand den Grund gesagt hat, sollte jemand noch einmal hinsehen.

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

**Die Arbeit tun und eine Spur hinterlassen.** Agenten tun das über MCP; hier geschieht es von Hand. Ändern Sie zwischen `start` und `step` die Datei `src/cart.go` auf `168h`.

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

**Verbinden und entscheiden.** Dieser Commit folgt jener Zeile, und die Zeile ist so gemeint. `-source` sagt, woher die Entscheidung kam.

```console
$ tigwx mark -id 4086 "Carts are kept for a week."
i90e7  README.md:3  4086  Carts are kept for a week. Code: [cart](src/cart.go).
$ tigwx confirm -m "the shop stays open longer now" -source "ticket 12" i90e7
i90e7#2  confirmed  Carts are kept for a week.
$ tigwx check
closed: yes
```

**Jetzt können Sie es fragen.**

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

**Sagen, was zwei Absichten miteinander zu tun haben,** wenn sie Code teilen, aber keine Bedeutung, oder eine die andere braucht. tig bewahrt das und nimmt nichts aus dem heraus, wonach Sie fragen.

```console
$ tigwx relate -kind parallel -m "nothing in common" i90e7 i70fd
i90e7 parallel i70fd: confirmed
```

Nehmen Sie `.tig` zusammen mit dem Rest des Projekts in Git auf, dann reist es mit.

## Mit einem Agenten verwenden

Für Claude Code:

```sh
claude mcp add tig -- tigwx mcp
```

Oder in der Konfiguration eines beliebigen MCP-Clients:

```json
{ "mcpServers": { "tig": { "command": "tigwx", "args": ["mcp"] } } }
```

`tigwx mcp` stellt das Projekt bereit, in dem es gestartet wird (oder `-C dir`). Der Agent liest den Entwurf, beginnt eine Arbeit für den betreffenden Teil, hält jeden Schritt fest und schlägt vor, welchen Sätzen sein Commit folgte; Sie bestätigen oder lehnen ab mit `tigwx confirm`, `tigwx reject`, `tigwx link` und `tigwx accept`. [Die Werkzeuge und was sie sicher macht](../docs/mcp.md).

Für ein Projekt mit Code und Dokumenten, die nichts verbindet:

```sh
tigwx reverse           # welche Zeilen scheinen welche Dateien zu betreffen; es schreibt nichts
tigwx reverse -write    # sie vorschlagen; nichts gilt, bis Sie zustimmen
tigwx init -force       # oder alles auf einmal: der Code ist ein Commit, und jede Zeile gehört dazu
```

## Befehle

`tigwx --help` listet alle Befehle und Flags. Jeder Befehl außer `mcp` nimmt `-json`. Auf der Kommandozeile hält ein Terminal ohne `-by` und `user.by` `human` fest, alles andere `unknown`: Ein Skript sollte sagen, für wen es spricht.

| Befehl | Was er tut |
|---|---|
| `init` | tig in diesem Ordner beginnen (`-root` das Hauptdokument); fragt nichts und hält keinen Commit fest. `-force`: der vorhandene Code ist ein Commit, und jede Zeile gehört dazu |
| `status` | wo das Projekt steht und was auf eine Entscheidung wartet |
| `config` | Ihren Namen, Ihr `by`, das Hauptdokument und die Pfade außerhalb der Kette lesen oder setzen |
| `snap`, `log`, `diff`, `restore` | Schnappschüsse: anlegen, auflisten, vergleichen, Dateien zurückholen. `log <intent>`: alles, was einer Absicht geschah |
| `blocks <file.md>` | die Textblöcke eines Dokuments und ihre Hashes |
| `chain` | jeder Block der Kette |
| `show <Block oder Referenz>` | ein Block, oder was eine Referenz bedeutet: `i7f3a`, `i7f3a#2`, `lc3d4e5:5-14`, `a1b2`, `ev91c0` |
| `search <Wörter>` | Blöcke, Arbeiten, Schritte und Dateien mit allen Wörtern |
| `check` | der Abschlussbericht |
| `trail <Block oder Datei>` | die Arbeiten dafür; `-steps` für jeden Schritt und seine Dateien |
| `work start`, `work step`, `work finish` | von Hand eine Spur hinterlassen |
| `commits`, `commit <id>` | die Commits, neueste zuerst; ein Commit mit seinen Dateien und Sätzen |
| `mark`, `sync`, `unmark` | sagen, was eine Zeile bedeutet; das Hauptbuch an geänderte Zeilen anpassen (`-m why`: die Änderung war gewollt); rückgängig gemachte Commits zurückziehen |
| `reverse`, `describe`, `link` | ohne Modell finden, welche Zeilen welche Dateien betreffen; sagen, dass eine Zeile vorhandene Dateien betrifft; entscheiden, ob Code zu einer Absicht gehört |
| `relate` | was zwei Absichten miteinander zu tun haben: `depends_on`, `part_of`, `parallel` oder Ihr eigenes Wort |
| `set`, `groups` | was eine Arbeit braucht; was zusammengehört |
| `confirm`, `reject`, `retire`, `accept` | eine Entscheidung zu einer Absicht oder einem Commit festhalten |
| `mcp` | dieses tig über MCP auf stdin und stdout bereitstellen |
| `push [-token t] [-m note] [-insecure] <url>` | Dokumente und Kette an einen Hub senden |

Ein Block heißt nach seinem Inhaltshash; jedes eindeutige Präfix ab 8 Zeichen genügt. Go-Programme können tig direkt nutzen: `import "github.com/h94u041/tig"`. Der Code liegt in [cmd/tigwx](../cmd/tigwx/) und [cmd/tigwx-setup](../cmd/tigwx-setup/); Releases entstehen durch [scripts](../scripts/) und [.github](../.github/).

## Wo das Potenzial liegt

Code wird billig; zu wissen, wofür er da ist und dass er hält, wird es nicht. Darauf zielt tig:

- **Ein Vertrag zwischen Menschen und Agenten.** Bestätigte Absichten sind das, wonach ein Agent bauen darf; seine Vermutungen bleiben als Vermutungen markiert. „Ist jede Zeile dieser Änderung abgestimmt?“ kann einen Merge freigeben wie heute „Laufen die Tests?“.
- **Ein Gedächtnis, das die Sitzung überlebt.** Die Spur ist gemeinsamer Zustand aller Agenten und Menschen eines Projekts: was getan wurde, für welche Absicht, von wem und ob es jemand angenommen hat. Der nächste Agent beginnt dort, wo der letzte aufhörte.
- **Nachvollziehbarkeit ohne Bürokratie.** Sicherheitskritische und regulierte Bereiche verlangen, dass jede Anforderung bis zur Umsetzung und zum Testnachweis nachverfolgbar ist, und zahlen dafür meist mit schweren Werkzeugen. tig gibt dieselbe Spur als Klartext in Git, aktuell gehalten als Nebenprodukt der Arbeit.
- **Entwurf prüfen statt Diffs.** Weil jede Absicht eine Identität hat, die Änderungen überlebt, lässt sich eine Änderung prüfen als „diese Bedeutung änderte sich, und diese Commits folgten ihr oder nicht“.
- **Veröffentlichen, was ein Projekt verspricht, ohne seinen Code.** Offene Spezifikationen für geschlossene Implementierungen, Kursmaterial, bei dem jeder Agent der Studierenden nach denselben Szenarien baut, und Audits.

## Wie es funktioniert

Alles liegt im Ordner `.tig` des Projekts: einfache, inhaltsadressierte Datensätze, die mit dem Projekt in Git gehen. Ein tig ist portabel: Geben Sie jemandem die Dokumente und die Spur, und sein Agent kann dasselbe erneut bauen.

- **[Spur](../docs/footprint.md).** Schnappschüsse als inhaltsadressierte Bäume, die daran getanen Arbeiten und was Schlüssel draußen hält.
- **[Kette](../docs/chain.md).** Textblöcke mit Inhaltshash, Zeilenmarkierungen, Absichten und Referenzen, Beziehungen zwischen Absichten, die Kette vom Hauptdokument bis zu den gebauten Dateien und die Abschlussregeln.
- **[MCP](../docs/mcp.md).** Die Werkzeuge und Ressourcen von `tigwx mcp`, wie ein Agent damit arbeitet und was sie sicher macht.
- **[Hub](../docs/hub.md).** `tigwx push` sendet nur die Dokumente und die Kette an einen Hub, `tighub`, wo man sie wie ein Repository auf einem Git-Host liest.
- **[Format](../docs/format.md).** `.tig` enthält versionierte Klartextobjekte und -datensätze, die auch andere Werkzeuge lesen und schreiben können. Jede Version liest, was frühere schrieben, und alte lesen, was neue schreiben.
- **[Auf tig aufbauen](../docs/tools.md).** Nutzung aus dem eigenen Werkzeug, als Befehl, in Go oder über die Dateien.

**Wie weit es trägt.** Gemessen unter Windows: Bei 5.000 Absichten dauert `check` unter einer Sekunde und das Markieren einer weiteren Zeile etwa eine halbe. Bei einem Projekt mit 12.000 Dateien (der Go-Standardbibliothek) dauert der erste Schnappschuss unter 20 Sekunden, danach `check` etwa 0,3 und `status` etwa 3 Sekunden. Unter Windows ist das Lesen frisch geschriebener Dateien langsamer, solange der Virenschutz sie prüft. `.tig` ist auf der Platte etwa so groß wie das Projekt, und Git komprimiert es gut. `TIG_SCALE=2000 go test -run TestScale -v .` ([scale_test.go](../scale_test.go)) misst es bei Ihnen.

## Sicherheit

tig führt nichts aus, hält Pfade im Projekt, prüft Datensätze gegen ihre Hashes und hält Dateien, die wie Schlüssel aussehen, aus seinen Schnappschüssen heraus. Es sagt auch klar, was es nicht kann: Lesen Sie [SECURITY.md](../SECURITY.md), bevor Sie sich darauf verlassen oder um ein Problem zu melden. **Bitte aktualisieren Sie von v0.1.0:** Dort steht, warum.

## Mitwirken

Issues und Pull Requests sind willkommen. [CONTRIBUTING.md](../CONTRIBUTING.md) sagt, was passt und wie. Was sich zwischen Versionen änderte, steht im [Änderungsprotokoll](../CHANGELOG.md).

Dieses Repository ist seine eigene geschlossene Kette: Sein `.tig` enthält die Spur, wie es gebaut wurde, und seine Abnahmeabschnitte nennen die Befehle, die von Hand auszuführen sind.

Dieses englische README ist die Quelle. Die Übersetzungen folgen ihm und liegen außerhalb der Kette (`outside` in `.tig/config`); bei Abweichungen gilt das englische.

## Lizenz

MIT, siehe [LICENSE](../LICENSE). Geschrieben in Go ([go.mod](../go.mod), [go.sum](../go.sum)).

## Abnahme

- `go vet ./...`
- `go test ./...`
- `sh scripts/compat.sh <neues tigwx> <tigwx 0.1.0>`
