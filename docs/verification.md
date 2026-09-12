# 検証記録

## 2026-09-13: Goバージョン設定後の再検証

`mise.toml` の `[tools]` にGo 1.26.8を指定しました。
Windows / mise 2026.8.6で、`MISE_GO_VERSION` が未設定の状態からGo 1.26.8が選択されることを確認しました。
`mise build`、`mise tidy`、`mise run fmt`、`mise vet`、`mise test`、`mise check` はすべて終了コード0でした。
テスト結果にはGoのテストキャッシュが使用されています。Goソースと依存ファイルに変更はありません。
race detector、GitHub Actions、実Azureへの接続は今回も検証していません。

## 2026-09-13: miseタスクの検証

Windows / Go 1.26.8 / mise 2026.8.6で、`mise run tidy`、`mise run fmt`、
`mise run check`（`go vet ./...`、`go test ./...`、`go build ./...`）が成功しました。
この時点では検証時に `MISE_GO_VERSION=1.26.8` を指定しており、環境変数なしでのタスク実行は未確認でした。
依存関係を取得し、生成された `go.sum` と更新された `go.mod` を追加しています。
race detector、GitHub Actions、実Azureへの接続は今回の検証では実行していません。

以下は初期作成時点の記録です。

作成日: 2026-09-12

## 結論

ソースコード・利用例・テストを実装していますが、**全体ビルド検証済みのリリースではありません**。
この作成環境では外部のGoモジュール取得に必要な名前解決／通信が利用できず、
Azure SDKをダウンロードできませんでした。実際のSDKを使った型検査・テスト結果の代わりに、
スタブSDKでの成功結果を提示することはしていません。

## 実行できたもの

環境: Linux amd64 / Go 1.23.2

| 検証 | 結果 |
|---|---|
| 全18個のGoファイルを `go/parser` で構文解析 | 成功。型検査ではない |
| `gofmt -l .` | 未整形ファイルなし |
| `go test -race -cover -count=1 ./internal/...` | 2パッケージ、8個のトップレベルテストが成功 |
| `go vet ./internal/...` | 成功 |

実行ログの要約:

```text
ok github.com/nuitsjp/azurego/internal/check  coverage: 100.0% of statements
ok github.com/nuitsjp/azurego/internal/paging coverage: 100.0% of statements
Parsed 18 Go files successfully (syntax only; not type checking).
```

上のカバレッジは**内部ユーティリティ2パッケージだけ**の値です。
ライブラリ全体のカバレッジを表しません。

確認した内部動作は、Subscription IDの検証、必須値の検証、未選択スコープのエラー、
全ページの集約、空リスト、途中のエラーで部分結果を破棄する動作、キャンセルです。

## 実行できなかったもの

`go mod tidy` を実行しましたが、`proxy.golang.org` の名前解決／通信エラーで失敗しました。
そのため `go.sum` は生成できていません。`go build ./...` も依存未解決で失敗しました。
これは「ソースのコンパイルエラーがない」という意味ではありません。

未検証の範囲:

- SDKに依存するパッケージとサンプルのコンパイル・全体の `go vet`。
- `azurego_test.go` の15個のテスト。偽の認証・HTTPトランスポートから実際のSDKを動かす設計で、実Azureの権限や料金は不要です。
- GitHub ActionsのWindows／Ubuntuワークフローの実行。
- 実Azureへの認証、読み取り、作成／更新／削除。

SDK接続テストには、RESTパス、認証ヘッダー、ページ継続、SKU／モデルの送信内容、
PATCHでモデルを上書きしないこと、エラーの保持、長時間処理の完了待ち、キャンセルを含めています。

## 手元での検証手順

リポジトリのルートで実行します。

```console
go mod tidy
go mod verify
go test ./...
go vet ./...
go build ./...
```

race detectorを利用できる環境では、追加で実行します。

```console
go test -race -count=1 ./...
```

続いて **読み取りだけ** の接続確認を行います。

```console
az login
go run ./examples/discover
go run ./examples/discover -subscription YOUR_SUBSCRIPTION_ID
go run ./examples/models -subscription YOUR_SUBSCRIPTION_ID -resource-group YOUR_RG -account YOUR_FOUNDRY_ACCOUNT
```

作成・更新・削除は対象リソース、料金、SKU、クォータを確認したテスト用環境で実施してください。
デプロイ更新を試す前には既存の設定を確認し、容量変更だけなら `Update` を使ってください。
