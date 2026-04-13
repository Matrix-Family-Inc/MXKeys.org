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

export const he = {
  nav: {
    home: 'בית',
    about: 'אודות',
    howItWorks: 'איך זה עובד',
    api: 'API',
    ecosystem: 'אקוסיסטם',
    homeAria: 'דף הבית של MXKeys',
    github: 'מאגר GitHub של MXKeys',
    language: 'שפה',
    openMenu: 'פתח תפריט ניווט',
    closeMenu: 'סגור תפריט ניווט',
  },

  hero: {
    title: 'MXKeys',
    subtitle: 'תשתית אמון פדרציה',
    tagline: 'אמון. אימות. פדרציה.',
    description: 'שכבת אמון מפתחות פדרציה עבור Matrix: אימות מפתחות, רישום שקיפות, זיהוי חריגות ותיאום אשכולות מאומת.',
    trust: 'שירות Go עם מטמון PostgreSQL, גילוי Matrix-spec ונקודות קצה תפעוליות.',
    learnMore: 'למד עוד',
    viewAPI: 'צפה ב-API',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },

  status: {
    online: 'תשתית מקוונת',
  },

  about: {
    title: 'מהו MXKeys?',
    description: 'MXKeys הוא תשתית אמון פדרציה של Matrix המסייעת לשרתי Matrix לאמת זהויות, לעקוב אחר שינויי מפתחות, לזהות חריגות ולאכוף מדיניות אמון.',
    
    problem: {
      title: 'הבעיה',
      description: 'פדרציית Matrix מסתמכת על מפתחות שרת עם נראות מוגבלת. קשה לעקוב אחר רוטציית מפתחות, קשה לזהות שרתים שנפרצו, ואין מסלול ביקורת לשינויי מפתחות. האמון הוא משתמע.',
    },
    
    solution: {
      title: 'הפתרון',
      description: 'MXKeys מספק אימות מפתחות עם חתימות פרספקטיבה, יומן שקיפות משורשר בגיבוב עם הוכחות Merkle, זיהוי חריגות, מדיניות אמון הניתנת להגדרה ומצבי אשכול מאומתים.',
    },
  },

  features: {
    title: 'תכונות',
    description: 'יכולות אימות מפתחות עבור פדרציית Matrix.',
    
    caching: {
      title: 'מטמון מפתחות',
      description: 'מאחסן מפתחות מאומתים ב-PostgreSQL. מפחית זמן תגובה ועומס על שרתי מקור.',
    },
    verification: {
      title: 'אימות חתימה',
      description: 'מאמת את כל המפתחות שנאספו מול חתימות השרת לפני השמירה במטמון.',
    },
    perspective: {
      title: 'חתימת פרספקטיבה',
      description: 'מוסיף חתימה משותפת של נוטריון (ed25519:mxkeys) למפתחות מאומתים — אישור עצמאי.',
    },
    discovery: {
      title: 'גילוי שרת',
      description: 'תמיכה בגילוי Matrix עבור הפניית .well-known, רשומות SRV (_matrix-fed._tcp), כתובות IP ישירות ונפילה לפורט בטווח נוטריון המפתחות של MXKeys.',
    },
    fallback: {
      title: 'תמיכת נפילה',
      description: 'אם האחזור הישיר נכשל, MXKeys יכול לשאול נוטריונים חלופיים מוגדרים כנתיב אמון תפעולי מפורש.',
    },
    performance: {
      title: 'ביצועים גבוהים',
      description: 'כתוב ב-Go. מטמון זיכרון, מאגר חיבורים, ניקוי יעיל ופריסת קובץ בינארי יחיד.',
    },
    opensource: {
      title: 'קוד פתוח',
      description: 'קוד הניתן לביקורת. ללא לוגיקה נסתרת, ללא תלויות קנייניות.',
    },
  },

  howItWorks: {
    title: 'איך זה עובד',
    description: 'תהליך אימות המפתחות.',
    
    steps: {
      request: {
        title: '1. בקשה',
        description: 'שרת Matrix שואל את MXKeys על מפתחות שרת אחר באמצעות POST /_matrix/key/v2/query',
      },
      cache: {
        title: '2. בדיקת מטמון',
        description: 'MXKeys בודק את מטמון הזיכרון, ואז PostgreSQL. אם קיים מפתח תקף במטמון — מחזיר מיידית.',
      },
      fetch: {
        title: '3. גילוי שרת',
        description: 'בהחמצת מטמון, MXKeys מפענח את שרת היעד באמצעות הפניית .well-known, רשומות SRV ונפילה לפורט — ואז מאחזר מפתחות באמצעות /_matrix/key/v2/server',
      },
      verify: {
        title: '4. אימות',
        description: 'MXKeys מאמת את החתימה העצמית של השרת באמצעות Ed25519. חתימות לא תקפות נדחות.',
      },
      sign: {
        title: '5. חתימה משותפת',
        description: 'MXKeys מוסיף את חתימת הפרספקטיבה שלו (ed25519:mxkeys) — מעיד שאימת את המפתחות.',
      },
      respond: {
        title: '6. תגובה',
        description: 'מפתחות עם חתימות מקוריות וחתימות נוטריון מוחזרים לשרת המבקש.',
      },
    },
  },

  api: {
    title: 'נקודות קצה API',
    description: 'MXKeys מיישם את Matrix Key Server API ובדיקות תפעוליות.',
    
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'מחזיר את המפתחות הציבוריים של MXKeys. משמש לאימות חתימות.',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'מחזיר מפתח MXKeys ספציפי לפי מזהה מפתח. מגיב עם M_NOT_FOUND כאשר המפתח חסר.',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'נקודת הקצה הראשית של הנוטריון. שואל מפתחות עבור שרתי Matrix ומחזיר מפתחות מאומתים עם חתימה משותפת של MXKeys.',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'מידע גרסת שרת.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'נקודת קצה בריאות. מחזירה מטא-נתוני בריאות השירות.',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'נקודת קצה מוכנות. מאמתת קישוריות DB ומפתח חתימה פעיל.',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'נקודת קצה בדיקת חיים. מחזירה מצב חיי התהליך.',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'מצב שירות מפורט: זמן פעילות, מדדי מטמון, סטטיסטיקות מסד נתונים ומצב תכונות ארגוניות אופציונלי.',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'חשיפת מדדי Prometheus עבור טלמטריית שירות וזמן ריצה.',
    },
    errorsTitle: 'מודל שגיאות',
    errorsDescription: 'אימות בקשות ובקרות שימוש לרעה משתמשים בקודי שגיאה תואמי Matrix: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE ו-M_LIMIT_EXCEEDED.',
    protectedTitle: 'נתיבים תפעוליים מוגנים',
    protectedDescription: 'נתיבי שקיפות, אנליטיקה, אשכול ומדיניות דורשים טוקן גישה ארגוני ומתועדים בנפרד מ-API הפדרציה הציבורי היציב.',
  },

  integration: {
    title: 'אינטגרציה',
    description: 'הגדר את שרת ה-Matrix שלך להשתמש ב-MXKeys כשרת מפתחות מהימן.',
    
    synapse: 'הגדרת Synapse',
    mxcore: 'הגדרת MXCore',
  },

  ecosystem: {
    title: 'חלק מ-Matrix Family',
    description: 'MXKeys פותח על ידי Matrix Family Inc. זמין לכל שרתי Matrix.',
    
    matrixFamily: {
      title: 'Matrix Family',
      description: 'מרכז האקוסיסטם',
    },
    hushme: {
      title: 'HushMe',
      description: 'לקוח Matrix',
    },
    hushmeStore: {
      title: 'HushMe Store',
      description: 'אפליקציות MFOS',
    },
    mxcore: {
      title: 'MXCore',
      description: 'שרת Matrix ביתי',
    },
    mfos: {
      title: 'MFOS',
      description: 'פלטפורמת מפתחים',
    },
  },

  footer: {
    ecosystem: 'אקוסיסטם',
    resources: 'משאבים',
    contact: 'יצירת קשר',
    
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    
    architecture: 'ארכיטקטורה',
    apiReference: 'מסמכי API',
    
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    
    protocol: 'פרוטוקול',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',

    copyrightPrefix: '© 2026 Matrix Family Inc. כל הזכויות שמורות. חלק מאקוסיסטם ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: '.',
    tagline: 'נוטריון מפתחות עבור פדרציית Matrix.',
  },
};
