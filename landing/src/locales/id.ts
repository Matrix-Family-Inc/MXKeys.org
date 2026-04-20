/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Sun Apr 13 2026 UTC
 * Status: Created
 */

export const id = {
  nav: {
    home: 'Beranda',
    about: 'Tentang',
    howItWorks: 'Cara Kerja',
    api: 'API',
    ecosystem: 'Ekosistem',
    homeAria: 'Beranda MXKeys',
    github: 'Repositori GitHub MXKeys',
    language: 'Bahasa',
    openMenu: 'Buka menu navigasi',
    closeMenu: 'Tutup menu navigasi',
  },

  hero: {
    title: 'MXKeys',
    subtitle: 'Infrastruktur Kepercayaan Federasi',
    tagline: 'Percaya. Verifikasi. Federasi.',
    description: 'Lapisan kepercayaan kunci federasi untuk Matrix: verifikasi kunci, pencatatan transparansi, deteksi anomali, dan koordinasi klaster terotentikasi.',
    trust: 'Layanan Go dengan caching PostgreSQL, penemuan Matrix-spec, dan endpoint operasional.',
    learnMore: 'Pelajari Lebih Lanjut',
    viewAPI: 'Lihat API',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },

  status: {
    online: 'Infrastruktur Online',
  },

  about: {
    title: 'Apa itu MXKeys?',
    description: 'MXKeys adalah Infrastruktur Kepercayaan Federasi Matrix yang membantu server Matrix memverifikasi identitas, melacak perubahan kunci, mendeteksi anomali, dan menerapkan kebijakan kepercayaan.',
    
    problem: {
      title: 'Masalah',
      description: 'Federasi Matrix bergantung pada kunci server dengan visibilitas terbatas. Rotasi kunci sulit dilacak, server yang disusupi sulit dideteksi, dan tidak ada jejak audit untuk perubahan kunci. Kepercayaan bersifat implisit.',
    },
    
    solution: {
      title: 'Solusi',
      description: 'MXKeys menyediakan verifikasi kunci dengan tanda tangan perspektif, log transparansi berantai hash dengan bukti Merkle, deteksi anomali, kebijakan kepercayaan yang dapat dikonfigurasi, dan mode klaster terotentikasi.',
    },
  },

  features: {
    title: 'Fitur',
    description: 'Kemampuan verifikasi kunci untuk federasi Matrix.',
    
    caching: {
      title: 'Caching Kunci',
      description: 'Menyimpan kunci terverifikasi di PostgreSQL. Mengurangi latensi dan beban pada server asal.',
    },
    verification: {
      title: 'Verifikasi Tanda Tangan',
      description: 'Memvalidasi semua kunci yang diambil terhadap tanda tangan server sebelum caching.',
    },
    perspective: {
      title: 'Penandatanganan Perspektif',
      description: 'Menambahkan tanda tangan bersama notaris (ed25519:mxkeys) pada kunci terverifikasi — pengesahan independen.',
    },
    discovery: {
      title: 'Penemuan Server',
      description: 'Dukungan penemuan Matrix untuk delegasi .well-known, catatan SRV (_matrix-fed._tcp), literal IP, dan fallback port dalam lingkup notaris kunci MXKeys.',
    },
    fallback: {
      title: 'Dukungan Fallback',
      description: 'Jika pengambilan langsung gagal, MXKeys dapat mengirim kueri ke notaris cadangan yang dikonfigurasi sebagai jalur kepercayaan operasional eksplisit.',
    },
    performance: {
      title: 'Performa Tinggi',
      description: 'Ditulis dalam Go. Caching memori, connection pooling, pembersihan efisien, dan deployment biner tunggal.',
    },
    opensource: {
      title: 'Sumber Terbuka',
      description: 'Kode yang dapat diaudit. Tanpa logika tersembunyi, tanpa dependensi proprietary.',
    },
  },

  howItWorks: {
    title: 'Cara Kerja',
    description: 'Alur verifikasi kunci.',
    
    steps: {
      request: {
        title: '1. Permintaan',
        description: 'Server Matrix mengirim kueri ke MXKeys untuk kunci server lain melalui POST /_matrix/key/v2/query',
      },
      cache: {
        title: '2. Pemeriksaan Cache',
        description: 'MXKeys memeriksa cache memori, lalu PostgreSQL. Jika kunci cache yang valid ada — langsung dikembalikan.',
      },
      fetch: {
        title: '3. Penemuan Server',
        description: 'Pada cache miss, MXKeys menyelesaikan server target menggunakan delegasi .well-known, catatan SRV, dan fallback port — lalu mengambil kunci melalui /_matrix/key/v2/server',
      },
      verify: {
        title: '4. Verifikasi',
        description: 'MXKeys memverifikasi tanda tangan mandiri server menggunakan Ed25519. Tanda tangan tidak valid ditolak.',
      },
      sign: {
        title: '5. Tanda Tangan Bersama',
        description: 'MXKeys menambahkan tanda tangan perspektifnya (ed25519:mxkeys) — membuktikan bahwa kunci telah diverifikasi.',
      },
      respond: {
        title: '6. Respons',
        description: 'Kunci dengan tanda tangan asli dan notaris dikembalikan ke server yang meminta.',
      },
    },
  },

  api: {
    title: 'Endpoint API',
    description: 'MXKeys mengimplementasikan Matrix Key Server API dan probe operasional.',
    
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'Mengembalikan kunci publik MXKeys. Digunakan untuk memverifikasi tanda tangan.',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'Mengembalikan kunci MXKeys tertentu berdasarkan ID kunci. Merespons dengan M_NOT_FOUND saat kunci tidak ada.',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'Endpoint notaris utama. Mengirim kueri kunci untuk server Matrix dan mengembalikan kunci terverifikasi dengan tanda tangan bersama MXKeys.',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'Informasi versi server.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'Endpoint kesehatan. Mengembalikan metadata kesehatan layanan.',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'Endpoint kesiapan. Memverifikasi konektivitas DB dan kunci penandatanganan aktif.',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'Endpoint probe keaktifan. Mengembalikan status proses hidup.',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'Status layanan terperinci: uptime, metrik cache, statistik database, dan status fitur enterprise opsional.',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'Eksposisi metrik Prometheus untuk telemetri layanan dan runtime.',
    },
    errorsTitle: 'Model error',
    errorsDescription: 'Validasi permintaan dan kontrol penyalahgunaan menggunakan kode error kompatibel Matrix: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE, dan M_LIMIT_EXCEEDED.',
    protectedTitle: 'Rute operasional terlindungi',
    protectedDescription: 'Rute transparansi, analitik, klaster, dan kebijakan memerlukan token akses enterprise dan didokumentasikan secara terpisah dari API federasi publik yang stabil.',
  },

  integration: {
    title: 'Integrasi',
    description: 'Konfigurasikan server Matrix Anda untuk menggunakan MXKeys sebagai server kunci tepercaya.',
    
    synapse: 'Konfigurasi Synapse',
    mxcore: 'Konfigurasi MXCore',
  },

  ecosystem: {
    title: 'Bagian dari Matrix Family',
    description: 'MXKeys dikembangkan oleh Matrix Family Inc. Tersedia untuk semua server Matrix.',
    
    matrixFamily: {
      title: 'Matrix Family',
      description: 'Pusat Ekosistem',
    },
    hushme: {
      title: 'HushMe',
      description: 'Klien Matrix',
    },
    hushmeStore: {
      title: 'HushMe Store',
      description: 'Aplikasi MFOS',
    },
    mxcore: {
      title: 'MXCore',
      description: 'Homeserver Matrix',
    },
    mfos: {
      title: 'MFOS',
      description: 'Platform Pengembang',
    },
  },

  footer: {
    ecosystem: 'Ekosistem',
    resources: 'Sumber Daya',
    contact: 'Kontak',
    
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    
    architecture: 'Arsitektur',
    apiReference: 'Referensi API',
    
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    
    protocol: 'Protokol',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',

    copyrightPrefix: '© 2026 Matrix Family Inc. Hak cipta dilindungi. Bagian dari ekosistem ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: '.',
    tagline: 'Key Notary untuk federasi Matrix.',
  },
};
