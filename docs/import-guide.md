# 接続設定のインポート

DBeaver や Sequel Ace で管理している MySQL 接続設定を mysh にインポートできます。

## 対応ツール

| ツール | 設定ファイルの場所 |
|--------|-------------------|
| DBeaver | `~/Library/DBeaverData/workspace6/General/.dbeaver/data-sources.json` |
| Sequel Ace | `~/Library/Containers/com.sequel-ace.sequel-ace/Data/Library/Application Support/Sequel Ace/Data/Favorites.plist` |

## 基本的な使い方

```bash
# DBeaver からインポート
mysh import --from dbeaver

# Sequel Ace からインポート
mysh import --from sequel-ace
```

実行すると、検出された接続の一覧が表示されます。

```
DBeaver: 5 connection(s) found

  #  NAME              HOST              PORT  USER   DATABASE  SSH
  1  lolipop-appdb-3   10.51.80.122      3306  admin  lolipop   manage001.phy.lolipop.jp
  2  server-db         serverdb-rep...   3306  admin  hosting   manage001.phy.lolipop.jp
  3  local-db          127.0.0.1         33306 root   devdb     -

Select connections (comma-separated numbers, or 'all') [all]:
```

番号をカンマ区切りで指定するか、`all` で全件インポートします。

## インポートの流れ

各接続について最小限の項目だけを対話的に設定します:

1. **接続名** — 既存の名前と重複する場合は新しい名前を入力
2. **SSH ユーザー** — 元の設定に SSH ユーザーが含まれていない場合のみ
3. **パスワード** — セキュリティ上、パスワードは再入力が必要（Enter でスキップ可）

インポート後、環境やマスク設定が必要な接続については案内が表示されます。

## パスワードについて

DBeaver、Sequel Ace ともにパスワードは暗号化/Keychain で保護されており、自動インポートできません。
各接続のインポート時にパスワードを入力するか、Enter でスキップして後から `mysh edit` で設定できます。

## インポート後の設定

インポートされた接続はすべて `development` 環境として登録されます。
本番・ステージング環境の接続には、出力マスク（個人情報の秘匿化）を設定することを推奨します。

```bash
# 接続の環境・マスク・ドライバを設定
mysh edit <connection-name>
```

`mysh edit` で設定できる項目:

| 項目 | 説明 |
|------|------|
| 環境 | production / staging / development |
| マスク対象カラム | email, phone など（ワイルドカード対応） |
| ドライバ | cli（デフォルト）/ native（MySQL 4.x 対応） |

production 環境ではクエリ結果のマスクが常に有効になります。
staging 環境ではパイプ出力時（AI ツール等）にマスクが有効になります。

## 全件一括インポート

`--all` フラグで選択画面をスキップし、検出された全接続をインポートします。

```bash
mysh import --from dbeaver --all
```

## インポートされる項目

| 項目 | DBeaver | Sequel Ace |
|------|---------|------------|
| ホスト | ✅ | ✅ |
| ポート | ✅ | ✅ |
| ユーザー | ✅ | ✅ |
| データベース名 | ✅ | ✅ |
| パスワード | ❌（再入力） | ❌（再入力） |
| SSH ホスト | ✅ | ✅ |
| SSH ポート | ✅ | ✅ |
| SSH ユーザー | △（未設定の場合あり） | ✅ |
| SSH 鍵パス | ✅ | ✅ |

## DBeaver からの移行手順

1. DBeaver が起動中でも問題ありません（設定ファイルの読み取りのみ）
2. `mysh import --from dbeaver` を実行
3. インポートする接続を選択
4. 各接続のパスワードを入力（Enter でスキップ可）
5. `mysh list` でインポート結果を確認
6. `mysh ping <name>` で接続テスト
7. 本番環境の接続には `mysh edit <name>` で環境とマスク設定を追加

## Sequel Ace からの移行手順

1. Sequel Ace の Favorites に接続が保存されていることを確認
2. `mysh import --from sequel-ace` を実行
3. インポートする接続を選択
4. 各接続のパスワードを入力（Enter でスキップ可）
5. `mysh list` でインポート結果を確認
6. `mysh ping <name>` で接続テスト
7. 本番環境の接続には `mysh edit <name>` で環境とマスク設定を追加

> **Note**: Sequel Ace のインポートには macOS の `plutil` コマンドを使用します（macOS 標準搭載）。

## トラブルシューティング

### 「No MySQL connections found」と表示される

- DBeaver: `~/Library/DBeaverData/workspace6/` にワークスペースがあるか確認してください。DBeaver のバージョンによってパスが異なる場合があります。
- Sequel Ace: Favorites.plist が存在するか確認してください。

### SSH ユーザーの入力を求められる

DBeaver は SSH ユーザーを `data-sources.json` に保存しないことがあります（OS のユーザー名を使用するため）。
mysh では SSH ユーザーを明示的に設定する必要があるため、インポート時に入力してください。

### インポート後にパスワードを設定したい

```bash
mysh edit <connection-name>
```

で後からパスワードを含む全項目を編集できます。
