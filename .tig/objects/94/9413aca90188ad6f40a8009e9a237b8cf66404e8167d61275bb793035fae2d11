# tig

**Le Markdown comme source.** tig fait des documents Markdown d'un projet la source de vérité de son code, les sert sous forme de serveur MCP à partir duquel n'importe quel agent d'IA peut construire, et garde une empreinte fidèle de chaque construction, pour que le travail puisse être vérifié, maîtrisé et reproduit.

[![Go Reference](https://pkg.go.dev/badge/github.com/h94u041/tig.svg)](https://pkg.go.dev/github.com/h94u041/tig)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT)

[English](../README.md) · [繁體中文](README.zh-TW.md) · [Deutsch](README.de.md) · **Français** · [Русский](README.ru.md) · [日本語](README.ja.md)

> Ceci est une traduction de [README.md](../README.md). La version anglaise est l'original, celle que tig vérifie ; en cas de différence, c'est elle qui fait foi.

> Ce tig n'est pas [tig, l'interface texte pour Git](https://github.com/jonas/tig). C'est pourquoi sa commande s'appelle `tigprint`.

## Pourquoi tig

Les agents écrivent désormais du code plus vite que quiconque ne peut le lire. Ce qui se perd, ce n'est pas le code, mais tout ce qui l'entoure :

- **L'intention.** Git enregistre quelles lignes ont changé, pas à quelle partie de la conception elles servaient. Six mois plus tard, personne ne sait dire pourquoi un fichier est tel qu'il est.
- **La preuve.** Un document de conception dit ce qui devrait être vrai ; rien ne dit si c'est encore le cas. La documentation s'éloigne du code en silence, et les tests vérifient le code, pas que chaque partie de la conception soit testée.
- **La passation.** Chaque session d'agent repart de zéro. Elle relit le projet, devine quels documents sont à jour, et refait un travail qu'un autre agent a déjà fait, ou le défait.

tig comble ces manques avec une seule idée : la conception est écrite en Markdown, chacune de ses parties dit comment elle est recettée, et chaque travail est enregistré auprès de la partie de la conception qu'il sert.

## Qu'est-ce que tig

- **Vous écrivez la conception** en Markdown ordinaire : un document principal qui renvoie à des documents de détail, et dans chaque partie une section *Acceptance* (recette) qui liste les commandes prouvant qu'elle fonctionne.
- **Les agents construisent à partir d'elle**, via MCP. Ils lisent et suivent la conception, voient comment elle a été construite auparavant, et enregistrent chaque étape de leur propre travail auprès de la partie de la conception qu'elle sert.
- **tig vérifie le résultat.** Il vous dit si chaque partie de la conception est couverte par une recette (la chaîne est *fermée*), si elle est passée, et où le code et la conception ont divergé.

Tout vit dans le dossier `.tig` du projet : des enregistrements en texte brut, adressés par leur contenu, qui vont dans Git avec le projet. Un tig est portable : donnez à quelqu'un les documents et l'empreinte, et son agent pourra reconstruire la même chose.

## Où se trouve son potentiel

Le code devient bon marché ; savoir à quoi il sert, et qu'il tient, ne l'est pas. C'est là que vise tig :

- **Un contrat entre les personnes et les agents.** Une chaîne fermée est une spécification qu'un agent peut exécuter et qu'une personne peut vérifier : les directions disent quoi construire, les guidelines ce qui a été construit et tient. « La chaîne est-elle fermée ? » peut conditionner une fusion comme le fait aujourd'hui « les tests passent-ils ? ».
- **Une mémoire qui survit à la session.** L'empreinte est un état partagé par chaque agent et chaque personne du projet : ce qui a été fait, pour quelle conception, par qui, et si cela est passé. L'agent suivant reprend là où le précédent s'est arrêté, pas à zéro.
- **La traçabilité sans bureaucratie.** Les domaines critiques et réglementés exigent que chaque exigence soit tracée jusqu'à son implémentation et à ses preuves de test, et le paient généralement avec des outils lourds. tig fournit la même traçabilité en texte brut, dans Git, tenue à jour comme un effet secondaire du travail.
- **Relire la conception, pas les diffs.** Comme chaque bloc de conception est nommé par son contenu et suivi à travers les modifications, un changement peut être relu ainsi : « cette partie de la conception a changé, et ces fichiers ont suivi, ou non ».
- **Publier ce qu'un projet promet et prouve, sans son code.** Avec le hub, une équipe peut montrer sa conception et quelles parties sont vérifiées, tandis que l'implémentation reste privée : spécifications ouvertes pour implémentations fermées, supports de cours et audits.

Ce que tig ne fait pas : une recette ne vaut que par ses commandes, et en mode `loose` tig croit les vérifications que rapportent les agents (utilisez `strict` quand cela compte). La recherche est une recherche par mots-clés. tig montre où la conception et le code divergent ; décider lequel a raison reste votre affaire.

## Fonctionnalités

- **Un serveur MCP pour votre projet.** `tigprint mcp` sert les documents et l'empreinte à n'importe quel client MCP (Claude Code, Cursor et d'autres) : une vue d'ensemble, la recherche par mots-clés, la lecture et le suivi des blocs de conception, l'empreinte de chaque bloc ou fichier, la vérification de fermeture et des outils pour enregistrer un nouveau travail.
- **Les documents comme une seule chaîne.** Le document principal, ses documents de détail et les fichiers vers lesquels ils renvoient sont lus comme un tout. Chaque bloc de texte est nommé par le hachage de son contenu, et tig suit chaque bloc à travers les modifications, déplacements, scissions et fusions selon des règles fixes et recalculables.
- **Une empreinte fidèle.** Chaque *work* (une construction, comme un commit) enregistre la conception qu'elle servait, son intention, le projet avant et après, et ses *steps* : chaque modification avec exactement les fichiers qu'elle a changés, chaque commande, vérification et décision.
- **Une reproduction à l'échelle de l'agent.** L'empreinte est assez complète pour qu'un second agent, avec seulement les documents et l'empreinte, reconstruise ce que le premier a construit. Les propres tests de tig font exactement cela.
- **Vérifications de fermeture.** Liens cassés, conception qu'aucune recette ne couvre, code modifié en dehors de l'empreinte ou laissé en arrière par un changement de conception, et fichiers qu'aucune conception n'atteint. Une conception couverte est une *direction* jusqu'à ce que sa recette passe, puis une *guideline*.
- **La maîtrise de ce qui s'exécute.** tig n'exécute que les commandes écrites dans les sections de recette, et via MCP seulement si vous démarrez le serveur avec `-run`. En mode `strict`, seules comptent les vérifications que tig a lui-même exécutées ; un agent ne peut pas simplement annoncer un succès.
- **Un hub où publier.** `tigprint push` envoie les documents et la chaîne, jamais le code, à un `tighub`, qui vérifie chaque envoi et affiche chaque document bloc par bloc avec son statut.
- **Sûr et portable.** Les objets sont vérifiés contre leur hachage à chaque lecture, les chemins venant des arbres d'autrui ne peuvent pas sortir du projet, et des écritures concurrentes ne perdent jamais d'enregistrement. Le format est du texte brut, versionné et fait pour durer.

## Cas d'usage

- **Enseignement et devoirs.** Hébergez une conception fermée comme serveur MCP et distribuez-la. L'agent de chaque étudiant construit selon les mêmes scénarios, et chaque rendu est une empreinte dont la recette peut être vérifiée, par tig lui-même en mode `strict`.
- **Suivre et maîtriser le travail des agents.** Chaque action d'un agent est enregistrée auprès de la conception qu'elle devait servir : ce qui a été fait, par qui, pourquoi, et si cela est passé.
- **Travail à plusieurs agents.** Les agents partagent une conception et une empreinte, de sorte que chacun voit ce que les autres ont construit et reprend là où ils se sont arrêtés.
- **Une conception qui reste vraie.** Quand la conception change et que le code ne suit pas, ou quand le code change en dehors de tout travail enregistré, tig le signale.
- **Montrer la conception sans le code.** Envoyez vers un hub pour que relecteurs, clients ou étudiants lisent la conception et voient ce qui est vérifié, sans accès au code source.

## Installation

Avec Go 1.25 ou plus récent :

```sh
go install github.com/h94u041/tig/cmd/tigprint@latest
```

Le hub est une commande à part : `go install github.com/h94u041/tig/cmd/tighub@latest`.

## Démarrage rapide

Dans un projet dont le document principal est `README.md` :

```sh
tigprint init              # créer .tig ici
tigprint chain             # chaque bloc de texte de la chaîne et son statut
tigprint check             # la chaîne est-elle fermée ? (code de sortie 1 sinon)
tigprint check -run-all    # faire exécuter chaque section de recette par tig lui-même
```

Une section intitulée `Acceptance` liste les commandes qui recettent la partie du document où elle se trouve :

````markdown
## Turning

How a layer of the cube turns. Code: [turn.js](src/turn.js).

### Acceptance

- `npm test -- turn`
````

Connectez ensuite vos agents. Pour Claude Code :

```sh
claude mcp add tig -- tigprint mcp
```

Ou dans la configuration de n'importe quel client MCP :

```json
{ "mcpServers": { "tig": { "command": "tigprint", "args": ["mcp"] } } }
```

`tigprint mcp` sert le projet dans lequel il est lancé (ou `-C dir`). Ajoutez `-run` pour laisser les agents faire exécuter les commandes de recette par tig.

## Fonctionnement

tig comporte trois parties, plus un hub où publier, chacune avec son propre document (en anglais).

### Empreinte

Des instantanés du projet sous forme d'arbres adressés par leur contenu, et les works faites dessus, chacune composée de steps. Voir [the footprint](../docs/footprint.md).

### Chaîne

Les blocs de texte nommés par le hachage de leur contenu, leur filiation d'un instantané à l'autre, la chaîne qui va du document principal aux fichiers construits pour lui, et les règles de fermeture. Voir [the chain](../docs/chain.md).

### MCP

Les outils et ressources que `tigprint mcp` donne aux agents, la façon dont un agent s'en sert, et ce qui les rend sûrs. Voir [the MCP server](../docs/mcp.md).

### Hub

`tigprint push` envoie les documents et la chaîne d'un projet, et rien d'autre, à un hub, `tighub`, où l'on les lit comme on lit un dépôt sur un hébergeur Git. Voir [the hub](../docs/hub.md).

## Commandes

`tigprint` seul liste chaque commande et ses options. Le code se trouve dans [cmd/tigprint](../cmd/tigprint/).

| Commande | Ce qu'elle fait |
|---|---|
| `init` | créer un tig dans ce dossier (`-root` le document principal, `-strict` pour ne compter que les vérifications exécutées par tig) |
| `snap`, `log`, `diff`, `restore` | instantanés : en prendre un, les lister, comparer, remettre des fichiers |
| `blocks <file.md>` | les blocs de texte d'un document et leurs hachages |
| `chain` | chaque bloc de la chaîne et son statut |
| `show <block>` | un bloc : son texte, son statut, ses détails, ce qui a été construit pour lui et ses works |
| `search <words>` | blocs, works, steps et fichiers contenant chaque mot |
| `check` | le rapport de fermeture ; `-run <block>` ou `-run-all` pour exécuter d'abord la recette ; `-json` |
| `trail <block or file>` | les works faites pour lui ; `-steps` pour chaque step et les fichiers qu'il a changés |
| `work start`, `work step`, `work finish` | laisser une empreinte à la main |
| `mcp` | servir ce tig via MCP sur stdin et stdout |
| `push [-token t] [-m note] <url>` | envoyer les documents et la chaîne à un hub |

Un bloc est nommé par le hachage de son contenu ; tout préfixe unique de 8 caractères ou plus fonctionne. Les programmes Go peuvent utiliser tig directement : `import "github.com/h94u041/tig"`.

## Format

Le dossier `.tig` contient des objets et des enregistrements en texte brut, versionnés, pour que d'autres outils puissent aussi les lire et les écrire. Voir [the format](../docs/format.md).

## État

Première version, 0.1. Le format de `.tig` est versionné et fait pour durer ; les commandes et les outils MCP peuvent encore changer. Ce dépôt est lui-même une chaîne fermée : son `.tig` contient l'empreinte de la façon dont il a été construit, et chaque section de recette passe quand tig l'exécute.

## Licence

MIT, voir [LICENSE](../LICENSE). Écrit en Go ([go.mod](../go.mod), [go.sum](../go.sum)).

## Recette

- `go vet ./...`
- `go test ./...`
