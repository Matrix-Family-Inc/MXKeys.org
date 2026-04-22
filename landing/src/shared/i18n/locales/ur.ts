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
    home: 'ہوم',
    about: 'تعارف',
    howItWorks: 'یہ کیسے کام کرتا ہے',
    api: 'API',
    ecosystem: 'ایکوسسٹم',
    homeAria: 'MXKeys ہوم',
    github: 'MXKeys GitHub ریپوزٹری',
    language: 'زبان',
    openMenu: 'نیویگیشن مینو کھولیں',
    closeMenu: 'نیویگیشن مینو بند کریں',
  },

  hero: {
    title: 'MXKeys',
    subtitle: 'فیڈریشن ٹرسٹ انفراسٹرکچر',
    tagline: 'اعتماد۔ تصدیق۔ فیڈریشن۔',
    description: 'Matrix کے لیے فیڈریشن کی ٹرسٹ لیئر: کلید کی تصدیق، شفافیت لاگنگ، بے قاعدگی کا پتہ لگانا، اور تصدیق شدہ کلسٹر کوآرڈینیشن۔',
    trust: 'PostgreSQL کیشنگ، Matrix-spec ڈسکوری، اور آپریشنل اینڈ پوائنٹس کے ساتھ Go سروس۔',
    learnMore: 'مزید جانیں',
    viewAPI: 'API دیکھیں',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },

  status: {
    online: 'انفراسٹرکچر آن لائن',
  },

  about: {
    title: 'MXKeys کیا ہے؟',
    description: 'MXKeys ایک Matrix فیڈریشن ٹرسٹ انفراسٹرکچر ہے جو Matrix سرورز کو شناخت کی تصدیق، کلید کی تبدیلیوں کو ٹریک کرنے، بے قاعدگیوں کا پتہ لگانے، اور ٹرسٹ پالیسیاں نافذ کرنے میں مدد کرتا ہے۔',
    
    problem: {
      title: 'مسئلہ',
      description: 'Matrix فیڈریشن محدود مرئیت والی سرور کلیدوں پر انحصار کرتی ہے۔ کلید کی گردش کو ٹریک کرنا مشکل ہے، سمجھوتہ شدہ سرورز کا پتہ لگانا مشکل ہے، اور کلید کی تبدیلیوں کے لیے کوئی آڈٹ ٹریل نہیں ہے۔ اعتماد مضمر ہے۔',
    },
    
    solution: {
      title: 'حل',
      description: 'MXKeys پرسپیکٹو دستخطوں کے ساتھ کلید کی تصدیق، Merkle ثبوتوں کے ساتھ ہیش چینڈ شفافیت لاگ، بے قاعدگی کا پتہ لگانا، قابل ترتیب ٹرسٹ پالیسیاں، اور تصدیق شدہ کلسٹر موڈز فراہم کرتا ہے۔',
    },
  },

  features: {
    title: 'خصوصیات',
    description: 'Matrix فیڈریشن کے لیے کلید کی تصدیق کی صلاحیتیں۔',
    
    caching: {
      title: 'کلید کیشنگ',
      description: 'تصدیق شدہ کلیدوں کو PostgreSQL میں محفوظ کرتا ہے۔ اصل سرورز پر تاخیر اور بوجھ کم کرتا ہے۔',
    },
    verification: {
      title: 'دستخط کی تصدیق',
      description: 'کیشنگ سے پہلے سرور دستخطوں کے خلاف تمام حاصل کی گئی کلیدوں کی توثیق کرتا ہے۔',
    },
    perspective: {
      title: 'پرسپیکٹو سائننگ',
      description: 'تصدیق شدہ کلیدوں میں ایک نوٹری شریک دستخط (ed25519:mxkeys) شامل کرتا ہے — ایک آزاد تصدیق۔',
    },
    discovery: {
      title: 'سرور ڈسکوری',
      description: 'MXKeys کی-نوٹری دائرے میں .well-known ڈیلیگیشن، SRV ریکارڈز (_matrix-fed._tcp)، IP لٹرلز، اور پورٹ فال بیک کے لیے Matrix ڈسکوری سپورٹ۔',
    },
    fallback: {
      title: 'فال بیک سپورٹ',
      description: 'اگر براہ راست فیچ ناکام ہو جائے، تو MXKeys ایک واضح آپریشنل ٹرسٹ پاتھ کے طور پر ترتیب شدہ فال بیک نوٹریز سے استفسار کر سکتا ہے۔',
    },
    performance: {
      title: 'اعلیٰ کارکردگی',
      description: 'Go میں لکھا گیا۔ میموری کیشنگ، کنکشن پولنگ، موثر صفائی، اور سنگل بائنری ڈیپلائمنٹ۔',
    },
    opensource: {
      title: 'اوپن سورس',
      description: 'قابل آڈٹ کوڈ۔ کوئی پوشیدہ لاجک نہیں، کوئی پروپرائٹری انحصار نہیں۔',
    },
  },

  howItWorks: {
    title: 'یہ کیسے کام کرتا ہے',
    description: 'کلید کی تصدیق کا عمل۔',
    
    steps: {
      request: {
        title: '1. درخواست',
        description: 'ایک Matrix سرور POST /_matrix/key/v2/query کے ذریعے دوسرے سرور کی کلیدوں کے لیے MXKeys سے استفسار کرتا ہے',
      },
      cache: {
        title: '2. کیش چیک',
        description: 'MXKeys پہلے میموری کیش، پھر PostgreSQL چیک کرتا ہے۔ اگر درست کیشڈ کلید موجود ہو — فوری طور پر واپس کرتا ہے۔',
      },
      fetch: {
        title: '3. سرور ڈسکوری',
        description: 'کیش مس پر، MXKeys .well-known ڈیلیگیشن، SRV ریکارڈز، اور پورٹ فال بیک استعمال کرتے ہوئے ہدف سرور کو ریزالو کرتا ہے — پھر /_matrix/key/v2/server کے ذریعے کلیدیں حاصل کرتا ہے',
      },
      verify: {
        title: '4. تصدیق',
        description: 'MXKeys Ed25519 استعمال کرتے ہوئے سرور کے سیلف سگنیچر کی تصدیق کرتا ہے۔ غلط دستخط مسترد کر دیے جاتے ہیں۔',
      },
      sign: {
        title: '5. شریک دستخط',
        description: 'MXKeys اپنا پرسپیکٹو دستخط (ed25519:mxkeys) شامل کرتا ہے — تصدیق کرتے ہوئے کہ اس نے کلیدوں کی تصدیق کی ہے۔',
      },
      respond: {
        title: '6. جواب',
        description: 'اصل اور نوٹری دونوں دستخطوں کے ساتھ کلیدیں درخواست کرنے والے سرور کو واپس کی جاتی ہیں۔',
      },
    },
  },

  api: {
    title: 'API اینڈ پوائنٹس',
    description: 'MXKeys Matrix Key Server API اور آپریشنل پروبز نافذ کرتا ہے۔',
    
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'MXKeys پبلک کلیدیں واپس کرتا ہے۔ دستخطوں کی تصدیق کے لیے استعمال ہوتا ہے۔',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'کلید ID کے ذریعے ایک مخصوص MXKeys کلید واپس کرتا ہے۔ کلید غیر موجود ہونے پر M_NOT_FOUND کے ساتھ جواب دیتا ہے۔',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'مرکزی نوٹری اینڈ پوائنٹ۔ Matrix سرورز کی کلیدوں کا استفسار کرتا ہے اور MXKeys شریک دستخط کے ساتھ تصدیق شدہ کلیدیں واپس کرتا ہے۔',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'سرور ورژن کی معلومات۔',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'ہیلتھ اینڈ پوائنٹ۔ سروس ہیلتھ میٹا ڈیٹا واپس کرتا ہے۔',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'ریڈینس اینڈ پوائنٹ۔ DB کنیکٹیویٹی اور فعال سائننگ کلید کی تصدیق کرتا ہے۔',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'لائیونس پروب اینڈ پوائنٹ۔ پروسیس الائیو سٹیٹ واپس کرتا ہے۔',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'تفصیلی سروس سٹیٹس: اپ ٹائم، کیش میٹرکس، ڈیٹابیس اسٹیٹس، اور اختیاری ذیلی نظاموں کی حالت۔',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'سروس اور رن ٹائم ٹیلی میٹری کے لیے Prometheus میٹرکس ایکسپوزیشن۔',
    },
    errorsTitle: 'ایرر ماڈل',
    errorsDescription: 'درخواست کی توثیق اور غلط استعمال کے کنٹرولز Matrix-compatible ایرر کوڈز استعمال کرتے ہیں: M_BAD_JSON، M_INVALID_PARAM، M_NOT_FOUND، M_TOO_LARGE، اور M_LIMIT_EXCEEDED۔',
    protectedTitle: 'صرف ایڈمن کے آپریشنل روٹس',
    protectedDescription: 'شفافیت، تجزیات، کلسٹر، اور پالیسی روٹس صرف ایڈمن کے ops/debug سطحی سرفیس ہیں۔ انہیں bearer ٹوکن (security.admin_access_token) کے ذریعے محفوظ کیا گیا ہے اور یہ مستحکم پبلک فیڈریشن API سے باہر ہیں۔',
  },

  integration: {
    title: 'انٹیگریشن',
    description: 'اپنے Matrix سرور کو MXKeys کو قابل اعتماد کلید سرور کے طور پر استعمال کرنے کے لیے ترتیب دیں۔',
    
    synapse: 'Synapse کنفیگریشن',
    mxcore: 'MXCore کنفیگریشن',
  },

  ecosystem: {
    title: 'Matrix Family کا حصہ',
    description: 'MXKeys کو Matrix Family Inc. نے تیار کیا ہے۔ تمام Matrix سرورز کے لیے دستیاب۔',
    
    matrixFamily: {
      title: 'Matrix Family',
      description: 'ایکوسسٹم ہب',
    },
    hushme: {
      title: 'HushMe',
      description: 'Matrix کلائنٹ',
    },
    hushmeStore: {
      title: 'HushMe Store',
      description: 'MFOS ایپس',
    },
    mxcore: {
      title: 'MXCore',
      description: 'Matrix ہوم سرور',
    },
    mfos: {
      title: 'MFOS',
      description: 'ڈیولپر پلیٹ فارم',
    },
  },

  footer: {
    ecosystem: 'ایکوسسٹم',
    resources: 'وسائل',
    contact: 'رابطہ',
    
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    
    architecture: 'آرکیٹیکچر',
    apiReference: 'API حوالہ',
    
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    
    protocol: 'پروٹوکول',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',

    copyrightPrefix: '© 2026 Matrix Family Inc. جملہ حقوق محفوظ ہیں۔ ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: ' ایکوسسٹم کا حصہ۔',
    tagline: 'Matrix فیڈریشن کے لیے Key Notary۔',
  },
};
