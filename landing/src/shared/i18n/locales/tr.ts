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
    home: 'Ana Sayfa',
    about: 'Hakkında',
    howItWorks: 'Nasıl Çalışır',
    api: 'API',
    ecosystem: 'Ekosistem',
    homeAria: 'MXKeys ana sayfa',
    github: 'MXKeys GitHub deposu',
    language: 'Dil',
    openMenu: 'Gezinme menüsünü aç',
    closeMenu: 'Gezinme menüsünü kapat',
  },

  hero: {
    title: 'MXKeys',
    subtitle: 'Federasyon Güven Altyapısı',
    tagline: 'Güven. Doğrula. Federasyon.',
    description: 'Matrix için federasyon anahtar güven katmanı: anahtar doğrulama, şeffaflık günlüğü, anomali tespiti ve kimliği doğrulanmış küme koordinasyonu.',
    trust: 'PostgreSQL önbellekleme, Matrix-spec keşfi ve operasyonel uç noktalar ile Go servisi.',
    learnMore: 'Daha Fazla Bilgi',
    viewAPI: 'API Görüntüle',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },

  status: {
    online: 'Altyapı Çevrimiçi',
  },

  about: {
    title: 'MXKeys Nedir?',
    description: 'MXKeys, Matrix sunucularının kimlikleri doğrulamasına, anahtar değişikliklerini izlemesine, anomalileri tespit etmesine ve güven politikalarını uygulamasına yardımcı olan bir Matrix Federasyon Güven Altyapısıdır.',
    
    problem: {
      title: 'Sorun',
      description: 'Matrix federasyonu sınırlı görünürlüğe sahip sunucu anahtarlarına dayanır. Anahtar rotasyonunu izlemek zordur, ele geçirilmiş sunucuları tespit etmek güçtür ve anahtar değişiklikleri için denetim izi yoktur. Güven örtüktür.',
    },
    
    solution: {
      title: 'Çözüm',
      description: 'MXKeys, perspektif imzalarıyla anahtar doğrulama, Merkle kanıtlarıyla karma zincirleme şeffaflık günlüğü, anomali tespiti, yapılandırılabilir güven politikaları ve kimliği doğrulanmış küme modları sağlar.',
    },
  },

  features: {
    title: 'Özellikler',
    description: 'Matrix federasyonu için anahtar doğrulama yetenekleri.',
    
    caching: {
      title: 'Anahtar Önbellekleme',
      description: 'Doğrulanmış anahtarları PostgreSQL\'de depolar. Kaynak sunucularda gecikmeyi ve yükü azaltır.',
    },
    verification: {
      title: 'İmza Doğrulama',
      description: 'Önbelleklemeden önce tüm alınan anahtarları sunucu imzalarına karşı doğrular.',
    },
    perspective: {
      title: 'Perspektif İmzalama',
      description: 'Doğrulanmış anahtarlara bir noter ortak imzası (ed25519:mxkeys) ekler — bağımsız bir onay.',
    },
    discovery: {
      title: 'Sunucu Keşfi',
      description: 'MXKeys anahtar-noter kapsamında .well-known delegasyonu, SRV kayıtları (_matrix-fed._tcp), IP değişmezleri ve port geri dönüşü için Matrix keşif desteği.',
    },
    fallback: {
      title: 'Geri Dönüş Desteği',
      description: 'Doğrudan getirme başarısız olursa, MXKeys açık bir operasyonel güven yolu olarak yapılandırılmış yedek noterlerden sorgulama yapabilir.',
    },
    performance: {
      title: 'Yüksek Performans',
      description: 'Go ile yazılmıştır. Bellek önbellekleme, bağlantı havuzu, verimli temizlik ve tek ikili dağıtım.',
    },
    opensource: {
      title: 'Açık Kaynak',
      description: 'Denetlenebilir kod. Gizli mantık yok, tescilli bağımlılık yok.',
    },
  },

  howItWorks: {
    title: 'Nasıl Çalışır',
    description: 'Anahtar doğrulama akışı.',
    
    steps: {
      request: {
        title: '1. İstek',
        description: 'Bir Matrix sunucusu, POST /_matrix/key/v2/query aracılığıyla başka bir sunucunun anahtarları için MXKeys\'i sorgular',
      },
      cache: {
        title: '2. Önbellek Kontrolü',
        description: 'MXKeys bellek önbelleğini, ardından PostgreSQL\'i kontrol eder. Geçerli önbelleklenmiş anahtar varsa — hemen döndürür.',
      },
      fetch: {
        title: '3. Sunucu Keşfi',
        description: 'Önbellek ıskasında, MXKeys hedef sunucuyu .well-known delegasyonu, SRV kayıtları ve port geri dönüşü kullanarak çözümler — ardından /_matrix/key/v2/server aracılığıyla anahtarları getirir',
      },
      verify: {
        title: '4. Doğrulama',
        description: 'MXKeys, sunucunun öz imzasını Ed25519 kullanarak doğrular. Geçersiz imzalar reddedilir.',
      },
      sign: {
        title: '5. Ortak İmza',
        description: 'MXKeys perspektif imzasını (ed25519:mxkeys) ekler — anahtarları doğruladığını onaylar.',
      },
      respond: {
        title: '6. Yanıt',
        description: 'Hem orijinal hem de noter imzalarına sahip anahtarlar talep eden sunucuya döndürülür.',
      },
    },
  },

  api: {
    title: 'API Uç Noktaları',
    description: 'MXKeys, Matrix Key Server API ve operasyonel probları uygular.',
    
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'MXKeys genel anahtarlarını döndürür. İmzaları doğrulamak için kullanılır.',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'Anahtar kimliğine göre belirli bir MXKeys anahtarını döndürür. Anahtar yoksa M_NOT_FOUND ile yanıt verir.',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'Ana noter uç noktası. Matrix sunucuları için anahtarları sorgular ve MXKeys ortak imzasıyla doğrulanmış anahtarları döndürür.',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'Sunucu sürüm bilgisi.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'Sağlık uç noktası. Servis sağlık meta verilerini döndürür.',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'Hazırlık uç noktası. DB bağlantısını ve aktif imzalama anahtarını doğrular.',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'Canlılık prob uç noktası. Süreç canlı durumunu döndürür.',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'Ayrıntılı servis durumu: çalışma süresi, önbellek metrikleri, veritabanı istatistikleri ve isteğe bağlı kurumsal özellik durumu.',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'Servis ve çalışma zamanı telemetrisi için Prometheus metrik sunumu.',
    },
    errorsTitle: 'Hata modeli',
    errorsDescription: 'İstek doğrulama ve kötüye kullanım kontrolleri Matrix uyumlu hata kodlarını kullanır: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE ve M_LIMIT_EXCEEDED.',
    protectedTitle: 'Korumalı operasyonel rotalar',
    protectedDescription: 'Şeffaflık, analitik, küme ve politika rotaları kurumsal erişim jetonu gerektirir ve kararlı genel federasyon API\'sinden ayrı olarak belgelenir.',
  },

  integration: {
    title: 'Entegrasyon',
    description: 'Matrix sunucunuzu MXKeys\'i güvenilir anahtar sunucusu olarak kullanacak şekilde yapılandırın.',
    
    synapse: 'Synapse Yapılandırması',
    mxcore: 'MXCore Yapılandırması',
  },

  ecosystem: {
    title: 'Matrix Family\'nin Parçası',
    description: 'MXKeys, Matrix Family Inc. tarafından geliştirilmiştir. Tüm Matrix sunucuları için kullanılabilir.',
    
    matrixFamily: {
      title: 'Matrix Family',
      description: 'Ekosistem Merkezi',
    },
    hushme: {
      title: 'HushMe',
      description: 'Matrix İstemcisi',
    },
    hushmeStore: {
      title: 'HushMe Store',
      description: 'MFOS Uygulamaları',
    },
    mxcore: {
      title: 'MXCore',
      description: 'Matrix Ev Sunucusu',
    },
    mfos: {
      title: 'MFOS',
      description: 'Geliştirici Platformu',
    },
  },

  footer: {
    ecosystem: 'Ekosistem',
    resources: 'Kaynaklar',
    contact: 'İletişim',
    
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    
    architecture: 'Mimari',
    apiReference: 'API Referansı',
    
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    
    protocol: 'Protokol',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',

    copyrightPrefix: '© 2026 Matrix Family Inc. Tüm hakları saklıdır. ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: ' ekosisteminin parçası.',
    tagline: 'Matrix federasyonu için Key Notary.',
  },
};
