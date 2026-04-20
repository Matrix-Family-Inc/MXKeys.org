/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
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
    homeAria: 'MXKeys: на главную',
    github: 'Репозиторий MXKeys на GitHub',
    language: 'Язык',
    openMenu: 'Открыть меню навигации',
    closeMenu: 'Закрыть меню навигации',
  },

  hero: {
    title: 'MXKeys',
    subtitle: 'Инфраструктура доверия федерации',
    tagline: 'Доверяй. Проверяй. Федерализуй.',
    description: 'Инфраструктура доверия для федерации Matrix: проверка ключей, transparency log, обнаружение аномалий и аутентифицированная координация кластеров.',
    trust: 'Сервис на Go с PostgreSQL-кэшем, Matrix-spec discovery и operational endpoints.',
    learnMore: 'Подробнее',
    viewAPI: 'Смотреть API',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },

  status: {
    online: 'Инфраструктура доступна',
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
      description: 'MXKeys предоставляет проверку ключей с perspective signatures, hash-chained transparency log с Merkle proofs, обнаружение аномалий, настраиваемые политики доверия и аутентифицированные режимы кластера.',
    },
  },

  features: {
    title: 'Возможности',
    description: 'Функции проверки ключей для федерации Matrix.',
    
    caching: {
      title: 'Кэширование ключей',
      description: 'Хранит проверенные ключи в PostgreSQL. Снижает задержки и нагрузку на серверы-источники.',
    },
    verification: {
      title: 'Проверка подписей',
      description: 'Валидирует все полученные ключи по подписям серверов перед кэшированием.',
    },
    perspective: {
      title: 'Perspective-подпись',
      description: 'Добавляет notary co-signature (ed25519:mxkeys) к проверенным ключам — независимое подтверждение.',
    },
    discovery: {
      title: 'Поиск сервера',
      description: 'Поддержка Matrix discovery в пределах key-notary scope: .well-known делегирование, SRV записи (_matrix-fed._tcp), IP литералы и fallback порта.',
    },
    fallback: {
      title: 'Fallback-режим',
      description: 'Если прямой запрос не удался, MXKeys может обратиться к явно заданным fallback notary как к отдельному operational trust path.',
    },
    performance: {
      title: 'Производительность',
      description: 'Написан на Go. Кэширование в памяти, пул соединений, эффективная очистка и один бинарник для развёртывания.',
    },
    opensource: {
      title: 'Открытый код',
      description: 'Аудируемый код. Никакой скрытой логики, никаких проприетарных зависимостей.',
    },
  },

  howItWorks: {
    title: 'Как это работает',
    description: 'Процесс проверки ключей.',
    
    steps: {
      request: {
        title: '1. Запрос',
        description: 'Matrix-сервер запрашивает у MXKeys ключи другого сервера через POST /_matrix/key/v2/query',
      },
      cache: {
        title: '2. Проверка кэша',
        description: 'MXKeys проверяет кэш в памяти, затем PostgreSQL. Если есть валидный кэш — возвращает сразу.',
      },
      fetch: {
        title: '3. Поиск сервера',
        description: 'При cache miss MXKeys определяет адрес сервера через .well-known, SRV записи и fallback порта — затем запрашивает ключи через /_matrix/key/v2/server',
      },
      verify: {
        title: '4. Проверка',
        description: 'MXKeys проверяет self-signature сервера с помощью Ed25519. Невалидные подписи отклоняются.',
      },
      sign: {
        title: '5. Co-sign',
        description: 'MXKeys добавляет свою perspective signature (ed25519:mxkeys) — подтверждение проверки ключей.',
      },
      respond: {
        title: '6. Ответ',
        description: 'Ключи с original и notary signature возвращаются запрашивающему серверу.',
      },
    },
  },

  api: {
    title: 'API-маршруты',
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
      description: 'Версия сервера.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'Health endpoint. Возвращает метаданные о состоянии сервиса.',
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
      description: 'Детальный статус сервиса: uptime, cache metrics, статистика БД и состояние дополнительных подсистем.',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'Prometheus-выгрузка метрик сервиса и runtime-телеметрии.',
    },
    errorsTitle: 'Модель ошибок',
    errorsDescription: 'Валидация запросов и abuse controls используют Matrix-compatible коды ошибок: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE и M_LIMIT_EXCEEDED.',
    protectedTitle: 'Защищённые operational routes',
    protectedDescription: 'Маршруты transparency, analytics, cluster и policy требуют enterprise access token и документируются отдельно от стабильного public federation API.',
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
    devChat: '#dev',
    
    protocol: 'Протокол',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',

    copyrightPrefix: '© 2026 Matrix Family Inc. All rights reserved. Часть экосистемы ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: '.',
    tagline: 'Key Notary for Matrix federation.',
  },
};
