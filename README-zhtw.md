# Dev.to Blog Translation Tool

[English](README.md) | [繁體中文](README-zhtw.md)

一個命令列工具，使用 Google Gemini AI 把你在 Dev.to 上的草稿文章，從任何語言自動翻譯成英文。

## 功能特色

- 🔍 自動抓取 Dev.to 帳號中的**所有**草稿文章（支援分頁，300 篇以上也沒問題）
- 🌐 使用 Gemini 3 Flash 翻譯文章標題與內文
- 🧠 智慧語言偵測 — 標題已經是英文的文章會自動跳過
- 📝 保留 markdown 格式、程式碼區塊與技術名詞
- 🧹 翻譯前自動移除 Jekyll/Hugo frontmatter 中繼資料
- ✂️ 翻譯後的標題會截斷至 Dev.to 的 128 字元上限
- 🚀 翻譯完成後自動發布到 Dev.to
- 🔒 以環境變數安全管理 API 金鑰
- ⚡ 具備速率限制，且請求可被取消
- 📊 詳細的進度回報與統計資訊

## 環境需求

- Go 1.24 以上（版本宣告於 `go.mod`）
- 具備 API 存取權限的 Dev.to 帳號
- Google Gemini API 金鑰

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

## 取得 API 金鑰

### Dev.to API 金鑰

1. 前往 [Dev.to Settings > Extensions](https://dev.to/settings/extensions)
2. 產生一組新的 API 金鑰
3. 把金鑰複製到 `.env`

### Google Gemini API 金鑰

1. 前往 [Google AI Studio](https://aistudio.google.com/app/apikey)
2. 建立一組新的 API 金鑰
3. 把金鑰複製到 `.env`

## 使用方式

執行翻譯工具：

```bash
go run ./cmd/translator
```

工具的執行流程：
1. 顯示確認提示，說明接下來會做什麼
2. 從 `.env` 載入 API 金鑰
3. 抓取 Dev.to 帳號中的**所有**草稿文章（支援分頁）
4. 逐篇處理草稿：
   - 顯示文章 ID 與原始標題
   - 檢查標題是否已經是英文（是的話跳過）
   - 使用 Gemini 把標題與內文翻譯成英文
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

### 執行結果範例

```
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
│       └── go.yml               # CI：push / PR 時建置與測試
├── cmd/
│   └── translator/
│       └── main.go              # 程式進入點
├── internal/
│   ├── devto/
│   │   └── client.go            # Dev.to API client（分頁與發布）
│   ├── gemini/
│   │   └── client.go            # 負責翻譯的 Gemini API client
│   ├── translator/
│   │   └── translator.go        # 翻譯服務邏輯
│   └── utils/
│       ├── language.go          # 語言偵測工具
│       └── markdown.go          # Markdown / frontmatter 處理
├── devto-article-translator/    # 打包成 skill 的版本（獨立的 Go module）
├── .env                         # 你的 API 金鑰（絕對不要 commit！）
├── .env.example                 # 環境變數範本
├── .gitignore                   # Git 忽略設定
├── go.mod                       # Go module 檔
├── test_security.sh             # 安全性檢查腳本
└── README.md                    # 英文版說明
```

> **注意：** `devto-article-translator/` 底下放的是同一支程式的打包副本，有自己的
> `go.mod`，所以從 repository 根目錄執行 `go build ./...` 並不會建置到它。把兩份
> 副本合併已列在 roadmap 中。

## 翻譯行為說明

### 語言偵測

由 `utils.IsEnglish` 判斷一篇文章需不需要翻譯。只要標題含有任何 CJK 字元（漢字、平假名、片假名、諺文），就視為**非**英文；否則當標題的字母與數字有 80% 以上是 ASCII 拉丁字元時，才算是英文。

判斷時**只看標題**。標題是英文的話，整篇文章會直接跳過，內文完全不會被翻譯。

### 標題長度

Dev.to 不接受超過 128 字元的標題。翻譯後的標題會截斷至 128 個 rune（不是 byte，所以不會把多位元組字元切壞），發生截斷時會印出警告。

### Token 預算

Gemini 3 Flash 是 thinking model：模型內部推理所用的 token 也會計入 `MaxOutputTokens`。因此這裡把上限開得很寬 — 內文 65,536、標題 4,096 — 因為預算抓太緊會被推理吃光，導致輸出在中途被截斷。計費是以實際用掉的 token 計算，所以把上限開高不會多花錢。

如果回應仍然卡在上限，工具會偵測到 `FinishReasonMaxTokens` 並回報截斷錯誤，而不是把翻譯到一半的文章發布出去。

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
go test ./...    # 測試
```

> **注意：** 這個專案目前還沒有測試。`go test ./...` 現在會直接成功結束，因為根本
> 沒有測試檔案。補上測試是 roadmap 的第一項工作。

## 疑難排解

### 「DEVTO_API_KEY environment variable is not set」

請確認你已經在專案根目錄建立 `.env`，並填入 API 金鑰。

### 「failed to fetch draft articles: API request failed with status 401」

你的 Dev.to API 金鑰可能無效或已過期。請到 Dev.to 設定頁重新產生一組。

### 「failed to generate translation」

請確認 Gemini API 金鑰有效，且沒有超過速率限制。

### 「translation truncated: hit MaxOutputTokens」

文章太長，模型的推理加上輸出把 token 預算用完了。可以調高 `internal/gemini/client.go` 裡的 `SetMaxOutputTokens`，或是先把文章拆短再翻譯。

### 「API request failed with status 422」

Dev.to 拒絕了這次的更新內容。最常見的原因是標題超過 128 字元，或是 tag 不合法。請查看錯誤訊息中隨狀態碼一起印出的回應內容。

## 相依套件

- [godotenv](https://github.com/joho/godotenv) — 環境變數管理
- [generative-ai-go](https://github.com/google/generative-ai-go) — Google Gemini AI SDK

## 授權

MIT License

## 參與貢獻

歡迎貢獻，請放心送 Pull Request。

## 作者

為了有效率地翻譯 Dev.to 部落格文章而寫。
