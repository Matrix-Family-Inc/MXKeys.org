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
    home: 'الرئيسية',
    about: 'حول',
    howItWorks: 'كيف يعمل',
    api: 'API',
    ecosystem: 'المنظومة',
    homeAria: 'الصفحة الرئيسية لـ MXKeys',
    github: 'مستودع MXKeys على GitHub',
    language: 'اللغة',
    openMenu: 'فتح قائمة التنقل',
    closeMenu: 'إغلاق قائمة التنقل',
  },

  hero: {
    title: 'MXKeys',
    subtitle: 'بنية ثقة الاتحاد',
    tagline: 'ثقة. تحقق. اتحاد.',
    description: 'طبقة ثقة مفاتيح الاتحاد لـ Matrix: التحقق من المفاتيح، تسجيل الشفافية، كشف الشذوذ، وتنسيق المجموعات المُصادق عليها.',
    trust: 'خدمة Go مع تخزين PostgreSQL المؤقت، واكتشاف Matrix-spec، ونقاط النهاية التشغيلية.',
    learnMore: 'اعرف المزيد',
    viewAPI: 'عرض API',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },

  status: {
    online: 'البنية التحتية متصلة',
  },

  about: {
    title: 'ما هو MXKeys؟',
    description: 'MXKeys هو بنية ثقة اتحاد Matrix التي تساعد خوادم Matrix على التحقق من الهويات، وتتبع تغييرات المفاتيح، وكشف الشذوذ، وإنفاذ سياسات الثقة.',
    
    problem: {
      title: 'المشكلة',
      description: 'يعتمد اتحاد Matrix على مفاتيح الخوادم ذات الرؤية المحدودة. يصعب تتبع تدوير المفاتيح، ويصعب اكتشاف الخوادم المخترقة، ولا يوجد سجل تدقيق لتغييرات المفاتيح. الثقة ضمنية.',
    },
    
    solution: {
      title: 'الحل',
      description: 'يوفر MXKeys التحقق من المفاتيح مع توقيعات المنظور، وسجل شفافية مسلسل بالتجزئة مع إثباتات Merkle، وكشف الشذوذ، وسياسات ثقة قابلة للتكوين، وأوضاع مجموعات مُصادق عليها.',
    },
  },

  features: {
    title: 'الميزات',
    description: 'قدرات التحقق من المفاتيح لاتحاد Matrix.',
    
    caching: {
      title: 'تخزين المفاتيح المؤقت',
      description: 'يخزن المفاتيح المُتحقق منها في PostgreSQL. يقلل زمن الاستجابة والحمل على خوادم المصدر.',
    },
    verification: {
      title: 'التحقق من التوقيع',
      description: 'يتحقق من صحة جميع المفاتيح المجلوبة مقابل توقيعات الخادم قبل التخزين المؤقت.',
    },
    perspective: {
      title: 'توقيع المنظور',
      description: 'يضيف توقيعًا مشتركًا للموثق (ed25519:mxkeys) على المفاتيح المُتحقق منها — شهادة مستقلة.',
    },
    discovery: {
      title: 'اكتشاف الخادم',
      description: 'دعم اكتشاف Matrix لتفويض .well-known، وسجلات SRV (_matrix-fed._tcp)، وعناوين IP المباشرة، والرجوع للمنفذ ضمن نطاق موثق مفاتيح MXKeys.',
    },
    fallback: {
      title: 'دعم الرجوع',
      description: 'إذا فشل الجلب المباشر، يمكن لـ MXKeys الاستعلام من موثقين احتياطيين مُكوَّنين كمسار ثقة تشغيلي صريح.',
    },
    performance: {
      title: 'أداء عالٍ',
      description: 'مكتوب بلغة Go. تخزين مؤقت في الذاكرة، تجميع الاتصالات، تنظيف فعال، ونشر ثنائي واحد.',
    },
    opensource: {
      title: 'مفتوح المصدر',
      description: 'كود قابل للتدقيق. لا منطق مخفي، لا تبعيات مملوكة.',
    },
  },

  howItWorks: {
    title: 'كيف يعمل',
    description: 'مسار التحقق من المفاتيح.',
    
    steps: {
      request: {
        title: '1. الطلب',
        description: 'يستعلم خادم Matrix من MXKeys عن مفاتيح خادم آخر عبر POST /_matrix/key/v2/query',
      },
      cache: {
        title: '2. فحص التخزين المؤقت',
        description: 'يفحص MXKeys ذاكرة التخزين المؤقت، ثم PostgreSQL. إذا وُجد مفتاح مُخزَّن صالح — يُعاد فورًا.',
      },
      fetch: {
        title: '3. اكتشاف الخادم',
        description: 'عند عدم وجود تخزين مؤقت، يحل MXKeys الخادم المستهدف باستخدام تفويض .well-known، وسجلات SRV، والرجوع للمنفذ — ثم يجلب المفاتيح عبر /_matrix/key/v2/server',
      },
      verify: {
        title: '4. التحقق',
        description: 'يتحقق MXKeys من التوقيع الذاتي للخادم باستخدام Ed25519. تُرفض التوقيعات غير الصالحة.',
      },
      sign: {
        title: '5. التوقيع المشترك',
        description: 'يضيف MXKeys توقيع المنظور الخاص به (ed25519:mxkeys) — مُشهدًا أنه تحقق من المفاتيح.',
      },
      respond: {
        title: '6. الاستجابة',
        description: 'تُعاد المفاتيح مع التوقيعات الأصلية وتوقيعات الموثق إلى الخادم الطالب.',
      },
    },
  },

  api: {
    title: 'نقاط نهاية API',
    description: 'ينفذ MXKeys واجهة Matrix Key Server API ومسابر التشغيل.',
    
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'يعيد المفاتيح العامة لـ MXKeys. يُستخدم للتحقق من التوقيعات.',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'يعيد مفتاح MXKeys محددًا بمعرف المفتاح. يستجيب بـ M_NOT_FOUND عند غياب المفتاح.',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'نقطة نهاية الموثق الرئيسية. يستعلم عن مفاتيح خوادم Matrix ويعيد مفاتيح مُتحقق منها مع توقيع MXKeys المشترك.',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'معلومات إصدار الخادم.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'نقطة نهاية الصحة. تعيد بيانات صحة الخدمة الوصفية.',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'نقطة نهاية الجاهزية. تتحقق من اتصال قاعدة البيانات ومفتاح التوقيع النشط.',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'نقطة نهاية مسبار الحياة. تعيد حالة نشاط العملية.',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'حالة الخدمة التفصيلية: وقت التشغيل، مقاييس التخزين المؤقت، إحصائيات قاعدة البيانات، وحالة ميزات المؤسسة الاختيارية.',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'عرض مقاييس Prometheus للقياس عن بُعد للخدمة ووقت التشغيل.',
    },
    errorsTitle: 'نموذج الأخطاء',
    errorsDescription: 'يستخدم التحقق من الطلبات وضوابط إساءة الاستخدام رموز أخطاء متوافقة مع Matrix: M_BAD_JSON، M_INVALID_PARAM، M_NOT_FOUND، M_TOO_LARGE، وM_LIMIT_EXCEEDED.',
    protectedTitle: 'المسارات التشغيلية المحمية',
    protectedDescription: 'تتطلب مسارات الشفافية والتحليلات والمجموعات والسياسات رمز وصول مؤسسي وهي موثقة بشكل منفصل عن واجهة الاتحاد العامة المستقرة.',
  },

  integration: {
    title: 'التكامل',
    description: 'قم بتكوين خادم Matrix الخاص بك لاستخدام MXKeys كخادم مفاتيح موثوق.',
    
    synapse: 'تكوين Synapse',
    mxcore: 'تكوين MXCore',
  },

  ecosystem: {
    title: 'جزء من Matrix Family',
    description: 'MXKeys مطوَّر بواسطة Matrix Family Inc. متاح لجميع خوادم Matrix.',
    
    matrixFamily: {
      title: 'Matrix Family',
      description: 'مركز المنظومة',
    },
    hushme: {
      title: 'HushMe',
      description: 'عميل Matrix',
    },
    hushmeStore: {
      title: 'HushMe Store',
      description: 'تطبيقات MFOS',
    },
    mxcore: {
      title: 'MXCore',
      description: 'خادم Matrix المنزلي',
    },
    mfos: {
      title: 'MFOS',
      description: 'منصة المطورين',
    },
  },

  footer: {
    ecosystem: 'المنظومة',
    resources: 'الموارد',
    contact: 'التواصل',
    
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    
    architecture: 'الهندسة المعمارية',
    apiReference: 'مرجع API',
    
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    
    protocol: 'البروتوكول',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',

    copyrightPrefix: '© 2026 Matrix Family Inc. جميع الحقوق محفوظة. جزء من منظومة ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: '.',
    tagline: 'موثق المفاتيح لاتحاد Matrix.',
  },
};
