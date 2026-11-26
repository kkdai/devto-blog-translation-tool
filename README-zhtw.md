# Dev.to Blog Translation Tool

[English](README.md) | [繁體中文](README-zhtw.md)

一個命令列工具，使用 Google Gemini AI 把你在 Dev.to 上的草稿文章，從任何語言自動翻譯成英文。

## 功能特色

- 🔍 自動抓取 Dev.to 帳號中的**所有**草稿文章（支援分頁，300 篇以上也沒問題）
- 🌐 使用 Gemini 4 Neon（`gemini-4-neon`，可由 `GEMINI_TEXT_MODEL` 覆寫）翻譯文章標題與內文
- 🖼️ 選用圖片翻譯 — 自動偵測文章中含中文的圖片，使用 Gemini 重新生成英文版、與原圖比對驗證後上傳到 Google Cloud Storage 並替換圖片網址
- 🧠 智慧語言偵測 — 標題已經是英文的文章會自動跳過
- 📝 保留 markdown 格式、程式碼區塊與技術名詞
- 🧹 翻譯前自動移除 Jekyll/Hugo frontmatter 中繼資料
- ✂️ 自動將翻譯後的標題控制在 Dev.to 的 128 字元上限內（優先請 Gemini 縮短，失敗時於單字邊界截斷）
- 🚀 翻譯完成後自動發布到 Dev.to
- 🔒 以環境變數安全管理 API 金鑰
- ⚡ 具備速率限制，且請求可被取消
- 📊 詳細的進度回報與統計資訊

## 環境需求

- Go 1.26 以上（版本宣告於 `go.mod`）
- 具備 API 存取權限的 Dev.to 帳號
- Google Gemini API 金鑰
- *（選用，圖片翻譯功能）* Google Cloud SDK（`gcloud` CLI）與已建立 Cloud Storage bucket 的 GCP 專案

## 安裝

1. 複製這個 repository：
```bash
git clone https://github.com/kkdai/devto-blog-translation-tool.git
cd devto-blog-translation-tool
```

2. 安裝相依套件：
```bash
go mod download
```

3. 設定環境變數：
```bash
cp .env.example .env
```

4. 編輯 `.env`，填入你的 API 金鑰：
```env
DEVTO_API_KEY=your_devto_api_key_here
GEMINI_API_KEY=your_gemini_api_key_here
```

## 環境變數設定

所有設定都可以寫在專案根目錄的 `.env` 檔，或直接設在 shell 環境變數中。程式透過 `godotenv` 載入 `.env`，**不會覆蓋** shell 中已存在的環境變數。

| 變數名稱 | 必填 | 預設值 | 說明 |
| --- | --- | --- | --- |
| `DEVTO_API_KEY` | 是 | — | Dev.to API 金鑰，用於抓取草稿與更新發布 |
| `GEMINI_API_KEY` | 是 | — | Google Gemini API 金鑰 |
| `GEMINI_TEXT_MODEL` | 否 | `gemini-4-neon` | 用於文章翻譯、標題縮短、圖片中文偵測與圖片驗證的 Gemini 模型 |
| `GCS_BUCKET` | 否 | *（未設定則停用）* | 存放翻譯後圖片的 GCS bucket 名稱；未設定時會停用圖片翻譯功能 |
| `GEMINI_IMAGE_MODEL` | 否 | `gemini-3-pro-image` | 用於生成英文版圖片的 Gemini 圖片模型 |
| `GEMINI_IMAGE_SIZE` | 否 | `2K` | 翻譯後圖片的輸出解析度（可選 `1K`、`2K` 或 `4K`） |
| `IMAGE_MAX_CHINESE_CHARS` | 否 | `60` | 估計中文字數超過此門檻的圖片直接跳過重畫 |

## 取得 API 金鑰

### Dev.to API 金鑰

1. 前往 [Dev.to Settings > Extensions](https://dev.to/settings/extensions)
2. 產生一組新的 API 金鑰
3. 把金鑰複製到 `.env`

### Google Gemini API 金鑰

1. 前往 [Google AI Studio](https://aistudio.google.com/app/apikey)
2. 建立一組新的 API 金鑰
3. 把金鑰複製到 `.env`

### Google Cloud Storage（選用 — 含中文圖片翻譯）

若未設定 `GCS_BUCKET`，圖片翻譯功能會自動停用，其餘翻譯流程完全照舊。若要啟用：

1. 使用 Application Default Credentials 完成本機認證：
   ```bash
   gcloud auth application-default login
   ```
2. 建立 GCS bucket 並設為公開可讀（支援 uniform bucket-level access，程式不會設定單一物件 ACL）：
   ```bash
   gcloud storage buckets create gs://YOUR_BUCKET_NAME --uniform-bucket-level-access
   gcloud storage buckets add-iam-policy-binding gs://YOUR_BUCKET_NAME \
     --member=allUsers --role=roles/storage.objectViewer
   ```
3. 在 `.env` 中加入 `GCS_BUCKET`（以及其他選用圖片設定）：
   ```env
   GCS_BUCKET=YOUR_BUCKET_NAME
   # 選用覆寫設定：
   GEMINI_TEXT_MODEL=gemini-4-neon
   GEMINI_IMAGE_MODEL=gemini-3-pro-image
   GEMINI_IMAGE_SIZE=2K
   IMAGE_MAX_CHINESE_CHARS=60
   ```

完整的處理流程與門檻調整建議請見下方[圖片翻譯](#圖片翻譯)章節。

## 使用方式

執行翻譯工具：

```bash
go run ./cmd/translator
```

工具的執行流程：
1. 從 `.env` 與環境變數載入 API 金鑰與設定
2. 顯示確認提示，說明接下來會做什麼
3. 抓取 Dev.to 帳號中的**所有**草稿文章（支援分頁）
4. 逐篇處理草稿：
   - 顯示文章 ID 與原始標題
   - 檢查標題是否已經是英文（是的話跳過）
   - 使用 Gemini 翻譯標題（若超過 128 字元會自動縮短）
   - 移除 Jekyll/Hugo frontmatter（若有）
   - *（僅設定 `GCS_BUCKET` 時）* 掃描並翻譯含中文的圖片、上傳至 GCS 並替換文章內的圖片網址
   - 使用 Gemini 把內文翻譯成英文
   - **更新原本那篇草稿**，寫入翻譯後的內容
   - 發布更新後的文章
5. 顯示統計摘要：
   - 已翻譯並發布的文章數
   - 跳過的文章數（原本就是英文）
   - 翻譯失敗的文章數

### ⚠️ 重要注意事項

1. **會更新既有草稿**：這個工具是**更新你現有的草稿文章**，不會建立新文章
2. **使用 PUT 請求**：透過 `PUT /articles/:id` 把翻譯內容寫回同一篇文章
3. **自動發布**：翻譯完成後，文章會被自動發布
4. **需要確認**：開始執行前工具會先詢問你
5. **顯示文章 ID**：每篇處理中的文章都會印出 ID，方便你確認改到的是正確的文章
6. **移除 frontmatter**：翻譯前會自動移除 Jekyll/Hugo frontmatter（`---` 之間的中繼資料），避免把中繼資料也翻掉
7. **一次跑完全部**：目前沒有 dry-run 模式，也沒有指定單篇文章的參數。工具會一次處理所有非英文的草稿
8. **圖片會上傳至公開的 GCS bucket**：設定 `GCS_BUCKET` 時，翻譯後的圖片會上傳到公開可讀的 GCS bucket，讓 Dev.to 文章可以正常顯示

### 執行結果範例

```
✅ API keys loaded successfully
✅ Gemini text model: gemini-4-neon
ℹ️  GCS_BUCKET not set; Chinese image translation is disabled

⚠️  WARNING ⚠️
=====================================
This tool will:
1. Fetch ALL your draft articles from Dev.to
2. Translate non-English articles to English
3. UPDATE the existing draft articles with translations
4. PUBLISH the updated articles

Note: This will UPDATE your existing drafts,
      NOT create new articles.
=====================================

Do you want to continue? (yes/no): yes

✅ Starting translation and publishing process...

🔍 Fetching draft articles from Dev.to...
  Fetched page 1 (30 articles, total: 30)
  Fetched page 2 (30 articles, total: 60)
  ...

✅ Found 300 draft article(s)

[1/300] Article ID: 2457408
  Original Title: 使用 Gemini 3.0 Pro Image API 打造 PDF 文字優化工具
  Translating title... ✅
  Translated Title: Building a PDF Text Optimization Tool with Gemini 3.0 Pro Image API
  Translating content... (removing frontmatter) ✅
  Updating article ID 2457408 and publishing... ✅ Published!

[2/300] Article ID: 2457409
  Original Title: Introduction to Machine Learning
  ⏭️  Skipping: Title is already in English

...

========================================
SUMMARY
========================================
✅ Translated & Published: 250
⏭️  Skipped (English): 45
❌ Failed: 5
📊 Total processed: 300
========================================

🎉 Process completed!
```

## 專案結構

```
devto-blog-translation-tool/
├── .github/
│   └── workflows/
│       └── go.yml                   # CI：push / PR 時建置與測試
├── cmd/
│   └── translator/
│       └── main.go                  # 程式進入點
├── internal/
│   ├── devto/
│   │   └── client.go                # Dev.to API client（分頁與發布）
│   ├── gemini/
│   │   ├── client.go                # Gemini 文字翻譯與標題縮短 client
│   │   ├── client_test.go           # Gemini 文字 client 單元測試
│   │   ├── image.go                 # Gemini 圖片中文偵測、生成與驗證
│   │   └── image_test.go            # Gemini 圖片功能單元測試
│   ├── images/
│   │   ├── markdown.go              # Markdown / HTML <img> 網址擷取與替換
│   │   ├── markdown_test.go         # 圖片網址解析單元測試
│   │   ├── processor.go             # 單張圖片翻譯完整流程處理器
│   │   └── processor_test.go        # 圖片處理器單元測試
│   ├── storage/
│   │   └── gcs.go                   # Google Cloud Storage 上傳器
│   ├── translator/
│   │   ├── translator.go            # 翻譯服務主流程
│   │   └── translator_test.go       # 翻譯服務單元測試
│   └── utils/
│       ├── language.go              # 語言偵測工具
│       ├── markdown.go              # Markdown / frontmatter 處理
│       ├── title.go                 # 單字邊界標題截斷工具
│       └── title_test.go            # 標題截斷單元測試
├── devto-article-translator/        # 打包成 skill 的版本（獨立的 Go module）
├── .env                             # 你的 API 金鑰（絕對不要 commit！）
├── .env.example                     # 環境變數範本
├── .gitignore                       # Git 忽略設定
├── go.mod                           # Go module 檔
├── test_security.sh                 # 安全性檢查腳本
├── README.md                        # 英文版說明
└── README-zhtw.md                   # 繁體中文版說明
```

> **注意：** `devto-article-translator/` 底下放的是同一支程式的打包副本，有自己的
> `go.mod`，所以從 repository 根目錄執行 `go build ./...` 並不會建置到它。把兩份
> 副本合併已列在 roadmap 中。

## 翻譯行為說明

### 語言偵測

由 `utils.IsEnglish` 判斷一篇文章需不需要翻譯。只要標題含有任何 CJK 字元（漢字、平假名、片假名、諺文），就視為**非**英文；否則當標題的字母與數字有 80% 以上是 ASCII 拉丁字元時，才算是英文。

判斷時**只看標題**。標題是英文的話，整篇文章會直接跳過，內文完全不會被翻譯。

### 標題長度

Dev.to 不接受超過 128 字元的標題。翻譯後的標題若超過 128 個 rune，工具會先請 Gemini（`ShortenTitle`）協助縮短，保留開頭的方括號標籤（例如 `[Notes]`、`[Tech]`）與核心資訊，最多嘗試 **2 次**。

若縮短失敗或結果仍超過 128 個 rune，才會退回以 `utils.TruncateAtWordBoundary` 在單字邊界截斷，並清除結尾多餘的空白或標點（`,`、`:`、`;`、`-`、`–`、`—`）；只有在 128 個 rune 內完全沒有空白時才會硬切。執行時會印出對應提示（`✂️` 表示縮短，`⚠️` 表示於單字邊界截斷）。

### Token 預算

Gemini 4 Neon（`gemini-4-neon`）是 thinking model：模型內部推理所用的 token 也會計入 `MaxOutputTokens`。因此上限開得相當寬裕 — **內文 65,536**、**標題翻譯與標題縮短 16,384** — 以免推理吃光預算導致輸出中途被截斷。計費是以實際用掉的 token 計算，所以把上限開高不會多花錢。

如果回應仍然卡在 `FinishReasonMaxTokens`（或回傳空內容），工具會自動重試一次；第二次仍失敗才回報截斷錯誤，而不是把翻譯到一半的文章發布出去。

## 圖片翻譯

設定 `GCS_BUCKET` 後，每篇非英文文章會在**翻譯內文之前**執行圖片翻譯流程，依序處理每張不重複的圖片：

```
擷取網址 ─▶ 下載圖片 ─▶ 偵測中文 ─▶ 生成英文版 ─▶ 原圖與新圖比對驗證 ─▶ 上傳 GCS 並替換網址
```

1. **擷取圖片網址**：掃描 markdown 的 `![alt](url)` 與 HTML 的 `<img src="url">`，自動跳過 fenced code block、inline code、非 `http(s)` 網址與重複網址。
2. **下載圖片**：套用 **30 秒逾時**與 **20 MB** 檔案上限，支援 `image/png`、`image/jpeg`、`image/webp`；`image/gif` 會自動跳過。
3. **偵測中文與估計字數**：送給文字模型（`GEMINI_TEXT_MODEL`）判斷是否含中文並估計中文字數：
   - 不含中文 → 跳過，保留原圖。
   - 估計中文字數超過 `IMAGE_MAX_CHINESE_CHARS`（預設 `60`）→ 跳過，保留原圖（文字過於密集的資訊圖或架構圖重畫容易出現亂碼或錯字）。
4. **生成英文版圖片**：交由圖片模型（`GEMINI_IMAGE_MODEL`，預設 `gemini-3-pro-image`）以 `GEMINI_IMAGE_SIZE`（`1K`、`2K` 或 `4K`）解析度重繪英文版，並自動選擇最接近原圖長寬比的支援比例（`1:1`、`2:3`、`3:2`、`3:4`、`4:3`、`9:16`、`16:9`、`21:9`）。
5. **原圖與新圖比對驗證**：將原圖與生成圖一併交給文字模型（`GEMINI_TEXT_MODEL`）檢查，確認已無中文且保留所有主要 UI 元素、面板、圖表與資訊；若仍有中文、缺少主要內容或驗證失敗，則保留原圖。
6. **上傳 GCS 並替換網址**：上傳至 GCS 路徑 `devto-images/<articleID>/<sha256前16碼>-en.<ext>`，將文章內的原網址替換為公開網址（`https://storage.googleapis.com/<bucket>/...`），接著再進行內文翻譯。

任何單張圖片處理失敗時，程式只會印出警告並保留原圖，不會中斷整篇文章的翻譯與發布。

### 執行輸出範例

```
  🖼️  Image 1/4: https://example.com/diagram-en.png ... ⏭️  no Chinese text
  🖼️  Image 2/4: https://example.com/ui-screenshot.png ... ✅ replaced
  🖼️  Image 3/4: https://example.com/infographic.png ... ⏭️  too much text (~390 Chinese chars), keeping original
  🖼️  Image 4/4: https://example.com/settings.png ... ⚠️  generated image lost content (sidebar panel missing), keeping original
```

### 門檻調整與成本提醒

- **門檻調整建議（`IMAGE_MAX_CHINESE_CHARS`）**：預設值 `60` 較為保守，以降低重繪出現亂碼的機率；若是文字量適中的 UI 截圖，可調高至 `100`–`150`，但偶爾可能會出現些許小錯字。
- **成本與時間提醒**：每張含中文（且未超過門檻）的圖片至少會觸發 **3 次 Gemini API 呼叫**（偵測、生成、驗證），處理多圖文章時請留意執行時間與 API 費用。

## Frontmatter 處理

工具會在翻譯前自動偵測並移除 Jekyll/Hugo frontmatter，避免中繼資料也被翻譯。

**範例：**

含有 frontmatter 的原始內容：
```markdown
---
title: [VS Code][Colab] Google Officially Releases Colab VS Code Plugin
published: false
date: 2025-11-14 00:00:00 UTC
tags:
canonical_url: https://www.evanlin.com/colab-vscode-plugin/
---

# Introduction

This is the actual article content...
```

工具會：
1. 偵測 frontmatter（`---` 之間的內容）
2. 在翻譯前把它移除
3. 只翻譯真正的文章內容：
```markdown
# Introduction

This is the actual article content...
```

**支援的格式：**
- 標準 frontmatter：`--- ... ---`
- 程式碼區塊形式：` ``` ... ``` `

## 安全性

🔒 **重要**：這個專案使用環境變數存放敏感的 API 金鑰。

- **絕對不要**把 `.env` commit 進 Git
- `.env` 已經列在 `.gitignore` 裡
- 給其他開發者參考時，請用 `.env.example` 當範本
- 程式不會把 API 金鑰寫進 log 或印在輸出中

執行內建的檢查腳本，確認金鑰沒有外洩到被追蹤的檔案裡：

```bash
./test_security.sh
```

## 建置

編譯成獨立執行檔：

```bash
go build -o translator ./cmd/translator
```

接著執行：
```bash
./translator
```

## 開發

```bash
go build ./...   # 編譯
go vet ./...     # 靜態檢查
gofmt -l .       # 格式檢查（不該印出任何東西）
go test ./...    # 執行單元測試
```

現有單元測試涵蓋 `internal/gemini`、`internal/images`、`internal/translator` 與 `internal/utils`，皆以 mock 與本地測試資料為主，不會呼叫外部 API，不需要設定 API 金鑰或 GCS 憑證即可直接執行。

## 疑難排解

### 「DEVTO_API_KEY environment variable is not set」

請確認你已經在專案根目錄建立 `.env` 並填入 API 金鑰（或已於 shell 設定環境變數）。

### 「failed to fetch draft articles: API request failed with status 401」

你的 Dev.to API 金鑰可能無效或已過期。請到 Dev.to 設定頁重新產生一組。

### 「failed to generate translation」

請確認 Gemini API 金鑰有效，且沒有超過速率限制。

### 「models/... is not found for API version v1」

圖片的中文偵測、生成與驗證呼叫已明確使用 `v1beta`（`google.golang.org/genai`）。若你透過 `GEMINI_TEXT_MODEL` 或 `GEMINI_IMAGE_MODEL` 自行指定其他模型，請確認該模型在 `v1beta` API 中可用。

### 「translation truncated: hit MaxOutputTokens」

遇到 `MaxOutputTokens` 時工具會自動重試一次。若兩次嘗試仍耗盡 token 預算，請調高 `internal/gemini/client.go` 裡的 `SetMaxOutputTokens`，或是先把文章拆短再翻譯。

### GCS：「could not find default credentials」

在啟用 `GCS_BUCKET` 前，請先執行 `gcloud auth application-default login` 建立本機的 Application Default Credentials。

### GCS：上傳成功但圖片打不開

GCS bucket 尚未開啟公開讀取權限。請賦予 `allUsers` `roles/storage.objectViewer` 角色：

```bash
gcloud storage buckets add-iam-policy-binding gs://YOUR_BUCKET_NAME \
  --member=allUsers --role=roles/storage.objectViewer
```

### 圖片全部都被跳過（`too much text`）

請檢查 `IMAGE_MAX_CHINESE_CHARS` 設定（預設 `60`）。若文章中的圖片多為字數較多的 UI 截圖，可嘗試調高至 `100`–`150`。

### 「API request failed with status 422」

Dev.to 拒絕了這次的更新內容。最常見的原因是標題超過 128 字元，或是 tag 不合法。請查看錯誤訊息中隨狀態碼一起印出的回應內容。

## 相依套件

- [godotenv](https://github.com/joho/godotenv) — 環境變數管理
- [generative-ai-go](https://github.com/google/generative-ai-go) — Google Gemini AI SDK（文字翻譯與標題縮短）
- [google.golang.org/genai](https://pkg.go.dev/google.golang.org/genai) — Google Gen AI SDK（圖片中文偵測、生成與驗證）
- [cloud.google.com/go/storage](https://pkg.go.dev/cloud.google.com/go/storage) — Google Cloud Storage client（上傳翻譯後圖片）

## 授權

MIT License

## 參與貢獻

歡迎貢獻，請放心送 Pull Request。

## 作者

為了有效率地翻譯 Dev.to 部落格文章而寫。
