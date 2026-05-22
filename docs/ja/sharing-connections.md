# 接続設定の共有

myshの接続設定をエクスポート・インポートして、チームメンバーのセットアップを簡単にする方法です。

## 概要

エンジニアが一度接続設定を作って、チームに配布します。受け取る側は自分の認証情報を入力するだけで、それ以外の設定（environment、SSH、マスキングルール）はそのまま引き継がれます。

```
エンジニア                          チームメンバー
──────────                          ──────────
mysh export prod > prod.yaml  →   mysh import --from yaml --file prod.yaml
                                   → パスワード入力
                                   → 完了
```

## エクスポート

### 特定の接続をエクスポート

```bash
mysh export prod > prod.yaml
```

### 全接続をエクスポート

```bash
mysh export > all-connections.yaml
```

### 含まれる内容

| 項目 | 含まれる | 備考 |
|------|----------|------|
| 接続名 | Yes | |
| Environment | Yes | production, staging, development |
| DBホスト | Yes | |
| DBポート | Yes | |
| DBユーザー | Yes | |
| DB名 | Yes | |
| DBパスワード | **No** | セキュリティのため常に除外 |
| SSH設定 | Yes | host, port, user, key path |
| マスクルール | Yes | columns と patterns |
| Driver | Yes | cli または native |
| Redash URL | Yes | Redash接続の場合 |
| Redash APIキー | **No** | セキュリティのため常に除外 |
| Redashデータソース ID | Yes | Redash接続の場合 |

### 出力例

```yaml
- name: production
  env: production
  ssh:
    host: bastion.example.com
    user: deploy
    key: ~/.ssh/id_ed25519
  db:
    host: 10.0.0.5
    port: 3306
    user: app
    database: myapp_production
  mask:
    columns:
      - email
      - phone
    patterns:
      - "*address*"
      - "*secret*"
```

## インポート

### YAMLファイルから

```bash
mysh import --from yaml --file prod.yaml
```

インポートフロー:

1. ファイル内の接続一覧を表示
2. インポートするものを選択
3. DBパスワード（またはRedash APIキー）を入力
4. 接続テスト
5. 設定を保存

### 全部まとめて一気にインポート

```bash
mysh import --from yaml --file prod.yaml --all
```

### DBユーザー名を各自のものに変えてインポート

DB直結の接続をチーム共有する場合、接続先は同じでもDBユーザー名はメンバーごとに異なることがあります。`--ask-user` を付けると、YAML内のユーザー名をデフォルトとして表示しながら、接続ごとに自分のDBユーザー名へ上書きできます。

```bash
mysh import --from yaml --file team-db.yaml --all --ask-user
```

複数接続で同じDBユーザーを使う場合は、`--db-user` で一括指定できます。

```bash
mysh import --from yaml --file team-db.yaml --all --db-user taro
```

同じパスワードを複数のDB接続で使う場合は、`--reuse-password` を付けるとパスワード入力が1回で済みます。

```bash
mysh import --from yaml --file team-db.yaml --all --db-user taro --reuse-password
```

`--reuse-password` は入力した1つのパスワードを全DB接続に適用するため、それらの接続が同一アカウントを共有している場合（通常は `--db-user` と併用）にのみ使ってください。共有パスワードで接続テストに失敗した接続もインポートされ、まとめて一覧表示されるので `mysh edit <name>` で修正できます。

`--ask-user` と `--db-user` は同時には使えません。Redash接続は従来通り、各メンバーが自分のRedash APIキーを入力します。

### 受け取り側に必要なもの

**直接DB接続の場合:**

- エンジニアから受け取ったYAMLファイル
- DBパスワード

**Redash接続の場合:**

- エンジニアから受け取ったYAMLファイル
- 自分のRedash APIキー（Redashのプロフィール設定から取得）

## ベストプラクティス

### 共有するエンジニア向け

1. **エクスポート前に environment とマスクルールを設定する** — 受け取り側はこれを引き継ぐ
2. 実ユーザーデータにアクセスする接続は **production environment** を指定
3. **マスクパターンを網羅的に設定する** ことで意図しないデータ露出を防ぐ:
   ```bash
   mysh edit prod
   # マスクを以下に設定: email,phone,*password*,*secret*,*token*,*address*,*name*
   ```
4. **YAMLファイルの共有経路はセキュアに** — パスワードは含まれないが、ホスト名やユーザー名は含まれる

### 受け取る側

1. `MYSH_MASTER_PASSWORD` をシェルプロファイルに設定しておくと、AIアシスタントから対話プロンプトなしでmyshを使える
2. インポート後 `mysh ping` で接続を必ず確認する
3. マスク設定はエンジニアの指示なしに変更しない

## Redash接続の共有

Redash接続も同じ流れで共有できます。

```bash
# エンジニアがエクスポート
mysh export analytics > analytics.yaml

# 受け取り側は自分のAPIキーを入力してインポート
mysh import --from yaml --file analytics.yaml
```

各メンバーが自分のRedash APIキーを使うので、アクセスはRedashの監査ログで個別に追跡できます。
