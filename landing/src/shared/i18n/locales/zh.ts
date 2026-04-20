/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Sun Apr 13 2026 UTC
 * Status: Created
 */

export default {
  nav: {
    home: '首页',
    about: '关于',
    howItWorks: '工作原理',
    api: 'API',
    ecosystem: '生态系统',
    homeAria: 'MXKeys 首页',
    github: 'MXKeys GitHub 仓库',
    language: '语言',
    openMenu: '打开导航菜单',
    closeMenu: '关闭导航菜单',
  },
  hero: {
    title: 'MXKeys',
    subtitle: '联邦信任基础设施',
    tagline: '信任。验证。联邦。',
    description: 'Matrix 联邦密钥信任层：密钥验证、透明日志记录、异常检测和经过身份验证的集群协调。',
    trust: 'Go 服务，具备 PostgreSQL 缓存、Matrix 规范发现和运维端点。',
    learnMore: '了解更多',
    viewAPI: '查看 API',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },
  status: {
    online: '基础设施在线',
  },
  about: {
    title: '什么是 MXKeys？',
    description: 'MXKeys 是一个 Matrix 联邦信任基础设施，帮助 Matrix 服务器验证身份、跟踪密钥变更、检测异常并实施信任策略。',
    problem: {
      title: '问题',
      description: 'Matrix 联邦依赖可见性有限的服务器密钥。密钥轮换难以追踪，受损服务器难以检测，且密钥变更没有审计记录。信任是隐式的。',
    },
    solution: {
      title: '解决方案',
      description: 'MXKeys 提供带有视角签名的密钥验证、基于哈希链的透明日志及 Merkle 证明、异常检测、可配置的信任策略和经过身份验证的集群模式。',
    },
  },
  features: {
    title: '功能特性',
    description: 'Matrix 联邦的密钥验证能力。',
    caching: {
      title: '密钥缓存',
      description: '在 PostgreSQL 中存储已验证的密钥。减少延迟和源服务器的负载。',
    },
    verification: {
      title: '签名验证',
      description: '在缓存之前，针对服务器签名验证所有获取的密钥。',
    },
    perspective: {
      title: '视角签名',
      description: '为已验证的密钥添加公证联合签名 (ed25519:mxkeys) — 独立的证明。',
    },
    discovery: {
      title: '服务器发现',
      description: '在 MXKeys 密钥公证范围内支持 Matrix 发现：.well-known 委托、SRV 记录 (_matrix-fed._tcp)、IP 字面量和端口回退。',
    },
    fallback: {
      title: '回退支持',
      description: '如果直接获取失败，MXKeys 可以查询已配置的回退公证人作为显式运维信任路径。',
    },
    performance: {
      title: '高性能',
      description: '使用 Go 编写。内存缓存、连接池、高效清理和单二进制部署。',
    },
    opensource: {
      title: '开源',
      description: '可审计代码。无隐藏逻辑，无专有依赖。',
    },
  },
  howItWorks: {
    title: '工作原理',
    description: '密钥验证流程。',
    steps: {
      request: {
        title: '1. 请求',
        description: 'Matrix 服务器通过 POST /_matrix/key/v2/query 向 MXKeys 查询另一个服务器的密钥',
      },
      cache: {
        title: '2. 缓存检查',
        description: 'MXKeys 检查内存缓存，然后检查 PostgreSQL。如果存在有效的缓存密钥 — 立即返回。',
      },
      fetch: {
        title: '3. 服务器发现',
        description: '缓存未命中时，MXKeys 通过 .well-known 委托、SRV 记录和端口回退解析目标服务器 — 然后通过 /_matrix/key/v2/server 获取密钥',
      },
      verify: {
        title: '4. 验证',
        description: 'MXKeys 使用 Ed25519 验证服务器的自签名。无效签名将被拒绝。',
      },
      sign: {
        title: '5. 联合签名',
        description: 'MXKeys 添加其视角签名 (ed25519:mxkeys) — 证明已验证密钥。',
      },
      respond: {
        title: '6. 响应',
        description: '带有原始签名和公证签名的密钥返回给请求服务器。',
      },
    },
  },
  api: {
    title: 'API 端点',
    description: 'MXKeys 实现 Matrix Key Server API 和运维探针。',
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: '返回 MXKeys 公钥。用于验证签名。',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: '按密钥 ID 返回特定的 MXKeys 密钥。密钥不存在时返回 M_NOT_FOUND。',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: '主公证端点。查询 Matrix 服务器的密钥并返回带有 MXKeys 联合签名的已验证密钥。',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: '服务器版本信息。',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: '健康端点。返回服务健康元数据。',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: '就绪端点。验证数据库连接和活动签名密钥。',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: '存活探针端点。返回进程存活状态。',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: '详细服务状态：运行时间、缓存指标、数据库统计和可选的企业功能状态。',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'Prometheus 指标暴露，用于服务和运行时遥测。',
    },
    errorsTitle: '错误模型',
    errorsDescription: '请求验证和滥用控制使用 Matrix 兼容的错误代码：M_BAD_JSON、M_INVALID_PARAM、M_NOT_FOUND、M_TOO_LARGE 和 M_LIMIT_EXCEEDED。',
    protectedTitle: '受保护的运维路由',
    protectedDescription: '透明、分析、集群和策略路由需要企业访问令牌，与稳定的公共联邦 API 分开记录。',
  },
  integration: {
    title: '集成',
    description: '配置您的 Matrix 服务器以使用 MXKeys 作为受信任的密钥服务器。',
    synapse: 'Synapse 配置',
    mxcore: 'MXCore 配置',
  },
  ecosystem: {
    title: 'Matrix Family 的一部分',
    description: 'MXKeys 由 Matrix Family Inc. 开发。适用于所有 Matrix 服务器。',
    matrixFamily: { title: 'Matrix Family', description: '生态系统中心' },
    hushme: { title: 'HushMe', description: 'Matrix 客户端' },
    hushmeStore: { title: 'HushMe Store', description: 'MFOS 应用' },
    mxcore: { title: 'MXCore', description: 'Matrix 家庭服务器' },
    mfos: { title: 'MFOS', description: '开发者平台' },
  },
  footer: {
    ecosystem: '生态系统',
    resources: '资源',
    contact: '联系方式',
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    architecture: '架构',
    apiReference: 'API 参考',
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    protocol: '协议',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',
    copyrightPrefix: '© 2026 Matrix Family Inc. 保留所有权利。属于 ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: ' 生态系统。',
    tagline: 'Matrix 联邦的密钥公证服务。',
  },
};
