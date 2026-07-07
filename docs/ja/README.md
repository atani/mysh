<p align="center">
  <img src="../../logo.png" alt="mysh logo" width="180">
</p>

<h1 align="center">mysh</h1>

<p align="center"><a href="../../README.md">English</a> | <b>日本語</b></p>

**本番データを AI コーディングエージェントに漏らさないための MySQL CLI です。**

mysh は AI コーディング時代に向けた、より安全な MySQL CLI です。接続プロファイル・SSH トンネル・クエリ出力を一括で管理します。Claude Code や Cursor、シェルスクリプト、MCP クライアントが本番相当のデータベースを扱うときも、生の機微データをそのまま渡しません。

*読みは「マイ・シュ」（/maɪʃ/）。"my shell" のもじりです。*

![demo](../../demo.gif)

> **はじめての方へ**: [非エンジニア向けのはじめにガイド](getting-started.md) を参照してください。5 分でセットアップし、Claude Code からクエリを実行できます。

## なぜ mysh なのか

従来の MySQL クライアントは人間が直接結果を読む前提で作られています。AI を使う開発では、クエリ結果がプロンプトにコピーされたり、スクリプトに取り込まれたり、最終的な確認なしにエージェントへ返ったりします。

mysh はそのギャップを埋めます。

- 本番および非 TTY のコンテキストで、設定した機微フィールドを自動でマスキングします
- MySQL の接続プロファイルと SSH トンネルを 1 コマンドでまとめて扱います
- パスワードを含めずに接続設定を共有し、各メンバーが自分の認証情報を入力できます
- DBeaver、Sequel Ace、MySQL Workbench から接続設定をインポートできます
- Homebrew、winget/MSI、単体バイナリ、`go install` でインストールできます

## AI セーフな出力マスキング

mysh は本番データや、AI エージェント／スクリプトに取り込まれる出力を検出すると、機微な値を CLI の外に出す前にマスキングします。

```bash
mysh run production -e "SELECT id, name, email, phone FROM users LIMIT 2" --format markdown
```

マスキング後の出力例:

| id | name | email | phone |
|----|------|-------|-------|
| 1 | A*** | a***@example.com | 0*** |
| 2 | B*** | b***@example.com | 0*** |

本番接続での `--raw` 出力は対話的な TTY 確認を求めるため、AI ツールや非対話のスクリプトがマスキングを黙って回避することはできません。

詳しい推奨設定とチーム運用は [AI コーディングエージェントと mysh を安全に使う](../ai-agent-safety.md)（英語）を参照してください。

## MCP サーバー（AI エージェントへのネイティブ統合）

mysh は [Model Context Protocol](https://modelcontextprotocol.io) サーバーを内蔵しています。AI コーディングエージェントは CLI を経由せず、データベースを第一級のツールとして直接クエリできます。同じマスキングルールが適用され、**MCP 経由では生出力を要求できません**。機微な値はエージェントに届く前にマスキングされます。

Claude Code への登録:

```bash
claude mcp add mysh -- mysh mcp
```

その他の MCP クライアント（Cursor、`claude_desktop_config.json` など）:

```json
{
  "mcpServers": {
    "mysh": { "command": "mysh", "args": ["mcp"] }
  }
}
```

詳細は [MCP サーバーガイド](../mcp-guide.md)（英語）を参照してください。

## 機能比較

| 機能 | `mysql` CLI | `mycli` | DBeaver | mysh |
|---|:---:|:---:|:---:|:---:|
| CLI 中心のワークフロー | ✅ | ✅ | ❌ | ✅ |
| 接続プロファイル管理 | ❌ | ⚠️ | ✅ | ✅ |
| SSH トンネル管理 | ❌ | ❌ | ✅ | ✅ |
| AI／非 TTY 出力の自動マスキング | ❌ | ❌ | ❌ | ✅ |
| 本番 `--raw` の安全確認 | ❌ | ❌ | ❌ | ✅ |
| AI エージェント向け MCP サーバー内蔵 | ❌ | ❌ | ❌ | ✅ |
| GUI DB クライアントからのインポート | ❌ | ❌ | — | ✅ |
| パスワードなしのチーム設定共有 | ❌ | ❌ | ⚠️ | ✅ |
| ローカル認証情報の暗号化保存 | ❌ | ⚠️ | ✅ | ✅ |
| Markdown/CSV/JSON/PDF エクスポート | ❌ | ❌ | ✅ | ✅ |
| MySQL 4.x old_password 対応 | ⚠️ | ⚠️ | ⚠️ | ✅ |

## おもな機能

- 暗号化したパスワード保存つきの対話的な接続セットアップ（AES-256-GCM + Argon2id）
- SSH トンネル管理（アドホックおよびバックグラウンド常駐トンネル）
- AI／非 TTY 実行向けの自動出力マスキング（本番の個人データを保護）
- MCP サーバー内蔵（`mysh mcp`）。AI エージェントがマスキング適用のままネイティブにクエリ可能
- 複数ターミナルでの複数接続を競合なく利用
- mycli を優先し、なければ標準 mysql クライアントにフォールバック
- MySQL 4.x old_password 認証に対応した Go ネイティブドライバ
- 出力フォーマット変換（plain、markdown、CSV、JSON、PDF）とファイル出力
- DBeaver、Sequel Ace、MySQL Workbench からの接続インポート

## インストール

### macOS / Linux

```bash
brew tap atani/tap
brew install mysh
```

### Windows

[最新リリース](https://github.com/atani/mysh/releases/latest) から `mysh-windows-x64.msi`（または `-arm64`）をダウンロードして実行するか、winget を使います。

```powershell
winget install atani.mysh
```

### Go

```bash
go install github.com/atani/mysh@latest
```

## クイックスタート

```bash
# 接続を対話的に追加
mysh add

# 接続テスト（接続が 1 つだけなら名前は省略可）
mysh ping

# 接続
mysh connect
```

## ドキュメント

- [はじめに（非エンジニア向け）](getting-started.md) — mysh をセットアップして Claude Code からクエリを実行する
- [接続設定のインポート](import-guide.md) — DBeaver、Sequel Ace、MySQL Workbench から移行する
- [Redash 連携ガイド](redash-guide.md) — Redash 経由でデータベースにクエリを投げる
- [接続設定の共有](sharing-connections.md) — チーム導入のために設定をエクスポート／インポートする

英語版のフルドキュメント（マスキングの詳細設定、SSH トンネル、出力フォーマット、設定リファレンスなど）は [ルートの README](../../README.md) を参照してください。

## Star と貢献

mysh が本番データを AI エージェントのコンテキストから守るのに役立ったら、リポジトリに ⭐ を付けてください。より安全な AI 連携のデータベース運用を、ほかのチームが見つけやすくなります。

- 💬 [Discussions](https://github.com/atani/mysh/discussions) で使い方の共有や質問ができます
- 🐣 [good first issue](https://github.com/atani/mysh/issues?q=is%3Aissue+is%3Aopen+label%3A%22good+first+issue%22) から貢献を始められます
- 🐛 不具合報告や機能要望は [Issues](https://github.com/atani/mysh/issues) へ

## ライセンス

[MIT License](../../LICENSE)
