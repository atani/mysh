# Redash連携ガイド

Redash経由で本番データベースにクエリを投げる方法です。DB認証情報やSSHトンネルは不要で、出力マスキングも自動で適用されます。

## 概要

```
あなた / Claude Code → mysh → Redash API → データベース
                        ↓
                    マスキング適用
                        ↓
                    安全な結果
```

myshがSQLをRedashに送信し、結果を受け取ってマスキングルールを適用し、サニタイズされた出力を返します。個人情報がAIアシスタントに届くことはありません。

## セットアップ

### 1. Redash APIキーを取得する

1. Redashにログイン
2. プロフィールアイコン → **Settings**
3. **API Key** をコピー

### 2. データソースIDを確認する

エンジニアに聞くか、Redashで以下を確認します。

1. **Settings** → **Data Sources**
2. 使いたいデータソースをクリック
3. URLの末尾がID: `https://redash.example.com/data_sources/3` → IDは `3`

### 3. 接続を追加する

```bash
mysh add --name prod \
  --redash-url https://redash.example.com \
  --redash-key YOUR_API_KEY \
  --redash-datasource 3
```

対話で以下が聞かれます。

- **Environment**: 自動マスキングを有効にするため `production` を選択
- **Mask columns**: そのままEnterでデフォルト（`email,phone,*password*,*secret*,*token*,*address*`）

### 4. 動作確認

```bash
mysh ping prod
# Connection "prod" (Redash): OK (85ms)
```

## 使い方

### クエリを実行する

```bash
# インラインSQL
mysh run prod -e "SELECT * FROM users LIMIT 5"

# SQLファイルから実行
mysh run prod query.sql
```

### 出力フォーマット

```bash
# CSV
mysh run prod -e "SELECT * FROM users" --format csv

# JSON
mysh run prod -e "SELECT * FROM users" --format json

# Markdownテーブル
mysh run prod -e "SELECT * FROM users" --format markdown

# ファイルへ保存
mysh run prod -e "SELECT * FROM users" --format csv -o users.csv
```

### Claude Codeと使う

やりたいことを自然言語で書くだけです。

> 「アクティブなサブスクリプションをプラン別に集計して」

Claude CodeがSQLを自動生成し、mysh経由で実行してくれます。

## マスキング

直接接続の場合と同じ挙動です。マスクルールに一致したカラムは自動で伏せ字になります。

| 種別 | 元の値 | マスク後 |
|------|--------|----------|
| メール | alice@example.com | a\*\*\*@example.com |
| 電話番号 | 090-1234-5678 | 0\*\*\* |
| 氏名 | Alice | A\*\*\* |

production接続では、端末から実行しようとスクリプトから実行しようとAIアシスタントから実行しようと、常にマスキングが適用されます。

## ノンインタラクティブセットアップ

スクリプトや自動化（新メンバーのプロビジョニング等）向けに、対話を飛ばせます。

```bash
export MYSH_MASTER_PASSWORD="the-master-password"

mysh add --name prod \
  --redash-url https://redash.example.com \
  --redash-key "$REDASH_API_KEY" \
  --redash-datasource 3 \
  --env production \
  --mask "email,phone,*password*,*secret*,*token*,*address*"
```

## 複数データソース

用途別に複数のRedash接続を登録できます。

```bash
mysh add --name analytics \
  --redash-url https://redash.example.com \
  --redash-key YOUR_KEY \
  --redash-datasource 5 \
  --env production \
  --mask "email,phone"

mysh add --name logs \
  --redash-url https://redash.example.com \
  --redash-key YOUR_KEY \
  --redash-datasource 8 \
  --env staging
```

```bash
mysh run analytics -e "SELECT count(*) FROM events WHERE date = CURDATE()"
mysh run logs -e "SELECT * FROM access_logs LIMIT 10"
```

## 接続設定の共有

エンジニアはRedash接続をチームに配布できます。

```bash
# エクスポート（APIキーは除外される）
mysh export prod > prod-redash.yaml

# 受け取り側は自分のAPIキーを入れてインポート
mysh import --from yaml --file prod-redash.yaml
```

## トラブルシューティング

### `redash API returned HTTP 403`

- APIキーが無効・期限切れ、または該当データソースへの権限がない
- Redashのプロフィール設定から新しいAPIキーを発行する

### `redash API returned HTTP 400`

- SQLのシンタックスエラー
- データソースIDが間違っている

### `redash API request failed: connection refused`

- Redash URLを確認
- 社内VPN接続が必要な場合もある

### `redash query failed: ...`

- Redashはクエリを実行したが失敗している（テーブルが存在しない、権限不足など）
- エラーメッセージで詳細を確認

### クエリに時間がかかる

- Redash側にクエリタイムアウト（通常5分）がある
- myshはRedashのジョブ完了を500msごとにポーリングしながら待機
- 大きなクエリには `LIMIT` を付けることを検討

### マスクしたいカラムがマスクされない

- `mysh edit prod` でマスク設定を確認
- カラム名は大文字小文字を区別せず比較される
- パターンはワイルドカードをサポート: `*address*` は `home_address`, `email_address` などにマッチ
