/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Sun Apr 13 2026 UTC
 * Status: Created
 */

export const ja = {
  nav: {
    home: 'ホーム',
    about: '概要',
    howItWorks: '仕組み',
    api: 'API',
    ecosystem: 'エコシステム',
    homeAria: 'MXKeys ホーム',
    github: 'MXKeys GitHub リポジトリ',
    language: '言語',
    openMenu: 'ナビゲーションメニューを開く',
    closeMenu: 'ナビゲーションメニューを閉じる',
  },
  hero: {
    title: 'MXKeys',
    subtitle: 'フェデレーション信頼インフラストラクチャ',
    tagline: '信頼。検証。フェデレーション。',
    description: 'Matrix のフェデレーション鍵信頼レイヤー：鍵検証、透明性ログ、異常検知、認証済みクラスター連携。',
    trust: 'PostgreSQL キャッシュ、Matrix 仕様ディスカバリ、運用エンドポイントを備えた Go サービス。',
    learnMore: '詳細を見る',
    viewAPI: 'API を見る',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },
  status: {
    online: 'インフラストラクチャ オンライン',
  },
  about: {
    title: 'MXKeys とは？',
    description: 'MXKeys は Matrix フェデレーション信頼インフラストラクチャです。Matrix サーバーの ID 検証、鍵変更の追跡、異常検知、信頼ポリシーの適用を支援します。',
    problem: {
      title: '課題',
      description: 'Matrix フェデレーションは可視性が限られたサーバー鍵に依存しています。鍵のローテーションは追跡が困難で、侵害されたサーバーの検出は難しく、鍵変更の監査証跡がありません。信頼は暗黙的です。',
    },
    solution: {
      title: '解決策',
      description: 'MXKeys はパースペクティブ署名による鍵検証、Merkle 証明付きハッシュチェーン透明性ログ、異常検知、設定可能な信頼ポリシー、認証済みクラスターモードを提供します。',
    },
  },
  features: {
    title: '機能',
    description: 'Matrix フェデレーションのための鍵検証機能。',
    caching: {
      title: '鍵キャッシュ',
      description: '検証済み鍵を PostgreSQL に格納。レイテンシとオリジンサーバーへの負荷を低減します。',
    },
    verification: {
      title: '署名検証',
      description: 'キャッシュ前に、取得したすべての鍵をサーバー署名に対して検証します。',
    },
    perspective: {
      title: 'パースペクティブ署名',
      description: '検証済み鍵に公証共同署名 (ed25519:mxkeys) を付加 — 独立した証明です。',
    },
    discovery: {
      title: 'サーバーディスカバリ',
      description: 'MXKeys 鍵公証スコープ内での Matrix ディスカバリ対応：.well-known 委任、SRV レコード (_matrix-fed._tcp)、IP リテラル、ポートフォールバック。',
    },
    fallback: {
      title: 'フォールバックサポート',
      description: '直接取得に失敗した場合、MXKeys は設定済みフォールバック公証人に明示的な運用信頼パスとして問い合わせることができます。',
    },
    performance: {
      title: '高パフォーマンス',
      description: 'Go で記述。メモリキャッシュ、コネクションプーリング、効率的なクリーンアップ、シングルバイナリデプロイ。',
    },
    opensource: {
      title: 'オープンソース',
      description: '監査可能なコード。隠れたロジックなし、プロプライエタリな依存関係なし。',
    },
  },
  howItWorks: {
    title: '仕組み',
    description: '鍵検証フロー。',
    steps: {
      request: {
        title: '1. リクエスト',
        description: 'Matrix サーバーが POST /_matrix/key/v2/query を介して別のサーバーの鍵を MXKeys に問い合わせます',
      },
      cache: {
        title: '2. キャッシュ確認',
        description: 'MXKeys はメモリキャッシュ、次に PostgreSQL を確認します。有効なキャッシュ済み鍵が存在する場合 — 即座に返却。',
      },
      fetch: {
        title: '3. サーバーディスカバリ',
        description: 'キャッシュミス時、MXKeys は .well-known 委任、SRV レコード、ポートフォールバックを使用して対象サーバーを解決し、/_matrix/key/v2/server 経由で鍵を取得します',
      },
      verify: {
        title: '4. 検証',
        description: 'MXKeys は Ed25519 を使用してサーバーの自己署名を検証します。無効な署名は拒否されます。',
      },
      sign: {
        title: '5. 共同署名',
        description: 'MXKeys がパースペクティブ署名 (ed25519:mxkeys) を付加 — 鍵を検証したことを証明します。',
      },
      respond: {
        title: '6. レスポンス',
        description: 'オリジナル署名と公証署名の両方を含む鍵がリクエスト元サーバーに返却されます。',
      },
    },
  },
  api: {
    title: 'API エンドポイント',
    description: 'MXKeys は Matrix Key Server API と運用プローブを実装しています。',
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'MXKeys の公開鍵を返します。署名の検証に使用されます。',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: '鍵 ID で特定の MXKeys 鍵を返します。鍵が存在しない場合、M_NOT_FOUND を返します。',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'メイン公証エンドポイント。Matrix サーバーの鍵を問い合わせ、MXKeys 共同署名付きの検証済み鍵を返します。',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'サーバーバージョン情報。',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'ヘルスエンドポイント。サービスヘルスメタデータを返します。',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'レディネスエンドポイント。DB 接続とアクティブな署名鍵を検証します。',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'ライブネスプローブエンドポイント。プロセスの生存状態を返します。',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: '詳細なサービスステータス：稼働時間、キャッシュメトリクス、データベース統計、およびオプションのエンタープライズ機能ステータス。',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'サービスおよびランタイムテレメトリ用の Prometheus メトリクスエクスポジション。',
    },
    errorsTitle: 'エラーモデル',
    errorsDescription: 'リクエスト検証と不正利用制御は Matrix 互換のエラーコードを使用します：M_BAD_JSON、M_INVALID_PARAM、M_NOT_FOUND、M_TOO_LARGE、M_LIMIT_EXCEEDED。',
    protectedTitle: '保護された運用ルート',
    protectedDescription: '透明性、分析、クラスター、ポリシーのルートにはエンタープライズアクセストークンが必要で、安定した公開フェデレーション API とは別に文書化されています。',
  },
  integration: {
    title: 'インテグレーション',
    description: 'MXKeys を信頼できる鍵サーバーとして使用するよう Matrix サーバーを構成します。',
    synapse: 'Synapse 設定',
    mxcore: 'MXCore 設定',
  },
  ecosystem: {
    title: 'Matrix Family の一員',
    description: 'MXKeys は Matrix Family Inc. が開発しています。すべての Matrix サーバーで利用可能です。',
    matrixFamily: { title: 'Matrix Family', description: 'エコシステムハブ' },
    hushme: { title: 'HushMe', description: 'Matrix クライアント' },
    hushmeStore: { title: 'HushMe Store', description: 'MFOS アプリ' },
    mxcore: { title: 'MXCore', description: 'Matrix ホームサーバー' },
    mfos: { title: 'MFOS', description: '開発者プラットフォーム' },
  },
  footer: {
    ecosystem: 'エコシステム',
    resources: 'リソース',
    contact: 'お問い合わせ',
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    architecture: 'アーキテクチャ',
    apiReference: 'API リファレンス',
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    protocol: 'プロトコル',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',
    copyrightPrefix: '© 2026 Matrix Family Inc. All rights reserved. ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: ' エコシステムの一員です。',
    tagline: 'Matrix フェデレーションのための鍵公証サービス。',
  },
};
