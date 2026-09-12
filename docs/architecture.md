# アーキテクチャ

AzureGoはCLIの操作体系を参考にしたFacadeです。CLIの振る舞い全体を再実装するものではありません。

```text
呼び出し元 → AzureGo → Azure SDK for Go → ARM REST API
                            │ 認証が必要なとき
                            └→ AzureCLICredential → az account get-access-token
```

認証はAPI呼び出しの前提として並行する関心事であり、REST APIアクセスがCLI経由になるわけではありません。
SDKのBearer Tokenポリシーにトークン利用を任せ、独自の永続トークンキャッシュは持ちません。
`AzureCLICredential.GetToken` 自体はCLIを起動するため、「CLI起動はプロセス全体で一回だけ」とは保証しません。

## クラス図（Goのstruct）

```mermaid
classDiagram
    class Client {
        +Account
        +Group
        +CognitiveServices
    }
    class AccountClient {
        +List(ctx)
        +Show(ctx, subscriptionID)
        +ListLocations(ctx, subscriptionID)
    }
    class GroupClient {
        +List(ctx)
        +Show(ctx, name)
    }
    class CognitiveServicesClient {
        +Account
        +Model
        +Usage
    }
    class CognitiveAccountClient {
        +Deployment
        +List(ctx, resourceGroup)
        +Show(ctx, resourceGroup, name)
        +ListModels(ctx, resourceGroup, name)
    }
    class DeploymentClient {
        +List(ctx, resourceGroup, account)
        +Show(ctx, resourceGroup, account, name)
        +Create(ctx, resourceGroup, account, name, options)
        +Update(ctx, resourceGroup, account, name, options)
        +Delete(ctx, resourceGroup, account, name)
    }
    Client *-- AccountClient
    Client *-- GroupClient
    Client *-- CognitiveServicesClient
    CognitiveServicesClient *-- CognitiveAccountClient
    CognitiveAccountClient *-- DeploymentClient
    AccountClient --> AzureSDK
    GroupClient --> AzureSDK
    CognitiveAccountClient --> AzureSDK
    DeploymentClient --> AzureSDK
    AzureSDK --> AzureCLICredential : 必要時に認証
```

## 一覧取得

```mermaid
sequenceDiagram
    participant App
    participant Go as AzureGo
    participant SDK as Azure SDK
    participant CLI as Azure CLI
    participant ARM
    App->>Go: Account.List(ctx)
    Go->>SDK: NewListPager / NextPage
    opt SDKが新しいトークンを必要とする
        SDK->>CLI: AzureCLICredential.GetToken
        CLI-->>SDK: access token
    end
    loop nextLinkがある間
        SDK->>ARM: GET /subscriptions または nextLink
        ARM-->>SDK: page
        SDK-->>Go: page.Value
    end
    Go-->>App: 全件のslice または error
```

## モデルデプロイの作成／変更

```mermaid
sequenceDiagram
    participant App
    participant Go as AzureGo
    participant SDK as Azure SDK
    participant ARM
    App->>Go: Deployment.Create(ctx, ..., options)
    Go->>Go: 必須値・正のcapacityを確認
    Go->>SDK: BeginCreateOrUpdate
    SDK->>ARM: PUT deployment
    ARM-->>SDK: 受付 または 即時完了
    Go->>SDK: PollUntilDone(ctx)
    loop 必要な場合のみ完了を監視
        SDK->>ARM: 状態確認
        ARM-->>SDK: 状態
    end
    SDK-->>Go: 完了結果 または error
    Go-->>App: Deployment または error
```

`Update` はSKU／容量だけのPATCH、`Delete` はDELETEです。いずれもSDKの完了待ちを使用します。
認証の図示は一覧取得と同じため省略しています。

## 設計上の境界

`New` はローカルな構築だけを行います。Subscription未指定でも名前空間を作成し、
スコープが必要な操作で明示的なエラーを返します。CLIの既定Subscriptionの自動選択はしません。

公開モデルはSDK型のエイリアス、操作の戻り値はモデルまたはスライスです。
全ページのメモリ保持が適さない規模になった場合に限り、別途ストリーミングAPIを検討します。

独自Repository／Provider抽象化、バックグラウンド更新、データキャッシュ、永続化、
秘密情報保管、汎用CLIラッパーはありません。
