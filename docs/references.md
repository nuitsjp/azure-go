# 実装時に確認した公式資料

確認日: 2026-09-12。Web資料と公開SDKシグニチャを確認した記録です。
ここでの確認は、ローカル環境で依存を取得してコンパイルできたことを意味しません。

## 認証

- [Goローカル開発でのCLI認証](https://learn.microsoft.com/en-us/azure/developer/go/sdk/authentication/local-development-dev-accounts)
- [採用したAzureCLICredentialのソース（azidentity v1.13.0）](https://github.com/Azure/azure-sdk-for-go/blob/sdk/azidentity/v1.13.0/sdk/azidentity/azure_cli_credential.go)

SubscriptionをCLI認証オプションへ渡し、CLIの設定を変更せず対象アカウントを選択します。
SDKの既存認証経路の利用と、組織内で自作ツールを利用してよいかという承認判断は別問題です。

## 命名と管理スコープ

- [az cognitiveservices account deployment](https://learn.microsoft.com/en-us/cli/azure/cognitiveservices/account/deployment?view=azure-cli-latest)
- [az cognitiveservices model](https://learn.microsoft.com/en-us/cli/azure/cognitiveservices/model?view=azure-cli-latest)
- [FoundryリソースとプロジェクトのARMリソース種別](https://learn.microsoft.com/en-us/azure/ai-foundry/how-to/create-resource-template)

Foundryリソースとプロジェクトの作成そのものは、このライブラリの初期スコープに含めていません。

## 固定したGo SDK

| モジュール | バージョン | 用途 |
|---|---|---|
| `sdk/azcore` | `v1.20.0` | ARM設定・共通認証型・レスポンスエラー |
| `sdk/azidentity` | `v1.13.0` | AzureCLICredential |
| `sdk/resourcemanager/resources/armsubscriptions` | `v1.3.0` | Subscription・リージョン参照 |
| `sdk/resourcemanager/resources/armresources` | `v1.2.0` | Resource Group参照 |
| `sdk/resourcemanager/cognitiveservices/armcognitiveservices/v3` | `v3.0.0` | Foundry／OpenAIアカウントとデプロイ |

今回の操作に対応する公開版を固定しています。各モジュールの最新版であるという主張ではありません。
間接依存は初回の `go mod tidy` で解決し、生成した `go.mod`／`go.sum` を管理してください。

- [armsubscriptions v1.3.0 API](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armsubscriptions@v1.3.0)
- [armresources v1.2.0 API](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/resources/armresources@v1.2.0)
- [armcognitiveservices/v3 v3.0.0 API](https://pkg.go.dev/github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/cognitiveservices/armcognitiveservices/v3@v3.0.0)

特に `DeploymentsClient.BeginCreateOrUpdate`、`BeginUpdate`、`BeginDelete`、
`NewListPager`、`NewListSKUsPager`、`AccountsClient.NewListModelsPager` を確認しています。
Cognitive Services v3.0.0の対象APIバージョンは `2025-09-01` です。

`Update` のSDK引数は `PatchResourceTagsAndSKU` です。モデルバージョンのPATCHではありません。
AzureGoはSKU／容量変更に限定し、他のプロパティを誤って更新する抽象化を避けています。
