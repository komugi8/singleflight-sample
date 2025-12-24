# Singleflight Cache Stampede Demo

このプロジェクトは、Go言語の`golang.org/x/sync/singleflight`パッケージを使用したCache Stampede対策のデモンストレーションです。

## 概要

Webサービスでキャッシュを利用する際に発生する**Cache Stampede**（キャッシュ・スタンピード）問題と、`singleflight`パッケージによる解決策を実際に体験できるサンプルプログラムです。

### Cache Stampedeとは？

キャッシュのTTL（有効期限）が切れた瞬間に大量のリクエストが同時に来ると、全てのリクエストがキャッシュミスとなり、データベースに直接アクセスしてしまう現象です。これにより：

- データベース負荷が急激に増加
- レスポンス時間の大幅な悪化
- 最悪の場合、サービス全体のダウン

### Singleflightによる解決

`singleflight`は「同じキーに対する処理が実行中の場合、その間に到着した他のリクエストを待機させ、最初の処理結果を共有する」仕組みです。

## プロジェクト構成

```
singleflight-sample/
├── main.go                 # メインアプリケーション
├── go.mod                  # Go依存関係
├── go.sum                  # 依存関係ロックファイル
├── client/
│   └── load_client.go      # 負荷テストクライアント
└── README.md               # このファイル
```

## 機能

### メインサーバー (`main.go`)

- **シンプルなキャッシュ**: インメモリマップでキャッシュ実装
- **モックDB**: `time.Sleep()`で重い処理をシミュレート
- **Singleflight**: 核心機能のみに集中
- **最小限のログ**: HIT/MISS/SHAREDの状態を表示

### 負荷テストクライアント (`client/load_client.go`)

2種類のsingleflightテスト：

- `stampede`: Cache Stampede テスト（100並行リクエスト）  
- `normal`: 通常負荷テスト（10並行リクエスト）

## 実行方法

### 1. 依存関係のインストール
```bash
go mod tidy
```

### 2. サーバー起動
```bash
# デフォルト設定（ポート80、DB遅延3秒）
go run main.go

# 環境変数でカスタマイズ
PORT=9000 DB_DELAY=5000 go run main.go
```

### 3. 負荷テスト実行
```bash
# Cache Stampedeテスト（推奨）
go run client/load_client.go stampede

# 通常負荷テスト
go run client/load_client.go normal

```

### 4. 手動テスト

サーバー起動後、以下のエンドポイントにアクセス：

```bash
# ランキングデータ取得
curl http://localhost:80/ranking

# キャッシュステータス確認
curl -I http://localhost:80/ranking
```

## テストシナリオ

### 1. Cache Stampede効果の確認

```bash
# サーバー起動
go run main.go

# 別ターミナルで
go run client/load_client.go stampede
```

**期待される結果：**
- 1つのリクエストのみが`LEADER`としてDB処理を実行
- 他の99リクエストは`SHARED`として結果を共有
- 総処理時間が約3秒（DB遅延時間）で完了

### 2. キャッシュヒット確認

```bash
# 最初のリクエスト（キャッシュミス）
curl -I http://localhost:80/ranking
# → X-Cache: MISS

# すぐに2回目のリクエスト（キャッシュヒット）
curl -I http://localhost:80/ranking
# → X-Cache: HIT
```

### 3. TTL切れの確認

```bash
# 1回目のリクエスト
curl -I http://localhost:80/ranking

# 11秒待機（TTLは10秒）
sleep 11

# 2回目のリクエスト（再びキャッシュミス）
curl -I http://localhost:80/ranking
# → X-Cache: MISS
```

## ログの読み方

```
Request started
Cache MISS - using singleflight
DB query started (LEADER)
DB query completed
Response sent (MISS, 3.002s)
Response sent (SHARED, 3.002s)
```

- `LEADER`: 実際にDB処理を実行したリクエスト
- `SHARED`: 他のリクエストの結果を共有したリクエスト  
- `HIT`: キャッシュから結果を取得

## カスタマイズ

### 環境変数

- `PORT`: サーバーポート（デフォルト: 80）
- `DB_DELAY`: DB処理遅延時間（ミリ秒、デフォルト: 3000）

### 設定例

```bash
# 高速テスト（遅延500ms）
DB_DELAY=500 go run main.go

# 重いDB処理シミュレート（遅延10秒）
DB_DELAY=10000 go run main.go
```

## ベンチマーク比較

### Singleflightなしの場合（仮想）

100並行リクエスト時：
- 100個のDB処理が並行実行
- 総処理時間: 約3秒（並行処理）
- DB負荷: 100倍

### Singleflightありの場合

100並行リクエスト時：
- 1個のDB処理のみ実行
- 総処理時間: 約3秒
- DB負荷: 1倍

## トラブルシューティング

### ポート使用中エラー

```bash
# 使用中のプロセスを確認
lsof -i :80

# 別ポートで起動
PORT=9000 go run main.go
```

### 依存関係エラー

```bash
# 依存関係をクリーンアップして再インストール
go mod tidy
go clean -modcache
go mod download
```

## 参考資料

- [docs/article.md](docs/article.md) - 詳細な技術解説  
- [golang.org/x/sync/singleflight](https://pkg.go.dev/golang.org/x/sync/singleflight) - 公式ドキュメント

このサンプルにより、キャッシュ全般のCache Stampede問題とsingleflightによる解決策を実際のコードで体験・学習できます。

## ライセンス

MIT License