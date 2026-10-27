# tig

**Markdown 即原始碼。** tig 讓專案的 Markdown 文件成為程式碼的唯一依據，以 MCP 伺服器的形式提供給任何 AI agent 據以建構，並忠實記錄每一次建構的足跡，讓工作可以被檢驗、被控管、被重現。

[![Go Reference](https://pkg.go.dev/badge/github.com/h94u041/tig.svg)](https://pkg.go.dev/github.com/h94u041/tig)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](https://opensource.org/licenses/MIT)

[English](../README.md) · **繁體中文** · [Deutsch](README.de.md) · [Français](README.fr.md) · [Русский](README.ru.md) · [日本語](README.ja.md)

> 本文是 [README.md](../README.md) 的翻譯。英文版是原本，也是 tig 檢查的那一份；兩者不一致時以英文版為準。

> 這個 tig 不是 [tig，Git 的文字介面工具](https://github.com/jonas/tig)。因此它的指令叫做 `tigprint`。

## 為什麼需要 tig

如今 agent 寫程式的速度，已經比任何人讀程式的速度還快。真正流失的不是程式碼，而是程式碼周圍的一切：

- **意圖。** Git 記錄哪幾行改了，卻不記錄這些修改是為了設計中的哪一部分。半年後，沒有人說得出某個檔案為什麼長這樣。
- **證據。** 設計文件說明「應該」成立什麼，卻沒有東西告訴你它「現在」是否仍然成立。文件會默默地和程式碼脫節；測試檢查的是程式碼，而不是設計的每一部分是否都被測到。
- **交接。** 每一次 agent 工作階段都從零開始。它重新讀一遍專案，猜哪些文件是最新的，然後重做另一個 agent 已經做過的事，甚至把它推翻。

tig 用一個想法補上這些缺口：設計以 Markdown 寫成，設計的每一部分都寫明如何驗收，而每一份工作都記錄在它所服務的那部分設計之下。

## tig 是什麼

- **你寫設計**，用一般的 Markdown：一份主文件連結到各份細節文件，每個部分都有一個 *Acceptance*（驗收）段落，列出證明它可行的指令。
- **Agent 依設計建構**，透過 MCP。它們閱讀並循著設計前進，查看之前是如何建構的，並把自己工作的每一步記錄在所服務的那部分設計之下。
- **tig 檢查結果。** 它告訴你是否每一部分設計都有驗收涵蓋（鏈是否*閉合*）、驗收是否通過，以及程式碼和設計在哪裡分道揚鑣。

一切都放在專案的 `.tig` 資料夾裡：純文字、以內容定址的紀錄，和專案一起進 Git。tig 是可攜的：把文件和足跡交給別人，他們的 agent 就能再建出同樣的東西。

## 它的潛力在哪裡

程式碼正在變便宜；知道它是為了什麼、以及它確實成立，並沒有變便宜。tig 瞄準的正是這裡：

- **人與 agent 之間的契約。** 一條閉合的鏈，是 agent 能執行、人能驗證的規格：direction（方向）說明要建什麼，guideline（準則）說明已經建好且成立的是什麼。「鏈閉合了嗎？」可以像今天的「測試通過了嗎？」一樣，作為合併的關卡。
- **比一次工作階段更長久的記憶。** 足跡是專案上每個 agent 和每個人共享的狀態：做了什麼、為了哪部分設計、誰做的、有沒有通過。下一個 agent 從上一個停下的地方接手，而不是從零開始。
- **不靠繁文縟節的可追溯性。** 安全關鍵與受監管的領域要求每一項需求都能追溯到實作和測試證據，通常得靠笨重的工具來付出代價。tig 以純文字、在 Git 裡提供同樣的追溯，並且隨著工作自然保持最新。
- **審查設計，而不是審查 diff。** 因為每個設計區塊都以內容命名、並在編輯中被追蹤，一次修改可以被審查為「設計的這一部分改了，這些檔案跟上了，或沒有跟上」。
- **公開專案承諾了什麼、證明了什麼，而不公開程式碼。** 透過 hub，團隊可以展示設計以及哪些部分已被驗證，而實作保持私有：封閉實作的公開規格、課程教材、稽核。

tig 不做的事：驗收的強度取決於它的指令；在 `loose` 模式下，tig 相信 agent 回報的檢查結果（重要時請用 `strict`）。搜尋是關鍵字搜尋。tig 指出設計和程式碼在哪裡不一致；決定哪一邊才對，仍然是你的事。

## 功能

- **為你的專案提供 MCP 伺服器。** `tigprint mcp` 把文件和足跡提供給任何 MCP 用戶端（Claude Code、Cursor 等）：總覽、關鍵字搜尋、閱讀並循著設計區塊前進、每個區塊或檔案的足跡、閉合檢查，以及記錄新工作的工具。
- **文件串成一條鏈。** 主文件、它的細節文件，以及它們連結到的檔案，被當作一個整體來讀。每個文字區塊以其內容雜湊命名，tig 依固定、可重算的規則，追蹤每個區塊經過的編輯、移動、拆分與合併。
- **忠實的足跡。** 每一份 *work*（一次建構，類似一個 commit）記錄它所服務的設計、它的意圖、建構前後的專案，以及它的 *steps*：每一次編輯和它確切改動的檔案、每一個指令、檢查與決定。
- **agent 層級的重現。** 足跡完整到足以讓第二個 agent 只拿著文件和足跡，就重建出第一個 agent 建出的東西。tig 自己的測試就是這樣做的。
- **閉合檢查。** 失效的連結、沒有驗收涵蓋的設計、在足跡之外被修改或因設計變更而被遺留的程式碼，以及沒有任何設計觸及的檔案。被涵蓋的設計在驗收通過前是 *direction*，通過後成為 *guideline*。
- **控管執行的內容。** tig 只執行寫在驗收段落中的指令；透過 MCP 時，只有在你以 `-run` 啟動伺服器時才會執行。在 `strict` 模式下，只有 tig 自己執行的檢查才算數；agent 無法只是回報成功了事。
- **可以發布的 hub。** `tigprint push` 把文件和鏈（絕不包含程式碼）送到 `tighub`；hub 會檢查每一次推送，並逐區塊顯示每份文件及其狀態。
- **安全且可攜。** 每次讀取都會比對物件和它的雜湊，來自他人 tree 的路徑無法跳出專案，同時寫入的程序也不會遺失紀錄。格式是純文字、有版本，並且打算長久使用。

## 使用情境

- **教學與作業。** 把一份閉合的設計當作 MCP 伺服器發給學生。每個學生的 agent 都依同樣的情境建構，每份作業都是一份足跡，其驗收可以被檢查，在 strict 模式下由 tig 親自執行。
- **追蹤與控管 agent 的工作。** agent 的每一個動作都記錄在它所要服務的設計之下：做了什麼、誰做的、為什麼、有沒有通過。
- **多 agent 協作。** agent 共享同一份設計與同一份足跡，因此每個 agent 都看得到其他 agent 建了什麼，並從它們停下的地方接手。
- **始終真實的設計。** 當設計改了而程式碼沒跟上，或程式碼在任何紀錄之外被修改時，tig 會指出來。
- **展示設計而不交出程式碼。** 推送到 hub，讓審查者、客戶或學生閱讀設計、看到哪些部分已被驗證，而不需要存取原始碼。

## 安裝

需要 Go 1.25 或更新版本：

```sh
go install github.com/h94u041/tig/cmd/tigprint@latest
```

hub 是另一個指令：`go install github.com/h94u041/tig/cmd/tighub@latest`。

## 快速開始

在一個主文件為 `README.md` 的專案中：

```sh
tigprint init              # 在這裡建立 .tig
tigprint chain             # 鏈上的每個文字區塊及其狀態
tigprint check             # 鏈閉合了嗎？（未閉合時結束碼為 1）
tigprint check -run-all    # 讓 tig 親自執行每個驗收段落
```

標題為 `Acceptance`（或 `驗收`）的段落，列出驗收它所在那部分文件的指令：

````markdown
## Turning

How a layer of the cube turns. Code: [turn.js](src/turn.js).

### Acceptance

- `npm test -- turn`
````

接著連接你的 agent。以 Claude Code 為例：

```sh
claude mcp add tig -- tigprint mcp
```

或寫在任何 MCP 用戶端的設定中：

```json
{ "mcpServers": { "tig": { "command": "tigprint", "args": ["mcp"] } } }
```

`tigprint mcp` 服務它啟動時所在的專案（或用 `-C dir` 指定）。加上 `-run` 可讓 agent 請 tig 執行驗收指令。

## 運作方式

tig 有三個部分，加上一個可以發布的 hub，各有自己的文件（英文）。

### 足跡

專案的快照（以內容定址的 tree），以及在專案上完成的 works，每一份由 steps 組成。見 [footprint](../docs/footprint.md)。

### 鏈

以內容雜湊命名的文字區塊、它們在快照之間的演變、從主文件到為其建構的檔案的鏈，以及閉合規則。見 [chain](../docs/chain.md)。

### MCP

`tigprint mcp` 提供給 agent 的工具與資源、agent 如何使用它們，以及保障安全的機制。見 [MCP 伺服器](../docs/mcp.md)。

### Hub

`tigprint push` 把專案的文件和鏈（僅此而已）送到 hub，也就是 `tighub`，讓人們像在 Git 平台上閱讀儲存庫一樣閱讀它們。見 [hub](../docs/hub.md)。

## 指令

單獨執行 `tigprint` 會列出所有指令及其選項。程式碼在 [cmd/tigprint](../cmd/tigprint/)。

| 指令 | 作用 |
|---|---|
| `init` | 在此資料夾建立 tig（`-root` 主文件，`-strict` 只計入 tig 執行的檢查） |
| `snap`、`log`、`diff`、`restore` | 快照：拍一張、列出、比較、把檔案還原 |
| `blocks <file.md>` | 一份文件的文字區塊及其雜湊 |
| `chain` | 鏈上的每個區塊及其狀態 |
| `show <block>` | 單一區塊：內容、狀態、細節、為它建構的檔案及其 works |
| `search <words>` | 包含所有關鍵字的區塊、works、steps 與檔案 |
| `check` | 閉合報告；`-run <block>` 或 `-run-all` 先執行驗收；`-json` |
| `trail <block or file>` | 為它所做的 works；`-steps` 顯示每一步及其改動的檔案 |
| `work start`、`work step`、`work finish` | 手動留下足跡 |
| `mcp` | 透過 stdin 和 stdout 以 MCP 提供這個 tig |
| `push [-token t] [-m note] <url>` | 把文件和鏈送到 hub |

區塊以其內容雜湊命名；任何 8 個字元以上、不重複的前綴都可以使用。Go 程式可以直接使用 tig：`import "github.com/h94u041/tig"`。

## 格式

`.tig` 資料夾存放純文字的物件與紀錄，並有版本，因此其他工具也能讀寫。見 [format](../docs/format.md)。

## 狀態

第一個版本，0.1。`.tig` 格式有版本並打算長久使用；指令與 MCP 工具仍可能變動。這個儲存庫本身就是一條閉合的鏈：它的 `.tig` 保存了它如何被建構的足跡，而且每個驗收段落在 tig 執行時都會通過。

## 授權

MIT，見 [LICENSE](../LICENSE)。以 Go 撰寫（[go.mod](../go.mod)、[go.sum](../go.sum)）。

## 驗收

- `go vet ./...`
- `go test ./...`
