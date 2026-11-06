# tig

**Markdown as source。** tig は意図と足跡の台帳です。プロジェクトの Markdown 文書をコードの信頼できる情報源にし、あらゆる AI エージェントがそれに沿って構築できるよう MCP サーバーとして提供し、各行の意味、誰が同意したか、それに続くすべての施工を記録します。

[![Go Reference](https://pkg.go.dev/badge/github.com/h94u041/tig.svg)](https://pkg.go.dev/github.com/h94u041/tig)
[![test](https://github.com/h94u041/tig/actions/workflows/test.yml/badge.svg)](https://github.com/h94u041/tig/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT)

[English](../README.md) · [繁體中文](README.zh-TW.md) · [Deutsch](README.de.md) · [Français](README.fr.md) · [Русский](README.ru.md) · **日本語**

> この tig は [Git のテキストインターフェースである tig](https://github.com/jonas/tig) とは別物です。そのためコマンド名は `tigwx` です。

## なぜ tig か

エージェントは今や、誰にも読みきれない速さでコードを書きます。失われるのはコードではなく、その周りのすべてです。

- **意図。** Git はどの行が変わったかを記録しますが、設計のどの部分のためだったかは記録しません。半年後には、なぜそのファイルがそうなっているのか誰にも説明できません。
- **合意。** 設計書は何が成り立つべきかを語りますが、誰がそう言ったのかは語りません。人なのか、推測しているエージェントなのか。文書は黙ってコードから離れていき、一文の編集は周囲すべてについての心変わりと見分けがつきません。
- **引き継ぎ。** エージェントのセッションは毎回ゼロから始まります。プロジェクトを読み直し、どの文書が最新かを推測し、別のエージェントがした作業を繰り返すか、元に戻してしまいます。

tig は一つの考え方でこの隙間を埋めます。設計を Markdown で書き、各行にその意味へつながる小さなマークを付け、すべての作業をその意味に対して記録する。tig は記録するだけで、判定はしません。作業が十分かどうかはあなたか、その上で動かすツールが決め、tig はその決定を保存します。

## tig にできること

**次の問いに答えられます。コマンドラインから、JSON で、または MCP 経由でエージェントに。**

| 知りたいこと | 聞き方 |
|---|---|
| この行の意味は？ 誰が言った？ 誰か同意した？ | `tigwx show i7f3a` |
| この意図に何が起き、なぜ変わった？ | `tigwx log i7f3a` |
| これに沿って作ったとき、この要件は何と書いてあった？ | `tigwx set -i i7f3a#1` |
| どの作業がこのファイルを変え、何のためだった？ | `tigwx trail src/cart.go` |
| これに取り組むのに読むべきものは何で、それだけで足りる？ | `tigwx set -i i7f3a` |
| 設計とコードはまだ一致している？ | `tigwx check`（一致しないと終了ステータス 1） |
| 誰もつないでいないコードで、どの行がどのファイルのことらしい？ | `tigwx reverse` |
| 決定待ちのものは何？ | `tigwx status` |

**次のものを区別して扱えます。Git や文書だけではできないことです。**

- 人の決定とエージェントの推測。エージェントの文は、誰かが同意するまで別の推測文書で待ちます。MCP 経由のエージェントは提案しかできません。
- その行が正しいかどうかと、このコードがそれに属するかどうか。別々の問いで、それぞれに答えがあります。
- 意図した編集と、誰も説明していない編集。編集が届いた意図だけ、もう一度見れば済みます。
- ファイルを共有するコードと、本当に関係のある意図。`depends_on`、`parallel`、あるいは自分の言葉で明示できます。

**次のことをしてくれます。**

- **プロジェクトをあらゆるエージェントに提供する。** `tigwx mcp` は Claude Code、Cursor などの MCP クライアントに、設計、足跡、閉包チェック、新しい作業を記録するツールを渡します。
- **忠実な足跡を残す。** 各作業は、対象の設計、前後のプロジェクト、すべてのステップを記録します。各編集が変えた正確なファイル、各コマンド、チェック、決定です。文書と足跡だけを渡された別のエージェントが同じものを作り直せます。tig 自身のテストがまさにそれを行っています。
- **すでにコードのあるプロジェクトをつなぐ。** モデルは使いません。`tigwx reverse` が行の指す先を読んで提案し、あなたが決めます。
- **サーバーなしでマージする。** 記録は変更されずハッシュで命名されるので、2 台のマシンはファイルを合わせるだけでマージできます。決定が食い違えば、黙ってどちらかを選ばずに tig がそう伝えます。
- **コードなしで設計を公開する。** `tigwx push` は文書とチェーンを、誰でも読めるハブへ送ります。コードは決して送りません。
- **邪魔をしない。** すべて `.tig` にプレーンテキストで置かれ、プロジェクトと一緒に Git に入ります。`init` は何も尋ねず、文書も変えません。tig は何も実行しません。

**しないこと。** チェックを実行せず、判定も下しません。エージェントが記録したチェックは、そのエージェントが言ったことにすぎません。「閉じている」とは文書、台帳、コードが互いに一致しているという意味で、コードが良いという意味ではありません。検索はキーワード検索です。`by` は書き手の主張で、署名はありません。tig は何が言われたかを伝え、誰が言えたのかは Git ホスティングが伝えます。

## インストール

[Releases](https://github.com/h94u041/tig/releases) からお使いのシステムのセットアップを入手して実行します。他には何も要りません。Windows では `%LOCALAPPDATA%\Programs\tigwx` に入り、PATH に加わり、インストール済みプログラムの一覧に表示されます。それ以外では `~/.local/bin` に入り、PATH にない場合は追加する行を案内します。`-dir`、`-yes`、`-portable`、`-uninstall` は文字どおりの動作です。プログラムだけのアーカイブと `SHA256SUMS` も同じ場所にあります。セットアップはまだ署名されていないため、Windows と macOS では実行前に警告が出ることがあります。

または Go 1.26 以降で：

```sh
go install github.com/h94u041/tig/cmd/tigwx@latest
```

`@latest` はタグ付きの最新リリースです。このページは `main` に沿っており、その後の変更は[変更履歴](../CHANGELOG.md)にあります。ハブは別コマンドです：`go install github.com/h94u041/tig/cmd/tighub@latest`。

## 5 分で試す

フォルダを作り、2 つのファイルを置きます。意図の ID はハッシュから作られるので、あなたの環境では異なります。

`README.md`：

```markdown
# Shop

Carts are kept for a day. Code: [cart](src/cart.go).

Prices come from the table.
```

`src/cart.go`：

```go
package shop

const keepFor = "24h"
```

**始めて、行の意味を言う。** `init` は何も尋ねず、文書も変えません。`mark` は行末に小さなマークを書き（Markdown ビューアーでは見えません）、意図の ID を返します。`-confirm` は、これが自分の意味したことだと言います。

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

**考えを変える。** `README.md` を *a week* に編集し、マークはそのままにします。意図は同一性を保ったまま新しいリビジョンになります。誰も理由を言っていないので、誰かがもう一度見るべきです。

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

**作業して足跡を残す。** エージェントは MCP でこれを行います。ここでは手作業です。`start` と `step` の間に `src/cart.go` を `168h` に変えます。

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

**結びつけて、決める。** このコミットはあの行に沿っており、あの行は自分の意味したことです。`-source` は決定の出どころを言います。

```console
$ tigwx mark -id 4086 "Carts are kept for a week."
i90e7  README.md:3  4086  Carts are kept for a week. Code: [cart](src/cart.go).
$ tigwx confirm -m "the shop stays open longer now" -source "ticket 12" i90e7
i90e7#2  confirmed  Carts are kept for a week.
$ tigwx check
closed: yes
```

**問いかけてみる。**

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

**2 つの意図の関係を言う。** コードは共有していても意味は別のとき、あるいは一方が他方を必要とするときに。tig はそれを保存し、求められたものから何も取り除きません。

```console
$ tigwx relate -kind parallel -m "nothing in common" i90e7 i70fd
i90e7 parallel i70fd: confirmed
```

`.tig` をプロジェクトの他の部分と一緒に Git に追加すれば、プロジェクトと一緒に移動します。

## エージェントと使う

Claude Code の場合：

```sh
claude mcp add tig -- tigwx mcp
```

または任意の MCP クライアントの設定に：

```json
{ "mcpServers": { "tig": { "command": "tigwx", "args": ["mcp"] } } }
```

`tigwx mcp` は起動した場所のプロジェクト（または `-C dir`）を提供します。エージェントは設計を読み、対象の部分について作業を始め、各ステップを記録し、自分のコミットがどの文に沿うかを提案します。あなたは `tigwx confirm`、`tigwx reject`、`tigwx link`、`tigwx accept` で確定または却下します。[ツールと安全性](../docs/mcp.md)。

すでにコードと文書があるのに何もつながっていないプロジェクトでは：

```sh
tigwx reverse           # どの行がどのファイルのことらしいか。何も書きません
tigwx reverse -write    # それらを提案します。同意するまで何も結びつきません
tigwx init -force       # または一度に：今あるコードを 1 コミットとし、全行をそれに沿わせます
```

## コマンド

`tigwx --help` がすべてのコマンドとフラグを示します。`mcp` 以外のすべてのコマンドが `-json` を受け付けます。コマンドラインで `-by` も `user.by` もないとき、端末からなら `human`、それ以外なら `unknown` と記録します。スクリプトは誰の代わりかを明示してください。

| コマンド | 内容 |
|---|---|
| `init` | このフォルダで tig を始める（`-root` で主文書を指定）。何も尋ねず、コミットも記録しない。`-force`：今あるコードを 1 コミットとし、全行をそれに沿わせる |
| `status` | プロジェクトの状況と、決定待ちのもの |
| `config` | 自分の名前、`by`、主文書、チェーンの外に置くパスを読み書きする |
| `snap`, `log`, `diff`, `restore` | スナップショットの作成、一覧、比較、復元。`log <intent>`：意図に起きたすべて |
| `blocks <file.md>` | 文書のテキストブロックとハッシュ |
| `chain` | チェーンのすべてのブロック |
| `show <ブロックまたは参照>` | 1 つのブロック、または参照の意味：`i7f3a`、`i7f3a#2`、`lc3d4e5:5-14`、`a1b2`、`ev91c0` |
| `search <語>` | すべての語を含むブロック、作業、ステップ、ファイル |
| `check` | 閉包レポート |
| `trail <ブロックまたはファイル>` | それに対する作業。`-steps` で各ステップと変更ファイル |
| `work start`, `work step`, `work finish` | 手作業で足跡を残す |
| `commits`, `commit <id>` | コミットの一覧（新しい順）。1 つのコミットのファイルと文 |
| `mark`, `sync`, `unmark` | 行の意味を言う。編集された行に台帳を追いつかせる（`-m why`：意図した変更）。取り消されたコミットを撤回する |
| `reverse`, `describe`, `link` | モデルなしで、どの行がどのファイルのことかを探す。行が既存ファイルのことだと言う。コードが意図に属するか決める |
| `relate` | 2 つの意図の関係：`depends_on`、`part_of`、`parallel`、または自分の言葉 |
| `set`, `groups` | 作業に必要なもの。まとまるもの |
| `confirm`, `reject`, `retire`, `accept` | 意図やコミットへの決定を記録する |
| `mcp` | この tig を stdin と stdout の MCP で提供する |
| `push [-token t] [-m note] [-insecure] <url>` | 文書とチェーンをハブへ送る |

ブロックは内容のハッシュで命名され、8 文字以上の一意な接頭辞で指定できます。Go プログラムからは直接使えます：`import "github.com/h94u041/tig"`。コードは [cmd/tigwx](../cmd/tigwx/) と [cmd/tigwx-setup](../cmd/tigwx-setup/) に、リリースは [scripts](../scripts/) と [.github](../.github/) でビルドされます。

## 可能性はどこにあるか

コードは安くなりつつありますが、それが何のためで、本当に成り立っているかを知ることは安くなっていません。tig が狙うのはそこです。

- **人とエージェントの契約。** 確定した意図はエージェントが構築してよいものです。自分の推測は推測として示され続けます。「この変更の全行に合意があるか？」が、今日の「テストは通ったか？」のようにマージの関門になり得ます。
- **セッションより長く残る記憶。** 足跡はプロジェクトの全エージェントと全員の共有状態です。何を、どの意図のために、誰が行い、受け入れられたか。次のエージェントはゼロからではなく、前のエージェントが止まった所から始めます。
- **官僚主義なしのトレーサビリティ。** 安全重視や規制のある分野では、すべての要件を実装とテストの証拠まで追跡する必要があり、たいてい重いツールが要ります。tig は同じ追跡をプレーンテキストで Git の中に提供し、作業の副産物として最新に保ちます。
- **差分ではなく設計のレビュー。** 意図には編集を超えて残る同一性があるので、変更を「この意味が変わり、これらのコミットが追随した、あるいはしなかった」とレビューできます。
- **コードなしでプロジェクトの約束を公開する。** 非公開実装の公開仕様、学生全員のエージェントが同じシナリオで構築する教材、監査など。

## 仕組み

すべてはプロジェクトの `.tig` フォルダにあります。内容アドレス方式のプレーンな記録で、プロジェクトと一緒に Git に入ります。tig は持ち運べます。文書と足跡を渡せば、相手のエージェントが同じものを作り直せます。

- **[足跡](../docs/footprint.md)。** 内容アドレスのツリーとしてのスナップショット、行われた作業、鍵を入れないための仕組み。
- **[チェーン](../docs/chain.md)。** 内容ハッシュで命名されたテキストブロック、行マーク、意図と参照、意図間の関係、主文書から構築ファイルまでのチェーン、閉包の規則。
- **[MCP](../docs/mcp.md)。** `tigwx mcp` がエージェントに渡すツールとリソース、使い方、安全性。
- **[ハブ](../docs/hub.md)。** `tigwx push` はプロジェクトの文書とチェーンだけをハブ（`tighub`）へ送り、Git ホスティングのリポジトリのように読めるようにします。
- **[形式](../docs/format.md)。** `.tig` にはバージョン付きのプレーンテキストのオブジェクトと記録があり、他のツールも読み書きできます。どのリリースも以前のリリースが書いたものを読め、古い版も新しい版が書いたものを読めます。
- **[tig の上に作る](../docs/tools.md)。** 自分のツールから、コマンド、Go、またはファイル経由で使う方法。

**どこまで耐えるか。** Windows での実測：意図 5,000 個で `check` は 1 秒未満、もう 1 行マークするのに約 0.5 秒。12,000 ファイルのプロジェクト（Go 標準ライブラリ）で、最初のスナップショットは 20 秒未満、以降 `check` は約 0.3 秒、`status` は約 3 秒。Windows では書き込み直後のファイルの読み取りは、ウイルス対策の検査中は遅くなります。`.tig` はディスク上でプロジェクトとほぼ同じ大きさですが、Git がよく圧縮します。`TIG_SCALE=2000 go test -run TestScale -v .` ([scale_test.go](../scale_test.go)) で自分の環境を測れます。

## セキュリティ

tig は何も実行せず、パスをプロジェクト内に留め、記録をハッシュで確認し、鍵らしいファイルをスナップショットに入れません。できないことも明示しています。頼る前に、また問題を報告するときは [SECURITY.md](../SECURITY.md) をお読みください。**v0.1.0 からアップグレードしてください。** 理由はそこにあります。

## 貢献

Issue と pull request を歓迎します。何が合うか、どうするかは [CONTRIBUTING.md](../CONTRIBUTING.md) に、バージョン間の変更は[変更履歴](../CHANGELOG.md)にあります。

このリポジトリ自体が閉じたチェーンです。`.tig` には構築の足跡があり、受け入れの節に手で実行するコマンドがあります。

この英語の README が原本です。翻訳はそれに従い、チェーンの外に置かれています（`.tig/config` の `outside`）。異なる場合は英語版が優先します。

## ライセンス

MIT。[LICENSE](../LICENSE) を参照。Go 製です（[go.mod](../go.mod)、[go.sum](../go.sum)）。

## 受け入れ

- `go vet ./...`
- `go test ./...`
- `sh scripts/compat.sh <新しい tigwx> <tigwx 0.1.0>`
