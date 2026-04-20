/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Sun Apr 13 2026 UTC
 * Status: Created
 */

export const it = {
  nav: {
    home: 'Home',
    about: 'Informazioni',
    howItWorks: 'Come funziona',
    api: 'API',
    ecosystem: 'Ecosistema',
    homeAria: 'Home di MXKeys',
    github: 'Repository GitHub di MXKeys',
    language: 'Lingua',
    openMenu: 'Apri menu di navigazione',
    closeMenu: 'Chiudi menu di navigazione',
  },

  hero: {
    title: 'MXKeys',
    subtitle: 'Infrastruttura di fiducia della federazione',
    tagline: 'Fiducia. Verifica. Federazione.',
    description: 'Livello di fiducia delle chiavi di federazione per Matrix: verifica delle chiavi, registrazione della trasparenza, rilevamento delle anomalie e coordinamento dei cluster autenticato.',
    trust: 'Servizio Go con caching PostgreSQL, discovery Matrix-spec e endpoint operativi.',
    learnMore: 'Scopri di più',
    viewAPI: 'Vedi API',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },

  status: {
    online: 'Infrastruttura online',
  },

  about: {
    title: 'Cos\'è MXKeys?',
    description: 'MXKeys è un\'infrastruttura di fiducia della federazione Matrix che aiuta i server Matrix a verificare le identità, tracciare le modifiche delle chiavi, rilevare anomalie e applicare le politiche di fiducia.',
    
    problem: {
      title: 'Il problema',
      description: 'La federazione Matrix si basa su chiavi del server con visibilità limitata. La rotazione delle chiavi è difficile da tracciare, i server compromessi sono difficili da rilevare e non esiste una traccia di audit per le modifiche delle chiavi. La fiducia è implicita.',
    },
    
    solution: {
      title: 'La soluzione',
      description: 'MXKeys fornisce verifica delle chiavi con firme prospettiche, un registro di trasparenza a catena hash con prove Merkle, rilevamento delle anomalie, politiche di fiducia configurabili e modalità cluster autenticate.',
    },
  },

  features: {
    title: 'Funzionalità',
    description: 'Capacità di verifica delle chiavi per la federazione Matrix.',
    
    caching: {
      title: 'Caching delle chiavi',
      description: 'Memorizza le chiavi verificate in PostgreSQL. Riduce la latenza e il carico sui server di origine.',
    },
    verification: {
      title: 'Verifica delle firme',
      description: 'Valida tutte le chiavi recuperate rispetto alle firme del server prima del caching.',
    },
    perspective: {
      title: 'Firma prospettica',
      description: 'Aggiunge una co-firma notarile (ed25519:mxkeys) alle chiavi verificate — un\'attestazione indipendente.',
    },
    discovery: {
      title: 'Discovery del server',
      description: 'Supporto discovery Matrix per delega .well-known, record SRV (_matrix-fed._tcp), letterali IP e fallback della porta nell\'ambito del notaio delle chiavi MXKeys.',
    },
    fallback: {
      title: 'Supporto fallback',
      description: 'Se il recupero diretto fallisce, MXKeys può interrogare notai di fallback configurati come percorso di fiducia operativo esplicito.',
    },
    performance: {
      title: 'Alte prestazioni',
      description: 'Scritto in Go. Caching in memoria, connection pooling, pulizia efficiente e deployment a binario singolo.',
    },
    opensource: {
      title: 'Open source',
      description: 'Codice verificabile. Nessuna logica nascosta, nessuna dipendenza proprietaria.',
    },
  },

  howItWorks: {
    title: 'Come funziona',
    description: 'Il flusso di verifica delle chiavi.',
    
    steps: {
      request: {
        title: '1. Richiesta',
        description: 'Un server Matrix interroga MXKeys per le chiavi di un altro server tramite POST /_matrix/key/v2/query',
      },
      cache: {
        title: '2. Controllo cache',
        description: 'MXKeys controlla la cache in memoria, poi PostgreSQL. Se esiste una chiave cache valida — restituisce immediatamente.',
      },
      fetch: {
        title: '3. Discovery del server',
        description: 'In caso di cache miss, MXKeys risolve il server di destinazione utilizzando la delega .well-known, i record SRV e il fallback della porta — poi recupera le chiavi tramite /_matrix/key/v2/server',
      },
      verify: {
        title: '4. Verifica',
        description: 'MXKeys verifica l\'auto-firma del server utilizzando Ed25519. Le firme non valide vengono respinte.',
      },
      sign: {
        title: '5. Co-firma',
        description: 'MXKeys aggiunge la sua firma prospettica (ed25519:mxkeys) — attestando di aver verificato le chiavi.',
      },
      respond: {
        title: '6. Risposta',
        description: 'Le chiavi con entrambe le firme originali e notarili vengono restituite al server richiedente.',
      },
    },
  },

  api: {
    title: 'Endpoint API',
    description: 'MXKeys implementa l\'API Matrix Key Server e le sonde operative.',
    
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'Restituisce le chiavi pubbliche di MXKeys. Utilizzato per verificare le firme.',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'Restituisce una chiave MXKeys specifica per ID chiave. Risponde con M_NOT_FOUND quando la chiave è assente.',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'Endpoint notarile principale. Interroga le chiavi per i server Matrix e restituisce chiavi verificate con co-firma MXKeys.',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'Informazioni sulla versione del server.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'Endpoint di salute. Restituisce i metadati di salute del servizio.',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'Endpoint di prontezza. Verifica la connettività DB e la chiave di firma attiva.',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'Endpoint sonda di vitalità. Restituisce lo stato del processo attivo.',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'Stato dettagliato del servizio: uptime, metriche cache, statistiche del database e stato opzionale delle funzionalità enterprise.',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'Esposizione metriche Prometheus per la telemetria del servizio e del runtime.',
    },
    errorsTitle: 'Modello di errore',
    errorsDescription: 'La validazione delle richieste e i controlli anti-abuso utilizzano codici di errore compatibili con Matrix: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE e M_LIMIT_EXCEEDED.',
    protectedTitle: 'Route operative protette',
    protectedDescription: 'Le route di trasparenza, analisi, cluster e politiche richiedono un token di accesso enterprise e sono documentate separatamente dall\'API di federazione pubblica stabile.',
  },

  integration: {
    title: 'Integrazione',
    description: 'Configura il tuo server Matrix per utilizzare MXKeys come server di chiavi attendibile.',
    
    synapse: 'Configurazione Synapse',
    mxcore: 'Configurazione MXCore',
  },

  ecosystem: {
    title: 'Parte di Matrix Family',
    description: 'MXKeys è sviluppato da Matrix Family Inc. Disponibile per tutti i server Matrix.',
    
    matrixFamily: {
      title: 'Matrix Family',
      description: 'Hub dell\'ecosistema',
    },
    hushme: {
      title: 'HushMe',
      description: 'Client Matrix',
    },
    hushmeStore: {
      title: 'HushMe Store',
      description: 'App MFOS',
    },
    mxcore: {
      title: 'MXCore',
      description: 'Homeserver Matrix',
    },
    mfos: {
      title: 'MFOS',
      description: 'Piattaforma sviluppatori',
    },
  },

  footer: {
    ecosystem: 'Ecosistema',
    resources: 'Risorse',
    contact: 'Contatti',
    
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    
    architecture: 'Architettura',
    apiReference: 'Riferimento API',
    
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    
    protocol: 'Protocollo',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',

    copyrightPrefix: '© 2026 Matrix Family Inc. Tutti i diritti riservati. Parte dell\'ecosistema ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: '.',
    tagline: 'Key Notary per la federazione Matrix.',
  },
};
