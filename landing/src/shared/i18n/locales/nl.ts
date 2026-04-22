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
    home: 'Home',
    about: 'Over',
    howItWorks: 'Hoe het werkt',
    api: 'API',
    ecosystem: 'Ecosysteem',
    homeAria: 'MXKeys home',
    github: 'MXKeys GitHub-repository',
    language: 'Taal',
    openMenu: 'Navigatiemenu openen',
    closeMenu: 'Navigatiemenu sluiten',
  },

  hero: {
    title: 'MXKeys',
    subtitle: 'Federatie-vertrouwensinfrastructuur',
    tagline: 'Vertrouwen. Verifiëren. Federeren.',
    description: 'Federatiesleutel-vertrouwenslaag voor Matrix: sleutelverificatie, transparantielogging, anomaliedetectie en geauthenticeerde clustercoördinatie.',
    trust: 'Go-service met PostgreSQL-caching, Matrix-spec-discovery en operationele endpoints.',
    learnMore: 'Meer informatie',
    viewAPI: 'API bekijken',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },

  status: {
    online: 'Infrastructuur online',
  },

  about: {
    title: 'Wat is MXKeys?',
    description: 'MXKeys is een Matrix Federatie-vertrouwensinfrastructuur die Matrix-servers helpt identiteiten te verifiëren, sleutelwijzigingen bij te houden, anomalieën te detecteren en vertrouwensbeleid af te dwingen.',
    
    problem: {
      title: 'Het probleem',
      description: 'Matrix-federatie is afhankelijk van serversleutels met beperkte zichtbaarheid. Sleutelrotatie is moeilijk te volgen, gecompromitteerde servers zijn moeilijk te detecteren en er is geen audittrail voor sleutelwijzigingen. Vertrouwen is impliciet.',
    },
    
    solution: {
      title: 'De oplossing',
      description: 'MXKeys biedt sleutelverificatie met perspectiefhandtekeningen, een hash-geketend transparantielog met Merkle-bewijzen, anomaliedetectie, configureerbaar vertrouwensbeleid en geauthenticeerde clustermodi.',
    },
  },

  features: {
    title: 'Functies',
    description: 'Sleutelverificatiemogelijkheden voor Matrix-federatie.',
    
    caching: {
      title: 'Sleutelcaching',
      description: 'Slaat geverifieerde sleutels op in PostgreSQL. Vermindert latentie en belasting van bronservers.',
    },
    verification: {
      title: 'Handtekeningverificatie',
      description: 'Valideert alle opgehaalde sleutels tegen serverhandtekeningen vóór caching.',
    },
    perspective: {
      title: 'Perspectiefondertekening',
      description: 'Voegt een notariële medeondertekening (ed25519:mxkeys) toe aan geverifieerde sleutels — een onafhankelijke attestatie.',
    },
    discovery: {
      title: 'Serverdiscovery',
      description: 'Matrix-discovery-ondersteuning voor .well-known-delegatie, SRV-records (_matrix-fed._tcp), IP-literals en poortfallback binnen het MXKeys-sleutelnotarisbereik.',
    },
    fallback: {
      title: 'Fallback-ondersteuning',
      description: 'Als directe ophaling mislukt, kan MXKeys geconfigureerde fallback-notarissen bevragen als een expliciet operationeel vertrouwenspad.',
    },
    performance: {
      title: 'Hoge prestaties',
      description: 'Geschreven in Go. Geheugencaching, connection pooling, efficiënte opschoning en single-binary deployment.',
    },
    opensource: {
      title: 'Open source',
      description: 'Auditeerbare code. Geen verborgen logica, geen propriëtaire afhankelijkheden.',
    },
  },

  howItWorks: {
    title: 'Hoe het werkt',
    description: 'De sleutelverificatiestroom.',
    
    steps: {
      request: {
        title: '1. Verzoek',
        description: 'Een Matrix-server bevraagt MXKeys voor de sleutels van een andere server via POST /_matrix/key/v2/query',
      },
      cache: {
        title: '2. Cachecontrole',
        description: 'MXKeys controleert de geheugencache, vervolgens PostgreSQL. Als een geldige gecachte sleutel bestaat — retourneert onmiddellijk.',
      },
      fetch: {
        title: '3. Serverdiscovery',
        description: 'Bij een cachemiss lost MXKeys de doelserver op via .well-known-delegatie, SRV-records en poortfallback — haalt vervolgens sleutels op via /_matrix/key/v2/server',
      },
      verify: {
        title: '4. Verificatie',
        description: 'MXKeys verifieert de zelfhandtekening van de server met Ed25519. Ongeldige handtekeningen worden afgewezen.',
      },
      sign: {
        title: '5. Medeondertekening',
        description: 'MXKeys voegt zijn perspectiefhandtekening (ed25519:mxkeys) toe — bevestigend dat het de sleutels heeft geverifieerd.',
      },
      respond: {
        title: '6. Antwoord',
        description: 'Sleutels met zowel originele als notariële handtekeningen worden teruggestuurd naar de verzoekende server.',
      },
    },
  },

  api: {
    title: 'API-endpoints',
    description: 'MXKeys implementeert de Matrix Key Server API en operationele probes.',
    
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'Retourneert de openbare sleutels van MXKeys. Gebruikt om handtekeningen te verifiëren.',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'Retourneert een specifieke MXKeys-sleutel op sleutel-ID. Antwoordt met M_NOT_FOUND wanneer de sleutel afwezig is.',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'Hoofd notaris-endpoint. Bevraagt sleutels voor Matrix-servers en retourneert geverifieerde sleutels met MXKeys-medeondertekening.',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'Serverversie-informatie.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'Gezondheidsendpoint. Retourneert metadata over de servicegezondheid.',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'Gereedheidsendpoint. Verifieert DB-connectiviteit en actieve ondertekeningssleutel.',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'Levendigheidsprobeendpoint. Retourneert de processtatus.',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'Gedetailleerde servicestatus: uptime, cachemetrics, databasestatistieken en status van optionele subsystemen.',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'Prometheus-metrieksexpositie voor service- en runtime-telemetrie.',
    },
    errorsTitle: 'Foutmodel',
    errorsDescription: 'Verzoekvalidatie en misbruikcontroles gebruiken Matrix-compatibele foutcodes: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE en M_LIMIT_EXCEEDED.',
    protectedTitle: 'Admin-only operationele routes',
    protectedDescription: 'Transparantie-, analyse-, cluster- en beleidsroutes zijn admin-only ops/debug-oppervlakken. Ze worden beschermd door een bearer-token (security.admin_access_token) en staan buiten de stabiele openbare federatie-API.',
  },

  integration: {
    title: 'Integratie',
    description: 'Configureer uw Matrix-server om MXKeys als vertrouwde sleutelserver te gebruiken.',
    
    synapse: 'Synapse-configuratie',
    mxcore: 'MXCore-configuratie',
  },

  ecosystem: {
    title: 'Onderdeel van Matrix Family',
    description: 'MXKeys is ontwikkeld door Matrix Family Inc. Beschikbaar voor alle Matrix-servers.',
    
    matrixFamily: {
      title: 'Matrix Family',
      description: 'Ecosysteemhub',
    },
    hushme: {
      title: 'HushMe',
      description: 'Matrix-client',
    },
    hushmeStore: {
      title: 'HushMe Store',
      description: 'MFOS-apps',
    },
    mxcore: {
      title: 'MXCore',
      description: 'Matrix-homeserver',
    },
    mfos: {
      title: 'MFOS',
      description: 'Ontwikkelaarsplatform',
    },
  },

  footer: {
    ecosystem: 'Ecosysteem',
    resources: 'Bronnen',
    contact: 'Contact',
    
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    
    architecture: 'Architectuur',
    apiReference: 'API-referentie',
    
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    
    protocol: 'Protocol',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',

    copyrightPrefix: '© 2026 Matrix Family Inc. Alle rechten voorbehouden. Onderdeel van het ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: '-ecosysteem.',
    tagline: 'Key Notary voor Matrix-federatie.',
  },
};
