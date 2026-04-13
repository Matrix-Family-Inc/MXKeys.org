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

export const uk = {
  nav: {
    home: 'Головна',
    about: 'Про нас',
    howItWorks: 'Як це працює',
    api: 'API',
    ecosystem: 'Екосистема',
    homeAria: 'Головна MXKeys',
    github: 'GitHub-репозиторій MXKeys',
    language: 'Мова',
    openMenu: 'Відкрити меню навігації',
    closeMenu: 'Закрити меню навігації',
  },
  hero: {
    title: 'MXKeys',
    subtitle: 'Інфраструктура довіри федерації',
    tagline: 'Довіра. Верифікація. Федерація.',
    description: 'Рівень довіри ключів федерації для Matrix: верифікація ключів, журналювання прозорості, виявлення аномалій та автентифікована координація кластерів.',
    trust: 'Go-сервіс з кешуванням PostgreSQL, виявленням відповідно до специфікації Matrix та операційними ендпоінтами.',
    learnMore: 'Дізнатися більше',
    viewAPI: 'Переглянути API',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },
  status: {
    online: 'Інфраструктура онлайн',
  },
  about: {
    title: 'Що таке MXKeys?',
    description: 'MXKeys — це інфраструктура довіри федерації Matrix, яка допомагає серверам Matrix верифікувати ідентичності, відстежувати зміни ключів, виявляти аномалії та застосовувати політики довіри.',
    problem: {
      title: 'Проблема',
      description: 'Федерація Matrix покладається на серверні ключі з обмеженою видимістю. Ротацію ключів важко відстежувати, скомпрометовані сервери важко виявити, а для змін ключів немає аудиторського сліду. Довіра є неявною.',
    },
    solution: {
      title: 'Рішення',
      description: 'MXKeys забезпечує верифікацію ключів з перспективними підписами, хеш-ланцюговий журнал прозорості з доказами Merkle, виявлення аномалій, налаштовувані політики довіри та автентифіковані режими кластерів.',
    },
  },
  features: {
    title: 'Функції',
    description: 'Можливості верифікації ключів для федерації Matrix.',
    caching: {
      title: 'Кешування ключів',
      description: 'Зберігає верифіковані ключі в PostgreSQL. Зменшує затримку та навантаження на сервери-джерела.',
    },
    verification: {
      title: 'Верифікація підписів',
      description: 'Перевіряє всі отримані ключі за серверними підписами перед кешуванням.',
    },
    perspective: {
      title: 'Перспективний підпис',
      description: 'Додає нотаріальний спільний підпис (ed25519:mxkeys) до верифікованих ключів — незалежне засвідчення.',
    },
    discovery: {
      title: 'Виявлення серверів',
      description: 'Підтримка виявлення Matrix для делегування .well-known, записів SRV (_matrix-fed._tcp), IP-літералів та резервного порту в межах нотаріального сервісу MXKeys.',
    },
    fallback: {
      title: 'Резервна підтримка',
      description: 'Якщо пряме отримання не вдається, MXKeys може запитувати налаштованих резервних нотаріусів як явний операційний шлях довіри.',
    },
    performance: {
      title: 'Висока продуктивність',
      description: 'Написано на Go. Кешування в пам\'яті, пул з\'єднань, ефективне очищення та розгортання одним бінарним файлом.',
    },
    opensource: {
      title: 'Відкритий код',
      description: 'Код, що піддається аудиту. Жодної прихованої логіки, жодних пропрієтарних залежностей.',
    },
  },
  howItWorks: {
    title: 'Як це працює',
    description: 'Процес верифікації ключів.',
    steps: {
      request: {
        title: '1. Запит',
        description: 'Сервер Matrix запитує в MXKeys ключі іншого сервера через POST /_matrix/key/v2/query',
      },
      cache: {
        title: '2. Перевірка кешу',
        description: 'MXKeys перевіряє кеш пам\'яті, потім PostgreSQL. Якщо існує дійсний кешований ключ — негайне повернення.',
      },
      fetch: {
        title: '3. Виявлення сервера',
        description: 'При відсутності в кеші MXKeys визначає цільовий сервер за допомогою делегування .well-known, записів SRV та резервного порту — потім отримує ключі через /_matrix/key/v2/server',
      },
      verify: {
        title: '4. Верифікація',
        description: 'MXKeys верифікує самопідпис сервера за допомогою Ed25519. Недійсні підписи відхиляються.',
      },
      sign: {
        title: '5. Спільний підпис',
        description: 'MXKeys додає свій перспективний підпис (ed25519:mxkeys) — засвідчуючи, що він верифікував ключі.',
      },
      respond: {
        title: '6. Відповідь',
        description: 'Ключі з оригінальними та нотаріальними підписами повертаються серверу, що надіслав запит.',
      },
    },
  },
  api: {
    title: 'Ендпоінти API',
    description: 'MXKeys реалізує Matrix Key Server API та операційні зонди.',
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'Повертає публічні ключі MXKeys. Використовується для верифікації підписів.',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'Повертає конкретний ключ MXKeys за ідентифікатором ключа. Відповідає M_NOT_FOUND, коли ключ відсутній.',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'Основний нотаріальний ендпоінт. Запитує ключі серверів Matrix та повертає верифіковані ключі зі спільним підписом MXKeys.',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'Інформація про версію сервера.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'Ендпоінт стану здоров\'я. Повертає метадані стану здоров\'я сервісу.',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'Ендпоінт готовності. Перевіряє підключення до БД та активний ключ підпису.',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'Ендпоінт перевірки живості. Повертає стан живості процесу.',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'Детальний статус сервісу: час роботи, метрики кешу, статистика бази даних та опціональний статус enterprise-функцій.',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'Експозиція метрик Prometheus для телеметрії сервісу та середовища виконання.',
    },
    errorsTitle: 'Модель помилок',
    errorsDescription: 'Валідація запитів та контроль зловживань використовують Matrix-сумісні коди помилок: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE та M_LIMIT_EXCEEDED.',
    protectedTitle: 'Захищені операційні маршрути',
    protectedDescription: 'Маршрути прозорості, аналітики, кластера та політик потребують enterprise-токен доступу та документовані окремо від стабільного публічного API федерації.',
  },
  integration: {
    title: 'Інтеграція',
    description: 'Налаштуйте свій сервер Matrix для використання MXKeys як довіреного сервера ключів.',
    synapse: 'Конфігурація Synapse',
    mxcore: 'Конфігурація MXCore',
  },
  ecosystem: {
    title: 'Частина Matrix Family',
    description: 'MXKeys розроблено Matrix Family Inc. Доступний для всіх серверів Matrix.',
    matrixFamily: { title: 'Matrix Family', description: 'Хаб екосистеми' },
    hushme: { title: 'HushMe', description: 'Клієнт Matrix' },
    hushmeStore: { title: 'HushMe Store', description: 'Додатки MFOS' },
    mxcore: { title: 'MXCore', description: 'Домашній сервер Matrix' },
    mfos: { title: 'MFOS', description: 'Платформа розробника' },
  },
  footer: {
    ecosystem: 'Екосистема',
    resources: 'Ресурси',
    contact: 'Контакти',
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    architecture: 'Архітектура',
    apiReference: 'Довідка API',
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    protocol: 'Протокол',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',
    copyrightPrefix: '© 2026 Matrix Family Inc. Усі права захищено. Частина ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: ' екосистеми.',
    tagline: 'Нотаріальний сервіс ключів для федерації Matrix.',
  },
};
