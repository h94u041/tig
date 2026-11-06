# tig

**Markdown as source。** tig 是一本意圖與足跡的帳本：它讓專案的 Markdown 文件成為程式碼的真實來源，以 MCP 伺服器提供給任何 AI Agent 依照施工，並記下每一行的意思、誰同意過，以及之後的每一次施工。

[![Go Reference](https://pkg.go.dev/badge/github.com/h94u041/tig.svg)](https://pkg.go.dev/github.com/h94u041/tig)
[![test](https://github.com/h94u041/tig/actions/workflows/test.yml/badge.svg)](https://github.com/h94u041/tig/actions/workflows/test.yml)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT)

[English](../README.md) · **繁體中文** · [Deutsch](README.de.md) · [Français](README.fr.md) · [Русский](README.ru.md) · [日本語](README.ja.md)

> 這個 tig 不是 [tig，Git 的文字介面](https://github.com/jonas/tig)。所以它的指令叫 `tigwx`。

## 為什麼需要 tig

Agent 寫程式的速度已經超過任何人讀得完的速度。流失的不是程式碼，而是程式碼周圍的一切：

- **意圖。** Git 記錄哪幾行變了，卻不記錄它們是為了設計的哪個部分。半年後，沒有人說得出某個檔案為什麼長這樣。
- **同意。** 設計文件說了什麼應該成立，卻沒說是誰說的：是一個人，還是一個在猜測的 Agent。文件會無聲地偏離程式碼，而改一句話，看起來和對周圍所有內容改變心意沒有兩樣。
- **交接。** 每一次 Agent 工作階段都從零開始。它重讀專案，猜哪些文件還算數，再重複另一個 Agent 做過的事，或把它復原。

tig 用一個想法補上這些缺口：設計寫成 Markdown，每一行帶一個小標記，連到它的意思，每一件工作都對照那個意思記錄下來。tig 只記錄，不評判。工作夠不夠好，由你或你架在它上面的工具決定，tig 保存那個決定。

## tig 能做什麼

**它能回答這些問題，從命令列、以 JSON，或透過 MCP 回答 Agent：**

| 你想知道 | 這樣問 |
|---|---|
| 這一行是什麼意思、誰說的、有人同意了嗎？ | `tigwx show i7f3a` |
| 這個意圖發生過什麼事、為什麼改過？ | `tigwx log i7f3a` |
| 當初照著施工時，這條需求是怎麼寫的？ | `tigwx set -i i7f3a#1` |
| 哪些工作改過這個檔案、當時是為了什麼？ | `tigwx trail src/cart.go` |
| 要做這件事，我需要讀什麼，而且只需要讀什麼？ | `tigwx set -i i7f3a` |
| 設計和程式碼還一致嗎？ | `tigwx check`（不一致時結束代碼為 1） |
| 在沒有人接上的程式碼裡，哪幾行看起來在講哪些檔案？ | `tigwx reverse` |
| 有什麼在等人決定？ | `tigwx status` |

**它能把這些分開，這是 Git 和單純的文件做不到的：**

- 人的決定和 Agent 的猜測。Agent 寫的句子會留在獨立的推測文件裡，等有人同意；透過 MCP，Agent 只能提議。
- 這一行對不對，以及這段程式碼是否屬於它。這是兩個問題，各有各的答案。
- 有意的修改，以及沒人解釋的修改。只有修改碰到的意圖才需要再看一次。
- 共用檔案的程式碼，以及真正相關的意圖。你可以明說：`depends_on`、`parallel`，或你自己的詞。

**它能替你做這些事：**

- **把專案提供給任何 Agent。** `tigwx mcp` 讓 Claude Code、Cursor 或任何 MCP 用戶端取得設計、足跡、封閉檢查，以及記錄新工作的工具。
- **留下忠實的足跡。** 每件工作記下它為了哪段設計、前後的專案狀態，以及每一步：每次修改動到的確切檔案、每條指令、每次檢查與決定。第二個 Agent 只拿到文件和足跡，就能再做出同樣的東西，tig 自己的測試就是這樣做的。
- **接上已經有程式碼的專案**，不用任何模型：`tigwx reverse` 讀出每一行指向什麼並提議，由你決定。
- **不靠伺服器就能合併。** 紀錄永不改變，以雜湊命名，所以兩台機器把檔案放在一起就合併了。當它們的決定不同，tig 會說出來，不會悄悄選一個。
- **只發布設計，不發布程式碼。** `tigwx push` 把文件和鏈送到大家能讀的 hub，絕不傳程式碼。
- **不擋你的路。** 所有東西都在 `.tig`，純文字，跟著專案進 Git。`init` 什麼都不問，也不改任何文件。tig 不執行任何東西。

**它不做什麼。** 它不跑檢查、不下判決，Agent 記錄的檢查只是 Agent 說的話。「封閉」表示文件、帳本和程式碼彼此一致，不表示程式碼是好的。搜尋是關鍵字搜尋。`by` 是寫入者自己的說法，沒有任何簽章，所以 tig 告訴你說過什麼，你的 Git 平台告訴你誰有辦法這樣說。

## 安裝

從 [Releases](https://github.com/h94u041/tig/releases) 取得你系統的安裝程式並執行，不需要其他東西。在 Windows 上會裝到 `%LOCALAPPDATA%\Programs\tigwx`，加入 PATH，並出現在已安裝程式清單中。其他系統會裝到 `~/.local/bin`，如果那裡不在 PATH 上，會告訴你要加哪一行。`-dir`、`-yes`、`-portable`、`-uninstall` 就是字面的意思。只含程式的壓縮檔和 `SHA256SUMS` 也在同一處。安裝程式還沒有簽章，所以 Windows 和 macOS 可能會在執行前警告你。

或是用 Go 1.26 以上：

```sh
go install github.com/h94u041/tig/cmd/tigwx@latest
```

`@latest` 是最新一個有標籤的發布版本；本頁跟著 `main`，之後新增了什麼請看 [changelog](../CHANGELOG.md)。hub 是另一個指令：`go install github.com/h94u041/tig/cmd/tighub@latest`。

## 五分鐘試用

建立一個資料夾，放兩個檔案。你的意圖 ID 會不一樣，因為它來自雜湊。

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

**開始，並說出每一行的意思。** `init` 什麼都不問，也不改任何文件。`mark` 在該行結尾寫一個小標記，在任何 Markdown 檢視器裡都看不見，並回覆意圖的 ID。`-confirm` 表示這就是你的意思。

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

**改變心意。** 把 `README.md` 改成 *a week*，標記留在原處。意圖保有它的身分，並得到新的修訂版本；因為沒有人說為什麼，應該有人再看一次。

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

**施工，並留下足跡。** Agent 會透過 MCP 做這件事；這裡是手動。在 `start` 和 `step` 之間，把 `src/cart.go` 改成 `168h`。

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

**把它們接起來，並做決定。** 這個 commit 跟著那一行，而那一行就是你的意思。`-source` 說明這個決定從哪裡來。

```console
$ tigwx mark -id 4086 "Carts are kept for a week."
i90e7  README.md:3  4086  Carts are kept for a week. Code: [cart](src/cart.go).
$ tigwx confirm -m "the shop stays open longer now" -source "ticket 12" i90e7
i90e7#2  confirmed  Carts are kept for a week.
$ tigwx check
closed: yes
```

**現在可以問它了。**

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

**說明兩個意圖之間的關係**，當它們共用程式碼但意思無關，或其中一個需要另一個時。tig 會保存這件事，不會從你要求的內容裡拿掉任何東西。

```console
$ tigwx relate -kind parallel -m "nothing in common" i90e7 i70fd
i90e7 parallel i70fd: confirmed
```

把 `.tig` 和專案其他部分一起加進 Git，它就跟著專案走。

## 搭配 Agent 使用

Claude Code：

```sh
claude mcp add tig -- tigwx mcp
```

或寫進任何 MCP 用戶端的設定：

```json
{ "mcpServers": { "tig": { "command": "tigwx", "args": ["mcp"] } } }
```

`tigwx mcp` 提供它啟動所在的專案（或 `-C dir`）。Agent 閱讀設計，為它要處理的部分開始一件工作，記錄每一步，並提議它的 commit 跟著哪些句子；你用 `tigwx confirm`、`tigwx reject`、`tigwx link`、`tigwx accept` 確認或拒絕。[工具清單與安全機制](../docs/mcp.md)。

如果專案已經有程式碼和文件，但沒有東西把它們接起來：

```sh
tigwx reverse           # 哪幾行看起來在講哪些檔案；它不寫入任何東西
tigwx reverse -write    # 提議它們；在你同意之前什麼都不會綁定
tigwx init -force       # 或一次說完：現有程式碼算一個 commit，每一行都跟著它
```

## 指令

`tigwx --help` 列出所有指令和旗標。除了 `mcp`，每個指令都接受 `-json`。在命令列上，沒有 `-by` 也沒有 `user.by` 時，終端機記錄為 `human`，其他情況記錄為 `unknown`：腳本應該說明自己代表誰。

| 指令 | 作用 |
|---|---|
| `init` | 在這個資料夾開始 tig（`-root` 指定主文件）；什麼都不問，不記錄 commit。`-force`：現有程式碼算一個 commit，每一行都跟著它 |
| `status` | 專案的狀況，以及什麼在等人決定 |
| `config` | 讀取或設定你的名字、你的 `by`、主文件，以及鏈外的路徑 |
| `snap`、`log`、`diff`、`restore` | 快照：建立、列出、比較、還原檔案。`log <intent>`：一個意圖發生過的所有事 |
| `blocks <file.md>` | 文件的文字區塊和它們的雜湊 |
| `chain` | 鏈上的每個區塊 |
| `show <區塊或參照>` | 一個區塊，或參照的意思：`i7f3a`、`i7f3a#2`、`lc3d4e5:5-14`、`a1b2`、`ev91c0` |
| `search <關鍵字>` | 包含所有關鍵字的區塊、工作、步驟和檔案 |
| `check` | 封閉報告 |
| `trail <區塊或檔案>` | 為它做過的工作；`-steps` 顯示每一步和它改過的檔案 |
| `work start`、`work step`、`work finish` | 手動留下足跡 |
| `commits`、`commit <id>` | commit 清單，最新的在前；單一 commit 的檔案與句子 |
| `mark`、`sync`、`unmark` | 說明一行的意思；讓帳本跟上修改過的行（`-m why`：這是有意的修改）；撤回已復原的 commit |
| `reverse`、`describe`、`link` | 不用模型找出哪幾行在講哪些檔案；說明一行在講既有檔案；決定程式碼是否屬於某個意圖 |
| `relate` | 兩個意圖的關係：`depends_on`、`part_of`、`parallel`，或你自己的詞 |
| `set`、`groups` | 一件工作需要什麼；什麼屬於同一組 |
| `confirm`、`reject`、`retire`、`accept` | 記錄對意圖或 commit 的決定 |
| `mcp` | 透過 stdin 和 stdout 以 MCP 提供這個 tig |
| `push [-token t] [-m note] [-insecure] <url>` | 把文件和鏈送到 hub |

區塊以內容雜湊命名；任何唯一且至少 8 個字元的前綴都可以。Go 程式可以直接使用 tig：`import "github.com/h94u041/tig"`。程式碼在 [cmd/tigwx](../cmd/tigwx/) 和 [cmd/tigwx-setup](../cmd/tigwx-setup/)；發布由 [scripts](../scripts/) 和 [.github](../.github/) 建置。

## 它的潛力在哪裡

程式碼正在變便宜；知道它為了什麼、而且確實成立，並沒有變便宜。tig 瞄準的是這裡：

- **人與 Agent 之間的契約。** 已確認的意圖是 Agent 可以依照施工的內容；它自己的猜測一直標示為猜測。「這次修改的每一行都有人同意嗎？」可以像今天的「測試過了嗎？」一樣把關合併。
- **活得比工作階段久的記憶。** 足跡是專案上每個 Agent 和人的共同狀態：做了什麼、為了哪個意圖、誰做的、有沒有人接受。下一個 Agent 從上一個停下的地方開始，而不是從零開始。
- **不靠官僚的可追溯性。** 安全關鍵和受監管的領域要求每條需求都能追溯到實作和測試證據，通常要靠笨重的工具才做得到。tig 用純文字、在 Git 裡提供同樣的追溯，並且隨著工作自然保持最新。
- **審查設計，而不是 diff。** 因為每個意圖都有不隨修改消失的身分，一次修改可以被審查成「這個意思變了，這些 commit 有跟上，或沒有」。
- **發布專案的承諾，而不發布程式碼。** 封閉實作的公開規格、讓每個學生的 Agent 照同樣情境施工的課程教材，以及稽核。

## 運作方式

所有東西都在專案的 `.tig` 資料夾：純文字、以內容定址的紀錄，跟著專案進 Git。tig 可以帶著走：把文件和足跡交給別人，他們的 Agent 就能再做出同樣的東西。

- **[足跡](../docs/footprint.md)。** 專案的快照（以內容定址的樹）、在上面做過的工作，以及讓金鑰不進去的機制。
- **[鏈](../docs/chain.md)。** 以內容雜湊命名的文字區塊、行標記、意圖與參照、意圖之間的關係、從主文件到為它建置的檔案的鏈，以及封閉規則。
- **[MCP](../docs/mcp.md)。** `tigwx mcp` 提供給 Agent 的工具和資源、Agent 如何使用，以及安全機制。
- **[Hub](../docs/hub.md)。** `tigwx push` 只把專案的文件和鏈送到 hub（`tighub`），大家可以像在 Git 平台上讀儲存庫一樣閱讀。
- **[格式](../docs/format.md)。** `.tig` 資料夾存放有版本的純文字物件和紀錄，其他工具也能讀寫。每個發布版本都讀得懂更早版本寫的內容，舊版本也讀得懂新版本寫的內容。
- **[建立在 tig 之上](../docs/tools.md)。** 從你自己的工具使用它：透過指令、Go，或直接讀寫檔案。

**能撐到多大。** 在 Windows 上實測：5,000 個意圖時，`check` 不到一秒，多標記一行約半秒。12,000 個檔案的專案（Go 標準函式庫），第一次快照不到 20 秒，之後 `check` 約 0.3 秒、`status` 約 3 秒。在 Windows 上，讀取剛寫入的檔案會因為防毒軟體檢查而比較慢。`.tig` 在磁碟上大約和專案一樣大，Git 能把它壓縮得很小。`TIG_SCALE=2000 go test -run TestScale -v .` ([scale_test.go](../scale_test.go)) 可以在你的機器上量測。

## 資安

tig 不執行任何東西、把路徑限制在專案內、以雜湊檢查紀錄，並讓看起來像金鑰的檔案不進入快照。它也清楚說明自己做不到什麼：依賴它之前，或要回報問題時，請讀 [SECURITY.md](../SECURITY.md)。**請從 v0.1.0 升級**，原因寫在那裡。

## 參與貢獻

歡迎提 issue 和 pull request。[CONTRIBUTING.md](../CONTRIBUTING.md) 說明什麼適合以及怎麼做。各版本的變更在 [changelog](../CHANGELOG.md)。

這個儲存庫本身就是一條封閉的鏈：它的 `.tig` 存著建置過程的足跡，驗收章節列出要手動執行的指令。

英文版 README 是來源。翻譯跟著它走，不在鏈內（`.tig/config` 的 `outside`）；內容不同時以英文版為準。

## 授權

MIT，請見 [LICENSE](../LICENSE)。以 Go 撰寫（[go.mod](../go.mod)、[go.sum](../go.sum)）。

## 驗收

- `go vet ./...`
- `go test ./...`
- `sh scripts/compat.sh <新版 tigwx> <tigwx 0.1.0>`
