# はじめに（非エンジニア向け）

myshをセットアップして、Claude Codeなどのアシスタントから本番データベースに安全にアクセスする手順です。

## 必要なもの

- ターミナルアプリ（macOSのTerminal、WindowsのPowerShell）
- [Claude Code](https://docs.anthropic.com/en/docs/claude-code) などのAIコーディングアシスタント

## myshのインストール

### macOS / Linux

```bash
brew tap atani/tap
brew install mysh
```

### Windows

1. [最新リリース](https://github.com/atani/mysh/releases/latest)から `mysh-windows-amd64.exe` をダウンロード
2. `mysh.exe` にリネーム
3. PATHが通ったフォルダに配置（わからなければエンジニアに聞いてください）

### 確認

```bash
mysh help
```

## セットアップ方法を選ぶ

| 方法 | こんなチームに | 必要なもの |
|------|--------------|-----------|
| [Redash経由](#方法a-redash推奨) | Redashを使っている | RedashのAPIキー |
| [DB直接接続](#方法b-db直接接続) | Redashがない | エンジニアからもらうYAMLファイル |

## 方法A: Redash（推奨）

チームで[Redash](https://redash.io/)を使っているなら、これが最も簡単です。DB認証情報もSSHも不要で、RedashのAPIキーだけで始められます。

### 1. APIキーを取得

1. RedashにログインしてMysh
2. 右上のプロフィールアイコン → **Settings**
3. **API Key** をコピー

### 2. 接続を追加

```bash
mysh add --name prod --redash-url https://redash.yourcompany.com --redash-key YOUR_API_KEY --redash-datasource 1
```

`--redash-datasource` の番号はエンジニアに確認してください。わからなければ `1` がメインのDBであることが多いです。

### 3. マスターパスワードの設定

myshが初回にマスターパスワードの設定を求めます。これはAPIキーを暗号化して保存するためのものです。

AIアシスタント経由で使う場合、毎回入力しなくて済むように環境変数を設定しておきます：

**macOS / Linux** — `~/.zshrc` または `~/.bashrc` に追記：
```bash
export MYSH_MASTER_PASSWORD="your-master-password"
```

**Windows** — PowerShellで実行：
```powershell
[Environment]::SetEnvironmentVariable("MYSH_MASTER_PASSWORD", "your-master-password", "User")
```

### 4. 接続テスト

```bash
mysh ping prod
```

以下のように表示されればOKです：
```
Connection "prod" (Redash): OK (123ms)
```

### 5. クエリを実行

Claude Codeに自然言語で聞くだけです：

> 「先月のプラン別の新規登録者数を出して」

Claude Codeが自動でSQLを生成し、`mysh run prod -e "SELECT ..."` で実行します。メールアドレスや電話番号などの機密カラムは自動でマスキングされます。

## 方法B: DB直接接続

Redashがないチームでは、エンジニアが接続設定をエクスポートして共有します。

### 1. 設定ファイルをもらう

エンジニアに以下を実行してもらいます：

```bash
mysh export prod > prod.yaml
```

このファイルをSlackやメールで受け取ります。パスワードは含まれていません。

### 2. 設定をインポート

```bash
mysh import --from yaml --file prod.yaml
```

DBパスワードの入力を求められます。エンジニアに聞いてください。

### 3. マスターパスワードの設定

Redashの場合と同じです。[上記の手順3](#3-マスターパスワードの設定)を参照してください。

### 4. テストして使い始める

```bash
mysh ping prod
```

あとはClaude Codeに聞くだけです。

## Claude Codeでの使い方

myshのセットアップが済めば、Claude Codeにこう聞けます：

- 「今月の注文数トップ10の顧客を見せて」
- 「先週作成されたサポートチケットの数は？」
- 「プラン別のユーザー分布を出して」

Claude Codeが：
1. SQLクエリを自動生成
2. mysh経由で実行（マスキング自動適用）
3. 結果を分析して表示

### 安全機能

- **本番データの自動マスキング**: メールアドレス、電話番号などが自動的にマスクされます（例：`alice@example.com` → `a***@example.com`）
- **AIツールはマスキングを回避不可**: `--raw` を使おうとしても、本番環境ではターミナルでの対話確認が必要です
- **APIキーは暗号化保存**: AES-256-GCMで保護されます

## トラブルシューティング

### "mysh: command not found"

myshにPATHが通っていません。macOS/Linuxの場合、Homebrew後にターミナルを再起動してください。

### "wrong master password"

マスターパスワードが間違っています。忘れた場合は `~/.config/mysh/.master_check` を削除して接続を再登録してください。

### "redash API returned HTTP 403"

RedashのAPIキーが無効か期限切れです。Redashのプロフィール設定から新しいキーを発行してください。

### 接続テストが失敗する

- VPN接続が必要な場合があります
- パスワードが正しいか確認してください
- エンジニアにSSH設定を確認してもらってください
