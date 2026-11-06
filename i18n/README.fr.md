# tig

**Markdown comme source.** tig est un registre d'intentions et de traces : il fait des documents Markdown d'un projet la source de vérité de son code, les sert via un serveur MCP à partir duquel n'importe quel agent IA peut construire, et consigne ce que signifie chaque ligne, qui l'a approuvée et chaque construction qui l'a suivie.

[![Go Reference](https://pkg.go.dev/badge/github.com/h94u041/tig.svg)](https://pkg.go.dev/github.com/h94u041/tig)
[![test](https://github.com/h94u041/tig/actions/workflows/test.yml/badge.svg)](https://github.com/h94u041/tig/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT)

[English](../README.md) · [繁體中文](README.zh-TW.md) · [Deutsch](README.de.md) · **Français** · [Русский](README.ru.md) · [日本語](README.ja.md)

> Ce tig n'est pas [tig, l'interface texte pour Git](https://github.com/jonas/tig). C'est pourquoi sa commande s'appelle `tigwx`.

## Pourquoi tig

Les agents écrivent désormais du code plus vite que quiconque ne peut le lire. Ce qui se perd n'est pas le code, mais tout ce qui l'entoure :

- **L'intention.** Git enregistre quelles lignes ont changé, pas à quelle partie de la conception elles servaient. Six mois plus tard, personne ne sait dire pourquoi un fichier est ainsi.
- **L'accord.** Un document de conception dit ce qui devrait être vrai, mais pas qui l'a dit : une personne, ou un agent qui devine. La documentation s'éloigne du code en silence, et modifier une phrase ressemble à un changement d'avis sur tout ce qui l'entoure.
- **La passation.** Chaque session d'agent repart de zéro. Elle relit le projet, devine quels documents sont à jour, puis refait le travail d'un autre agent, ou le défait.

tig comble ces manques avec une seule idée : la conception est écrite en Markdown, chaque ligne porte une petite marque qui mène à sa signification, et chaque travail est consigné par rapport à cette signification. tig enregistre ; il ne juge pas. Vous, ou l'outil que vous placez au-dessus, décidez si un travail suffit, et tig conserve la décision.

## Ce que tig sait faire

**Il répond à ces questions, en ligne de commande, en JSON, ou à un agent via MCP :**

| Vous voulez savoir | Demandez |
|---|---|
| Que signifie cette ligne, qui l'a dit, quelqu'un l'a-t-il approuvée ? | `tigwx show i7f3a` |
| Qu'est-il arrivé à cette intention, et pourquoi a-t-elle changé ? | `tigwx log i7f3a` |
| Que disait cette exigence quand on a construit d'après elle ? | `tigwx set -i i7f3a#1` |
| Quels travaux ont modifié ce fichier, et dans quel but ? | `tigwx trail src/cart.go` |
| Que dois-je lire pour travailler là-dessus, et rien de plus ? | `tigwx set -i i7f3a` |
| La conception et le code sont-ils encore d'accord ? | `tigwx check` (code de sortie 1 sinon) |
| Quelles lignes semblent concerner quels fichiers, dans du code que personne n'a relié ? | `tigwx reverse` |
| Qu'est-ce qui attend une décision ? | `tigwx status` |

**Il sépare ce que Git et un document seuls ne peuvent pas séparer :**

- La décision d'une personne et la supposition d'un agent. La phrase d'un agent attend dans un document déduit séparé jusqu'à ce que quelqu'un l'approuve, et via MCP un agent ne peut que proposer.
- Si une ligne est juste, et si ce code lui correspond. Ce sont deux questions, chacune avec sa réponse.
- Une modification voulue, et une modification que personne n'a expliquée. Seules les intentions atteintes par une modification demandent un nouveau regard.
- Du code qui partage un fichier, et des intentions réellement liées. Vous pouvez le dire : `depends_on`, `parallel`, ou votre propre mot.

**Il fait cela pour vous :**

- **Servir votre projet à n'importe quel agent.** `tigwx mcp` donne à Claude Code, Cursor ou tout client MCP la conception, la trace, la vérification de clôture et des outils pour consigner le nouveau travail.
- **Tenir une trace fidèle.** Chaque travail consigne la conception qu'il servait, le projet avant et après, et chaque étape : chaque modification avec exactement les fichiers touchés, chaque commande, vérification et décision. Un second agent, avec seulement les documents et la trace, peut reconstruire la même chose, et les tests de tig font exactement cela.
- **Relier un projet qui a déjà du code,** sans modèle : `tigwx reverse` lit ce que désignent les lignes et le propose, et vous décidez.
- **Fusionner sans serveur.** Les enregistrements ne changent jamais et sont nommés par leur hash, donc deux machines fusionnent en réunissant leurs fichiers. Si elles ont décidé différemment, tig le dit au lieu d'en choisir une en silence.
- **Publier la conception sans le code.** `tigwx push` envoie les documents et la chaîne, jamais le code, à un hub que chacun peut lire.
- **Rester discret.** Tout vit dans `.tig`, en texte brut, dans Git avec votre projet. `init` ne demande rien et ne modifie aucun document. tig n'exécute rien.

**Ce qu'il ne fait pas.** Il n'exécute aucune vérification et ne rend aucun verdict ; une vérification consignée par un agent n'est que ce que l'agent a dit. Clos signifie que les documents, le registre et le code sont d'accord entre eux, pas que le code est bon. La recherche est une recherche par mots-clés. `by` est ce que l'auteur a déclaré et rien n'est signé : tig vous dit ce qui a été dit, et votre hébergeur Git qui a pu le dire.

## Installation

Récupérez le programme d'installation de votre système sur [Releases](https://github.com/h94u041/tig/releases) et lancez-le ; il ne demande rien d'autre. Sous Windows, il va dans `%LOCALAPPDATA%\Programs\tigwx`, rejoint votre PATH et apparaît dans la liste des programmes installés. Ailleurs, il va dans `~/.local/bin` et vous indique la ligne à ajouter si ce dossier n'est pas dans votre PATH. `-dir`, `-yes`, `-portable` et `-uninstall` font ce qu'ils disent. Des archives contenant seulement le programme, et `SHA256SUMS`, sont au même endroit. Le programme d'installation n'est pas encore signé : Windows et macOS peuvent vous avertir avant de le lancer.

Ou avec Go 1.26 ou plus récent :

```sh
go install github.com/h94u041/tig/cmd/tigwx@latest
```

`@latest` est la dernière version étiquetée ; cette page suit `main`, et le [journal des modifications](../CHANGELOG.md) dit ce qui est nouveau depuis. Le hub est une commande à part : `go install github.com/h94u041/tig/cmd/tighub@latest`.

## Essayer en cinq minutes

Créez un dossier avec deux fichiers. Vos identifiants d'intention seront différents : ils viennent d'un hash.

`README.md` :

```markdown
# Shop

Carts are kept for a day. Code: [cart](src/cart.go).

Prices come from the table.
```

`src/cart.go` :

```go
package shop

const keepFor = "24h"
```

**Commencer, et dire ce que signifient les lignes.** `init` ne demande rien et ne modifie aucun document. `mark` écrit une petite marque en fin de ligne, invisible dans tout afficheur Markdown, et répond avec l'identifiant de l'intention. `-confirm` dit que c'est bien ce que vous vouliez dire.

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

**Changer d'avis.** Modifiez `README.md` pour écrire *a week*, et laissez la marque en place. L'intention garde son identité et reçoit une nouvelle révision ; comme personne n'a dit pourquoi, quelqu'un devrait y regarder à nouveau.

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

**Faire le travail, et laisser une trace.** Les agents le font via MCP ; ici, c'est à la main. Entre `start` et `step`, passez `src/cart.go` à `168h`.

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

**Relier, et décider.** Ce commit suit cette ligne, et cette ligne est ce que vous vouliez dire. `-source` dit d'où vient la décision.

```console
$ tigwx mark -id 4086 "Carts are kept for a week."
i90e7  README.md:3  4086  Carts are kept for a week. Code: [cart](src/cart.go).
$ tigwx confirm -m "the shop stays open longer now" -source "ticket 12" i90e7
i90e7#2  confirmed  Carts are kept for a week.
$ tigwx check
closed: yes
```

**Maintenant, interrogez-le.**

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

**Dire ce que deux intentions ont à voir l'une avec l'autre,** quand elles partagent du code mais pas de sens, ou que l'une a besoin de l'autre. tig le conserve et ne retire rien de ce que vous demandez.

```console
$ tigwx relate -kind parallel -m "nothing in common" i90e7 i70fd
i90e7 parallel i70fd: confirmed
```

Ajoutez `.tig` à Git avec le reste du projet, et il voyage avec lui.

## Utilisation avec un agent

Pour Claude Code :

```sh
claude mcp add tig -- tigwx mcp
```

Ou dans la configuration de n'importe quel client MCP :

```json
{ "mcpServers": { "tig": { "command": "tigwx", "args": ["mcp"] } } }
```

`tigwx mcp` sert le projet dans lequel il est lancé (ou `-C dir`). L'agent lit la conception, commence un travail pour la partie concernée, consigne chaque étape et propose les phrases suivies par son commit ; vous confirmez ou rejetez avec `tigwx confirm`, `tigwx reject`, `tigwx link` et `tigwx accept`. [Les outils et ce qui les rend sûrs](../docs/mcp.md).

Pour un projet qui a du code et des documents que rien ne relie :

```sh
tigwx reverse           # quelles lignes semblent concerner quels fichiers ; il n'écrit rien
tigwx reverse -write    # les proposer ; rien ne s'applique avant votre accord
tigwx init -force       # ou tout d'un coup : le code présent est un commit, et chaque ligne s'y rattache
```

## Commandes

`tigwx --help` liste toutes les commandes et leurs options. Toutes sauf `mcp` acceptent `-json`. En ligne de commande, sans `-by` ni `user.by`, un terminal enregistre `human` et tout le reste `unknown` : un script doit dire pour qui il parle.

| Commande | Ce qu'elle fait |
|---|---|
| `init` | démarrer tig dans ce dossier (`-root` le document principal) ; ne demande rien et n'enregistre aucun commit. `-force` : le code présent est un commit et chaque ligne s'y rattache |
| `status` | où en est le projet, et ce qui attend une décision |
| `config` | lire ou définir votre nom, votre `by`, le document principal et les chemins hors de la chaîne |
| `snap`, `log`, `diff`, `restore` | instantanés : en prendre, les lister, comparer, restaurer des fichiers. `log <intent>` : tout ce qui est arrivé à une intention |
| `blocks <file.md>` | les blocs de texte d'un document et leurs hashes |
| `chain` | chaque bloc de la chaîne |
| `show <bloc ou référence>` | un bloc, ou ce que signifie une référence : `i7f3a`, `i7f3a#2`, `lc3d4e5:5-14`, `a1b2`, `ev91c0` |
| `search <mots>` | blocs, travaux, étapes et fichiers contenant tous les mots |
| `check` | le rapport de clôture |
| `trail <bloc ou fichier>` | les travaux effectués pour lui ; `-steps` pour chaque étape et ses fichiers |
| `work start`, `work step`, `work finish` | laisser une trace à la main |
| `commits`, `commit <id>` | les commits, du plus récent au plus ancien ; un commit avec ses fichiers et ses phrases |
| `mark`, `sync`, `unmark` | dire ce que signifie une ligne ; mettre le registre à jour des lignes modifiées (`-m why` : le changement était voulu) ; retirer des commits annulés |
| `reverse`, `describe`, `link` | trouver sans modèle quelles lignes concernent quels fichiers ; dire qu'une ligne concerne des fichiers existants ; décider si du code correspond à une intention |
| `relate` | ce que deux intentions ont à voir : `depends_on`, `part_of`, `parallel`, ou votre propre mot |
| `set`, `groups` | ce dont un travail a besoin ; ce qui va ensemble |
| `confirm`, `reject`, `retire`, `accept` | consigner une décision sur une intention ou un commit |
| `mcp` | servir ce tig via MCP sur stdin et stdout |
| `push [-token t] [-m note] [-insecure] <url>` | envoyer les documents et la chaîne à un hub |

Un bloc est nommé par le hash de son contenu ; tout préfixe unique de 8 caractères ou plus convient. Les programmes Go peuvent utiliser tig directement : `import "github.com/h94u041/tig"`. Le code est dans [cmd/tigwx](../cmd/tigwx/) et [cmd/tigwx-setup](../cmd/tigwx-setup/) ; les versions sont construites par [scripts](../scripts/) et [.github](../.github/).

## Où est son potentiel

Le code devient bon marché ; savoir à quoi il sert, et qu'il tient, ne le devient pas. C'est là que vise tig :

- **Un contrat entre personnes et agents.** Les intentions confirmées sont ce à partir de quoi un agent peut construire ; ses suppositions restent marquées comme telles. « Chaque ligne de ce changement est-elle approuvée ? » peut conditionner une fusion comme « les tests passent-ils ? » aujourd'hui.
- **Une mémoire qui survit à la session.** La trace est un état partagé par tous les agents et personnes d'un projet : ce qui a été fait, pour quelle intention, par qui, et si quelqu'un l'a accepté. L'agent suivant commence là où le précédent s'est arrêté.
- **La traçabilité sans bureaucratie.** Les domaines critiques et réglementés exigent que chaque exigence remonte à son implémentation et à ses preuves de test, et le paient souvent d'outils lourds. tig offre la même trace en texte brut, dans Git, tenue à jour comme effet secondaire du travail.
- **Relire la conception, pas les diffs.** Comme chaque intention a une identité qui survit aux modifications, un changement peut se relire ainsi : « ce sens a changé, et ces commits l'ont suivi, ou non ».
- **Publier ce qu'un projet promet, sans son code.** Spécifications ouvertes pour implémentations fermées, supports de cours où l'agent de chaque étudiant construit d'après les mêmes scénarios, et audits.

## Comment ça marche

Tout vit dans le dossier `.tig` du projet : des enregistrements simples, adressés par contenu, qui vont dans Git avec le projet. Un tig est portable : donnez à quelqu'un les documents et la trace, et son agent pourra reconstruire la même chose.

- **[Trace](../docs/footprint.md).** Les instantanés sous forme d'arbres adressés par contenu, les travaux effectués, et ce qui tient les clés à l'écart.
- **[Chaîne](../docs/chain.md).** Blocs de texte nommés par hash, marques de ligne, intentions et références, relations entre intentions, chaîne du document principal aux fichiers construits, et règles de clôture.
- **[MCP](../docs/mcp.md).** Les outils et ressources de `tigwx mcp`, comment un agent s'en sert, et ce qui les rend sûrs.
- **[Hub](../docs/hub.md).** `tigwx push` envoie seulement les documents et la chaîne à un hub, `tighub`, où on les lit comme un dépôt sur un hébergeur Git.
- **[Format](../docs/format.md).** `.tig` contient des objets et enregistrements en texte brut, versionnés, que d'autres outils peuvent lire et écrire. Chaque version lit ce qu'écrivaient les précédentes, et les anciennes lisent ce qu'écrivent les nouvelles.
- **[Construire sur tig](../docs/tools.md).** L'utiliser depuis votre outil : en commande, en Go ou par les fichiers.

**Jusqu'où ça tient.** Mesuré sous Windows : avec 5 000 intentions, `check` prend moins d'une seconde et marquer une ligne de plus environ une demi-seconde. Sur un projet de 12 000 fichiers (la bibliothèque standard de Go), le premier instantané prend moins de 20 secondes, puis `check` environ 0,3 seconde et `status` environ 3. Sous Windows, lire des fichiers tout juste écrits est plus lent pendant que l'antivirus les examine. `.tig` a à peu près la taille du projet sur disque, et Git le compresse bien. `TIG_SCALE=2000 go test -run TestScale -v .` ([scale_test.go](../scale_test.go)) le mesure chez vous.

## Sécurité

tig n'exécute rien, garde les chemins dans le projet, vérifie les enregistrements par leurs hashes et garde les fichiers ressemblant à des clés hors de ses instantanés. Il dit aussi clairement ce qu'il ne peut pas faire : lisez [SECURITY.md](../SECURITY.md) avant de vous y fier, ou pour signaler un problème. **Veuillez mettre à niveau depuis v0.1.0** : il explique pourquoi.

## Contribuer

Les issues et pull requests sont les bienvenues. [CONTRIBUTING.md](../CONTRIBUTING.md) dit ce qui convient et comment. Ce qui a changé entre les versions est dans le [journal des modifications](../CHANGELOG.md).

Ce dépôt est sa propre chaîne close : son `.tig` contient la trace de sa construction, et ses sections d'acceptation listent les commandes à lancer à la main.

Ce README en anglais est la source. Les traductions le suivent et restent hors de la chaîne (`outside` dans `.tig/config`) ; en cas de différence, l'anglais prévaut.

## Licence

MIT, voir [LICENSE](../LICENSE). Écrit en Go ([go.mod](../go.mod), [go.sum](../go.sum)).

## Acceptation

- `go vet ./...`
- `go test ./...`
- `sh scripts/compat.sh <nouveau tigwx> <tigwx 0.1.0>`
