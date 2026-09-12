# AzureGo

Azure CLIに近い命名で、公式Azure SDK for Goを利用する小さな管理用ライブラリです。
認証だけ `AzureCLICredential` に委譲し、リソース操作はGoからAzure Resource Manager（ARM）へ直接送信します。
PowerShellの呼び出し、操作コマンドのラップ、独自のOAuthアプリ登録は行いません。

> **初期実装の検証状態**: Goの構文検査とSDKに依存しない内部テストは実行済みです。
> 作成環境のネットワーク制約によりAzure SDKの依存パッケージを取得できず、
> **ライブラリ全体のビルド・SDKを含むテスト・実Azureでの検証は未実施**です。
> 最初に下記の `go mod tidy` とテストを実行してください。詳細は [検証記録](docs/verification.md) に記載しています。

## 対象

Subscription、Resource Group、既存のFoundry／Azure OpenAIリソースを参照し、
アカウント直下のモデルデプロイを管理します。
対象リソースは `Microsoft.CognitiveServices/accounts` とその `deployments` です。

| 対象 | AzureGo API | 対応するAzure CLI |
|---|---|---|
| Subscription | `Account.List / Show` | `az account list / show` |
| リージョン | `Account.ListLocations` | `az account list-locations` |
| Resource Group | `Group.List / Show` | `az group list / show` |
| Foundry／OpenAIアカウント | `CognitiveServices.Account.List / Show` | `az cognitiveservices account list / show` |
| アカウントのモデル候補 | `CognitiveServices.Account.ListModels` | `az cognitiveservices account list-models` |
| リージョンのモデル候補 | `CognitiveServices.Model.List` | `az cognitiveservices model list` |
| デプロイ | `CognitiveServices.Account.Deployment.List / Show / Create / Delete` | `az cognitiveservices account deployment ...` |
| SKU・容量の変更 | `CognitiveServices.Account.Deployment.Update` | AzureGo独自の補助API。SDKのPATCHを使用 |
| デプロイのSKU候補 | `CognitiveServices.Account.Deployment.ListSKUs` | AzureGo独自の補助API |
| リージョンのクォータ | `CognitiveServices.Usage.List` | `az cognitiveservices usage list` |

**対象外**: Subscription／Resource Group／Foundryアカウントの作成・削除、プロジェクト管理、
Hub／Azure Machine Learningワークスペース配下のデプロイ、モデルの重みの登録・削除、
ファインチューニング、推論、APIキーの取得・ローテーション、Marketplace規約への同意、
課金額や実トークン使用量の収集、Foundryポータルの全モデルカタログ検索。

モデルそのものを新規作成するのではなく、提供済みモデルを**デプロイ**します。
`AI Foundryのモデル管理` 全般を網羅するものではありません。

## 動作前提

- Go 1.23以降。実運用・開発には現在サポートされているGoを使用してください。
- 既定認証では、Goプログラムから起動できるAzure CLIと、事前の `az login` が必要です。
- 対象の読み取り・デプロイ変更に必要なAzure権限が必要です。新規アプリ登録を省けても、RBAC・組織ポリシーを回避するものではありません。

ライブラリはサインイン画面を開きません。認証失敗時はエラーを返すので、利用者がCLIで再ログインしてください。
Windowsで実行する場合はWindows側のAzure CLIを、WSLで実行する場合はWSL側のAzure CLIを準備します。
標準構成はAzure Public Cloudです。他のクラウドは個別に構成・確認してください。

## 最初の起動

ZIPを展開した `azurego` ディレクトリで実行します。以下はWindowsのターミナルでもUbuntuでも同じです。

```console
go mod tidy
go mod verify
go test -race ./...
go vet ./...
go build ./...
az login
go run ./examples/discover
```

`-race` には対応プラットフォームとCコンパイラが必要です。利用できない環境では
まず `go test ./...` を実行し、CIの対応環境で `-race` を実施してください。

作成環境で依存取得ができなかったため、初期ZIPには `go.sum` はありません。
`go mod tidy` が生成した `go.sum` と更新後の `go.mod` を初回コミットに含めてください。
パッケージを `latest` で取得するのではなく、`go.mod` に固定した直接依存を基準に解決します。

テナントを指定してSubscriptionを確認する例:

```console
az login --tenant YOUR_TENANT_ID
go run ./examples/discover -tenant YOUR_TENANT_ID
```

選んだSubscriptionのResource GroupとCognitive Servicesアカウントを参照:

```console
go run ./examples/discover -subscription YOUR_SUBSCRIPTION_ID
```

`YOUR_SUBSCRIPTION_ID` には表示名ではなくUUIDを指定します。先頭のSubscriptionを勝手に選びません。

## ライブラリの利用

以下は呼び出し元の関数内での例です。

```go
client, err := azurego.New(&azurego.Options{
    SubscriptionID: subscriptionID,
    // TenantID: tenantID, // 必要な場合だけ指定
})
if err != nil {
    return err
}

ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
defer cancel()

groups, err := client.Group.List(ctx)
if err != nil {
    return err
}

models, err := client.CognitiveServices.Account.ListModels(ctx, "rg-ai", "my-foundry")
if err != nil {
    return err
}

deployments, err := client.CognitiveServices.Account.Deployment.List(ctx, "rg-ai", "my-foundry")
if err != nil {
    return err
}

// groups / models / deployments を利用する
```

`New(nil)` も利用でき、Subscriptionの探索だけを行えます。
`SubscriptionID` なしで `Group` や `CognitiveServices` を操作すると
`azurego.ErrSubscriptionRequired` を返します。ポインタがnilの名前空間は公開しません。

```go
client, err := azurego.New(nil)
if err != nil {
    return err
}
subscriptions, err := client.Account.List(ctx)
```

**複数Subscription**: 選択したIDごとに `New(&Options{SubscriptionID: id})` でクライアントを作ります。
SDKの操作対象とCLI認証に同じSubscription IDを渡し、`az account set` は実行しません。
`New` はネットワーク通信せず、初回のAPI呼び出し時に認証します。
作成したクライアントは繰り返し利用してください。

**複数テナント**: `Account.List` は認証されたテナントのARM一覧です。
`az account list` のローカルな複数テナントのキャッシュを集約した結果とは異なります。
必要なテナントごとに明示して探索してください。

## モデルの確認とデプロイ

モデル名・バージョン・フォーマット・SKUを現在の一覧から選びます。
利用できる具体的なモデル名をライブラリに固定していません。

```console
go run ./examples/models -subscription YOUR_SUBSCRIPTION_ID -resource-group rg-ai -account my-foundry -location eastus
```

リージョンは**Resource GroupではなくFoundry／OpenAIアカウント自身の配置先**を確認してください。
モデルが一覧に存在しても、SKU、残りクォータ、サービス側容量、追加承認などによりデプロイできない場合があります。
`Usage.List` はクォータ用であり、課金やトークン消費量を表すAPIではありません。
`SKUCapacity` の単位はモデル／SKUごとに異なり、一律にTPMへ換算しません。

一覧と詳細の取得:

```console
go run ./examples/deployments -subscription YOUR_SUBSCRIPTION_ID -resource-group rg-ai -account my-foundry
go run ./examples/deployments -action show -subscription YOUR_SUBSCRIPTION_ID -resource-group rg-ai -account my-foundry -deployment my-model
```

作成例（`MODEL_*`／`SKU_NAME`／`CAPACITY` は確認した値に置換）:

```console
go run ./examples/deployments -action create -subscription YOUR_SUBSCRIPTION_ID -resource-group rg-ai -account my-foundry -deployment my-model -model-format MODEL_FORMAT -model-name MODEL_NAME -model-version MODEL_VERSION -sku-name SKU_NAME -sku-capacity CAPACITY -apply
```

**`Create` は作成または更新（PUT）です。同名デプロイが存在すれば変更されます。**
部分更新ではないため、意図するモデル定義・SKU・設定を明示してください。
`CreateOptions` にない高度な既存プロパティを取得して引き継ぐ処理はありません。
モデルバージョンの変更にも `Create` を使用しますが、既存設定への影響とサービス側の変更可否を確認してください。

容量だけの変更は `Update`（PATCH）を使用します。モデル／コンテンツフィルターのプロパティを再送しません。

```console
go run ./examples/deployments -action update -subscription YOUR_SUBSCRIPTION_ID -resource-group rg-ai -account my-foundry -deployment my-model -sku-name SKU_NAME -sku-capacity CAPACITY -apply
```

削除:

```console
go run ./examples/deployments -action delete -subscription YOUR_SUBSCRIPTION_ID -resource-group rg-ai -account my-foundry -deployment my-model -apply
```

例の実行プログラムは変更系操作に `-apply` を要求します。
ライブラリ自体はUIを持たず、確認画面も出さないため、呼び出し元が利用者の意図を確認してください。
作成・変更・推論利用は課金を伴う場合があります。指定SKU、容量、対象を確認してください。

## 動作の約束

- `List` は全ページを集めてスライスを返します。成功して0件なら空スライス、途中失敗なら `nil, error` です。部分結果を完全な一覧として返しません。
- `Create`／`Update`／`Delete` は完了まで待ちます。処理受付時点で成功を返しません。
- `context` のキャンセル・期限切れは待機を停止しますが、**Azure側の処理を取り消しません**。再実行前に `Show` で状態を確認してください。
- 元のSDKエラーを `%w` で保持します。`errors.As` で `*azcore.ResponseError`、`errors.Is` でcontextエラーを判定できます。
- 独自リトライ、CLIへの操作フォールバック、自動ログイン、暗黙のSubscription切替は実装しません。SDK標準のリトライ・HTTP接続管理を利用します。
- 自動のResource Provider登録は無効です。未登録の場合は管理者の運用手順で対応してください。

```go
var responseError *azcore.ResponseError
if errors.As(err, &responseError) {
    fmt.Printf("status=%d code=%s\n", responseError.StatusCode, responseError.ErrorCode)
}
```

公開するデータ型はSDKモデルの型エイリアスです。SDKのResponse、Pager、Pollerは通常の戻り値に出しませんが、
モデルのポインタ型フィールド・JSON構造・SDKバージョンへの依存は残します。独自DTOを増やさないための意図的な設計です。
SDKメジャーバージョンの変更時は公開モデルへの影響も確認してください。

`Options.Credential` と `Options.ARMOptions` により、テスト用の認証情報・HTTPトランスポートを注入できます。
呼び出し元が別の認証を必要とする場合にも利用できますが、その権限やテナント選択は呼び出し元の責任です。
`Credential` と `TenantID` の同時指定は受け付けません。

## リポジトリ

```text
azurego/
├─ azurego.go / options.go / doc.go
├─ account/                    # Subscription・リージョン参照
├─ group/                      # Resource Group参照
├─ cognitiveservices/          # CLIの名前空間を公開する薄いSDKラッパー
│  ├─ client.go / account.go
│  ├─ model.go / usage.go
│  └─ deployment.go
├─ internal/check/             # 入力検証
├─ internal/paging/            # 共通の全件取得
├─ examples/                   # discover / models / deployments
├─ azurego_test.go              # 偽HTTPを使うSDK接続テスト
├─ docs/                       # 設計・出典・検証記録
└─ .github/workflows/ci.yml     # Windows / Ubuntuのビルド・テスト
```

小さいため、Cognitive Services配下は同一Goパッケージ内でファイルを分けています。
公開APIの階層はCLIに合わせますが、メソッドごとにパッケージやService／Repository層は追加しません。

Module pathは公開先の仮設定として `github.com/nuitsjp/azurego` にしています。
このZIPの作成ではGitHub上のリポジトリ作成・公開は行っていません。
別の公開先を使用するときは `go.mod` のmodule行とGoソースの同プレフィックスのimportを一括変更してください。
公開前に別のローカルプロジェクトから参照するときは、呼び出し側で次のように指定できます。

```console
go mod edit -require=github.com/nuitsjp/azurego@v0.0.0
go mod edit -replace=github.com/nuitsjp/azurego=../azurego
go mod tidy
```

本プロジェクトは[MITライセンス](LICENSE)の下で公開されています。

詳細: [アーキテクチャ](docs/architecture.md) / [公式資料](docs/references.md) / [検証記録](docs/verification.md)
