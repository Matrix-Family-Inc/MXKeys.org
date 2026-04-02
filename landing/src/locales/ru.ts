/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Tue Jan 27 2026 UTC
 * Status: Created
 */

export const ru = {
  nav: {
    home: 'Главная',
    about: 'О сервисе',
    howItWorks: 'Как работает',
    api: 'API',
    ecosystem: 'Экосистема',
  },

  hero: {
    title: 'MXKeys',
    subtitle: 'Federation Trust Infrastructure',
    tagline: 'Trust. Verify. Federate.',
    description: 'Комплексная инфраструктура доверия для федерации Matrix: верификация ключей, transparency log, обнаружение аномалий и распределённая координация кластеров. Любой Matrix-сервер может использовать MXKeys как trusted key server.',
    trust: 'Core notary service is production-deployed. Security-hardened. Tested under load and failure scenarios.',
    learnMore: 'Подробнее',
    viewAPI: 'Смотреть API',
  },

  status: {
    online: 'Infrastructure Online',
  },

  about: {
    title: 'Что такое MXKeys?',
    description: 'MXKeys — это Matrix Federation Trust Infrastructure — комплексная инфраструктура доверия, помогающая Matrix-серверам верифицировать ключи, отслеживать изменения, обнаруживать аномалии и применять политики доверия.',
    
    problem: {
      title: 'Проблема',
      description: 'Федерация Matrix зависит от ключей серверов с ограниченной видимостью. Ротацию ключей сложно отследить, компрометированные серверы трудно обнаружить, нет аудит-лога изменений ключей. Доверие неявное.',
    },
    
    solution: {
      title: 'Решение',
      description: 'MXKeys предоставляет комплексную инфраструктуру доверия: верификация ключей с perspective signatures, append-only transparency log с Merkle proofs, обнаружение аномалий, настраиваемые политики доверия и распределённые режимы notary cluster.',
    },
  },

  features: {
    title: 'Возможности',
    description: 'Функции проверки ключей для федерации Matrix.',
    
    caching: {
      title: 'Key Caching',
      description: 'Хранит проверенные ключи в PostgreSQL. Снижает задержки и нагрузку на серверы-источники.',
    },
    verification: {
      title: 'Signature Verification',
      description: 'Валидирует все полученные ключи по подписям серверов перед кэшированием.',
    },
    perspective: {
      title: 'Perspective Signing',
      description: 'Добавляет notary co-signature (ed25519:mxkeys) к проверенным ключам — независимое подтверждение.',
    },
    discovery: {
      title: 'Server Discovery',
      description: 'Полная поддержка спецификации: .well-known делегирование, SRV записи (_matrix-fed._tcp), IP литералы, автоматический fallback порта.',
    },
    fallback: {
      title: 'Fallback Support',
      description: 'Если прямой запрос не удался, обращается к fallback notary. Никакой единой точки доверия.',
    },
    performance: {
      title: 'High Performance',
      description: 'Написан на Go. Кэширование в памяти, пул соединений, эффективная очистка. Один бинарник с минимальными зависимостями.',
    },
    opensource: {
      title: 'Open Source',
      description: 'Аудируемый код. Никакой скрытой логики, никаких проприетарных зависимостей.',
    },
  },

  howItWorks: {
    title: 'Как это работает',
    description: 'Процесс проверки ключей.',
    
    steps: {
      request: {
        title: '1. Request',
        description: 'Matrix-сервер запрашивает у MXKeys ключи другого сервера через POST /_matrix/key/v2/query',
      },
      cache: {
        title: '2. Cache Check',
        description: 'MXKeys проверяет кэш в памяти, затем PostgreSQL. Если есть валидный кэш — возвращает сразу.',
      },
      fetch: {
        title: '3. Server Discovery',
        description: 'При cache miss MXKeys определяет адрес сервера через .well-known, SRV записи и fallback порта — затем запрашивает ключи через /_matrix/key/v2/server',
      },
      verify: {
        title: '4. Verify',
        description: 'MXKeys проверяет self-signature сервера с помощью Ed25519. Невалидные подписи отклоняются.',
      },
      sign: {
        title: '5. Co-Sign',
        description: 'MXKeys добавляет свою perspective signature (ed25519:mxkeys) — подтверждение проверки ключей.',
      },
      respond: {
        title: '6. Respond',
        description: 'Ключи с original и notary signature возвращаются запрашивающему серверу.',
      },
    },
  },

  api: {
    title: 'API Endpoints',
    description: 'MXKeys реализует Matrix Key Server API и operational probes.',
    
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'Возвращает публичные ключи MXKeys. Используется для проверки подписей.',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'Возвращает конкретный ключ MXKeys по key ID. При отсутствии возвращает M_NOT_FOUND.',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'Основной notary endpoint. Запрашивает ключи Matrix-серверов и возвращает проверенные ключи с co-signature MXKeys.',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'Server version.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'Liveness endpoint. Возвращает healthy, если процесс работает.',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'Readiness endpoint. Проверяет доступность БД и активный signing key.',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'Liveness probe endpoint. Возвращает состояние process alive.',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'Детальный статус сервиса: uptime, cache metrics и статистика подключений к БД.',
    },
    errorsTitle: 'Error model',
    errorsDescription: 'Валидация запросов использует Matrix-compatible коды ошибок: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE.',
  },

  integration: {
    title: 'Интеграция',
    description: 'Настройте ваш Matrix-сервер использовать MXKeys как trusted key server.',
    
    synapse: 'Synapse Configuration',
    mxcore: 'MXCore Configuration',
  },

  ecosystem: {
    title: 'Часть Matrix Family',
    description: 'MXKeys разрабатывается Matrix Family Inc. Доступен для всех Matrix-серверов.',
    
    matrixFamily: {
      title: 'Matrix Family',
      description: 'Ecosystem Hub',
    },
    hushme: {
      title: 'HushMe',
      description: 'Matrix Client',
    },
    hushmeStore: {
      title: 'HushMe Store',
      description: 'MFOS Apps',
    },
    mxcore: {
      title: 'MXCore',
      description: 'Matrix Homeserver',
    },
    mfos: {
      title: 'MFOS',
      description: 'Developer Platform',
    },
  },

  footer: {
    ecosystem: 'Экосистема',
    resources: 'Ресурсы',
    contact: 'Контакты',
    
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    
    architecture: 'Architecture',
    apiReference: 'API Reference',
    
    support: '@support',
    developer: '@dev',
    
    protocol: 'Протокол',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',

    copyrightPrefix: '© 2026 Matrix Family Inc. All rights reserved. Часть экосистемы ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: '.',
    tagline: 'Key Notary for Matrix federation.',
  },
};
