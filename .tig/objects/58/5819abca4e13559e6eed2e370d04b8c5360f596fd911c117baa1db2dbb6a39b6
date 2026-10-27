# tig

**Markdown als Quelle.** tig macht die Markdown-Dokumente eines Projekts zur maßgeblichen Quelle für seinen Code, stellt sie als MCP-Server bereit, nach dem jeder KI-Agent bauen kann, und hält von jedem Bau eine getreue Spur fest, sodass sich die Arbeit prüfen, steuern und reproduzieren lässt.

[![Go Reference](https://pkg.go.dev/badge/github.com/h94u041/tig.svg)](https://pkg.go.dev/github.com/h94u041/tig)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT)

[English](../README.md) · [繁體中文](README.zh-TW.md) · **Deutsch** · [Français](README.fr.md) · [Русский](README.ru.md) · [日本語](README.ja.md)

> Dies ist eine Übersetzung von [README.md](../README.md). Die englische Fassung ist das Original und die, die tig prüft; bei Abweichungen gilt sie.

> Dieses tig ist nicht [tig, die Textoberfläche für Git](https://github.com/jonas/tig). Deshalb heißt sein Befehl `tigprint`.

## Warum tig

Agenten schreiben Code inzwischen schneller, als irgendjemand ihn lesen kann. Verloren geht dabei nicht der Code, sondern alles um ihn herum:

- **Absicht.** Git hält fest, welche Zeilen sich geändert haben, nicht, für welchen Teil des Entwurfs. Ein halbes Jahr später kann niemand mehr sagen, warum eine Datei so aussieht, wie sie aussieht.
- **Nachweis.** Ein Entwurfsdokument sagt, was gelten soll; nichts sagt, ob es noch gilt. Dokumentation entfernt sich still vom Code, und Tests prüfen den Code, nicht, ob überhaupt jeder Teil des Entwurfs getestet wird.
- **Übergabe.** Jede Agenten-Sitzung beginnt kalt. Sie liest das Projekt neu, rät, welche Dokumente aktuell sind, und wiederholt Arbeit, die ein anderer Agent schon getan hat, oder macht sie rückgängig.

tig schließt diese Lücken mit einer Idee: Der Entwurf wird in Markdown geschrieben, jeder seiner Teile sagt, wie er abgenommen wird, und jede Arbeit wird bei dem Teil des Entwurfs festgehalten, dem sie dient.

## Was ist tig

- **Sie schreiben den Entwurf** in gewöhnlichem Markdown: ein Hauptdokument, das auf Detaildokumente verweist, und in jedem Teil ein Abschnitt *Acceptance* (Abnahme) mit den Befehlen, die belegen, dass er funktioniert.
- **Agenten bauen danach**, über MCP. Sie lesen den Entwurf und folgen ihm, sehen, wie früher gebaut wurde, und halten jeden Schritt ihrer eigenen Arbeit bei dem Teil des Entwurfs fest, dem er dient.
- **tig prüft das Ergebnis.** Es sagt Ihnen, ob jeder Teil des Entwurfs von einer Abnahme abgedeckt ist (die Kette ist *geschlossen*), ob sie bestanden wurde und wo Code und Entwurf auseinandergelaufen sind.

Alles liegt im Ordner `.tig` des Projekts: einfache, inhaltsadressierte Datensätze, die mit dem Projekt in Git landen. Ein tig ist portabel: Geben Sie jemandem die Dokumente und die Spur, und sein Agent kann dasselbe noch einmal bauen.

## Wo das Potenzial liegt

Code wird billig; zu wissen, wofür er da ist und dass er hält, wird es nicht. Genau darauf zielt tig:

- **Ein Vertrag zwischen Menschen und Agenten.** Eine geschlossene Kette ist eine Spezifikation, die ein Agent ausführen und ein Mensch überprüfen kann: Directions sagen, was zu bauen ist, Guidelines, was gebaut wurde und hält. „Ist die Kette geschlossen?“ kann einen Merge freigeben, so wie heute „Laufen die Tests durch?“.
- **Ein Gedächtnis, das eine Sitzung überdauert.** Die Spur ist gemeinsamer Zustand für jeden Agenten und jeden Menschen im Projekt: was getan wurde, für welchen Entwurf, von wem und ob es bestanden hat. Der nächste Agent beginnt dort, wo der letzte aufgehört hat, nicht bei null.
- **Rückverfolgbarkeit ohne Bürokratie.** Sicherheitskritische und regulierte Bereiche verlangen, dass sich jede Anforderung zu ihrer Umsetzung und ihrem Testnachweis zurückverfolgen lässt, und bezahlen das meist mit schwergewichtigen Werkzeugen. tig liefert dieselbe Rückverfolgung als Klartext in Git, aktuell gehalten als Nebenprodukt der Arbeit.
- **Entwürfe prüfen statt Diffs.** Weil jeder Block des Entwurfs nach seinem Inhalt benannt und durch Änderungen hindurch verfolgt wird, lässt sich eine Änderung so prüfen: „Dieser Teil des Entwurfs hat sich geändert, und diese Dateien sind gefolgt, oder eben nicht.“
- **Veröffentlichen, was ein Projekt verspricht und belegt, ohne seinen Code.** Mit dem Hub kann ein Team seinen Entwurf zeigen und welche Teile verifiziert sind, während die Implementierung privat bleibt: offene Spezifikationen für geschlossene Implementierungen, Kursmaterial und Audits.

Was tig nicht tut: Eine Abnahme ist nur so stark wie ihre Befehle, und im Modus `loose` glaubt tig den Prüfungen, die Agenten melden (verwenden Sie `strict`, wenn es darauf ankommt). Die Suche ist eine Stichwortsuche. tig zeigt, wo Entwurf und Code sich widersprechen; welche Seite recht hat, entscheiden weiterhin Sie.

## Funktionen

- **Ein MCP-Server für Ihr Projekt.** `tigprint mcp` stellt die Dokumente und die Spur jedem MCP-Client bereit (Claude Code, Cursor und anderen): eine Übersicht, Stichwortsuche, Lesen und Verfolgen von Entwurfsblöcken, die Spur jedes Blocks und jeder Datei, die Geschlossenheitsprüfung und Werkzeuge, um neue Arbeit festzuhalten.
- **Dokumente als eine Kette.** Das Hauptdokument, seine Detaildokumente und die Dateien, auf die sie verweisen, werden als ein Ganzes gelesen. Jeder Textblock ist nach seinem Inhalts-Hash benannt, und tig verfolgt jeden Block durch Bearbeitungen, Verschiebungen, Aufteilungen und Zusammenführungen nach festen, nachrechenbaren Regeln.
- **Eine getreue Spur.** Jede *Work* (ein Bau, wie ein Commit) hält fest, für welchen Entwurf sie war, ihre Absicht, das Projekt davor und danach und ihre *Steps*: jede Bearbeitung mit genau den Dateien, die sie geändert hat, jeden Befehl, jede Prüfung und jede Entscheidung.
- **Reproduktion auf Agentenebene.** Die Spur ist vollständig genug, dass ein zweiter Agent, der nur die Dokumente und die Spur bekommt, nachbauen kann, was der erste gebaut hat. Die Tests von tig selbst tun genau das.
- **Geschlossenheitsprüfungen.** Defekte Links, Entwurf, den keine Abnahme abdeckt, Code, der außerhalb der Spur geändert oder von einer Entwurfsänderung zurückgelassen wurde, und Dateien, die kein Entwurf erreicht. Abgedeckter Entwurf ist eine *Direction*, bis seine Abnahme besteht, dann eine *Guideline*.
- **Kontrolle darüber, was läuft.** tig führt nur die Befehle aus, die in Abnahmeabschnitten stehen, und über MCP nur, wenn Sie den Server mit `-run` starten. Im Modus `strict` zählen nur die Prüfungen, die tig selbst ausgeführt hat; ein Agent kann nicht einfach Erfolg melden.
- **Ein Hub zum Veröffentlichen.** `tigprint push` sendet die Dokumente und die Kette, niemals den Code, an einen `tighub`, der jeden Push prüft und jedes Dokument Block für Block mit seinem Status zeigt.
- **Sicher und portabel.** Objekte werden bei jedem Lesen gegen ihren Hash geprüft, Pfade aus fremden Bäumen können das Projekt nicht verlassen, und gleichzeitige Schreiber verlieren nie einen Datensatz. Das Format ist Klartext, versioniert und auf Dauer angelegt.

## Anwendungsfälle

- **Lehre und Aufgaben.** Stellen Sie einen geschlossenen Entwurf als MCP-Server bereit und teilen Sie ihn aus. Der Agent jedes Studierenden baut gegen dieselben Szenarien, und jede Abgabe ist eine Spur, deren Abnahme sich prüfen lässt, im Modus `strict` von tig selbst.
- **Agentenarbeit nachverfolgen und steuern.** Jede Aktion eines Agenten wird bei dem Entwurf festgehalten, dem sie dienen sollte: was getan wurde, von wem, warum und ob es bestanden hat.
- **Zusammenarbeit mehrerer Agenten.** Agenten teilen einen Entwurf und eine Spur, sodass jeder sieht, was die anderen gebaut haben, und dort weitermacht, wo sie aufgehört haben.
- **Ein Entwurf, der wahr bleibt.** Wenn sich der Entwurf ändert und der Code nicht folgt, oder wenn sich der Code außerhalb jeder festgehaltenen Arbeit ändert, sagt tig es.
- **Den Entwurf ohne den Code zeigen.** Pushen Sie an einen Hub, damit Prüfer, Kunden oder Studierende den Entwurf lesen und sehen können, was verifiziert ist, ohne Zugriff auf den Quellcode.

## Installation

Mit Go 1.25 oder neuer:

```sh
go install github.com/h94u041/tig/cmd/tigprint@latest
```

Der Hub ist ein eigener Befehl: `go install github.com/h94u041/tig/cmd/tighub@latest`.

## Schnellstart

In einem Projekt, dessen Hauptdokument `README.md` ist:

```sh
tigprint init              # hier .tig anlegen
tigprint chain             # jeder Textblock der Kette und sein Status
tigprint check             # ist die Kette geschlossen? (Exit-Status 1, wenn nicht)
tigprint check -run-all    # tig jeden Abnahmeabschnitt selbst ausführen lassen
```

Ein Abschnitt mit dem Titel `Acceptance` listet die Befehle, die den Teil des Dokuments abnehmen, in dem er steht:

````markdown
## Turning

How a layer of the cube turns. Code: [turn.js](src/turn.js).

### Acceptance

- `npm test -- turn`
````

Dann verbinden Sie Ihre Agenten. Für Claude Code:

```sh
claude mcp add tig -- tigprint mcp
```

Oder in der Konfiguration eines beliebigen MCP-Clients:

```json
{ "mcpServers": { "tig": { "command": "tigprint", "args": ["mcp"] } } }
```

`tigprint mcp` bedient das Projekt, in dem es gestartet wird (oder `-C dir`). Mit `-run` dürfen Agenten tig Abnahmebefehle ausführen lassen.

## Wie es funktioniert

tig hat drei Teile und einen Hub zum Veröffentlichen, jeder mit einem eigenen Dokument (auf Englisch).

### Spur

Schnappschüsse des Projekts als inhaltsadressierte Bäume und die daran getanen Works, jede aus Steps. Siehe [the footprint](../docs/footprint.md).

### Kette

Textblöcke, benannt nach ihrem Inhalts-Hash, ihre Herkunft von Schnappschuss zu Schnappschuss, die Kette vom Hauptdokument bis zu den Dateien, die dafür gebaut wurden, und die Regeln der Geschlossenheit. Siehe [the chain](../docs/chain.md).

### MCP

Die Werkzeuge und Ressourcen, die `tigprint mcp` Agenten gibt, wie ein Agent damit arbeitet und was sie sicher macht. Siehe [the MCP server](../docs/mcp.md).

### Hub

`tigprint push` sendet die Dokumente und die Kette eines Projekts, und nichts sonst, an einen Hub, `tighub`, wo man sie liest wie ein Repository auf einem Git-Hoster. Siehe [the hub](../docs/hub.md).

## Befehle

`tigprint` allein listet jeden Befehl und seine Optionen. Der Code liegt in [cmd/tigprint](../cmd/tigprint/).

| Befehl | Was er tut |
|---|---|
| `init` | ein tig in diesem Ordner anlegen (`-root` das Hauptdokument, `-strict` nur von tig ausgeführte Prüfungen zählen) |
| `snap`, `log`, `diff`, `restore` | Schnappschüsse: einen aufnehmen, auflisten, vergleichen, Dateien zurückholen |
| `blocks <file.md>` | die Textblöcke eines Dokuments und ihre Hashes |
| `chain` | jeder Block der Kette und sein Status |
| `show <block>` | ein Block: sein Text, Status, Details, was dafür gebaut wurde und seine Works |
| `search <words>` | Blöcke, Works, Steps und Dateien, die jedes Wort enthalten |
| `check` | der Geschlossenheitsbericht; `-run <block>` oder `-run-all`, um zuerst die Abnahme auszuführen; `-json` |
| `trail <block or file>` | die dafür getanen Works; `-steps` für jeden Step und die Dateien, die er geändert hat |
| `work start`, `work step`, `work finish` | von Hand eine Spur hinterlassen |
| `mcp` | dieses tig über MCP auf stdin und stdout bereitstellen |
| `push [-token t] [-m note] <url>` | die Dokumente und die Kette an einen Hub senden |

Ein Block ist nach seinem Inhalts-Hash benannt; jedes eindeutige Präfix aus 8 oder mehr Zeichen funktioniert. Go-Programme können tig direkt verwenden: `import "github.com/h94u041/tig"`.

## Format

Der Ordner `.tig` enthält Objekte und Datensätze als Klartext, versioniert, sodass auch andere Werkzeuge sie lesen und schreiben können. Siehe [the format](../docs/format.md).

## Stand

Erste Version, 0.1. Das Format von `.tig` ist versioniert und auf Dauer angelegt; die Befehle und die MCP-Werkzeuge können sich noch ändern. Dieses Repository ist selbst eine geschlossene Kette: Sein `.tig` enthält die Spur, wie es gebaut wurde, und jeder Abnahmeabschnitt besteht, wenn tig ihn ausführt.

## Lizenz

MIT, siehe [LICENSE](../LICENSE). Geschrieben in Go ([go.mod](../go.mod), [go.sum](../go.sum)).

## Abnahme

- `go vet ./...`
- `go test ./...`
