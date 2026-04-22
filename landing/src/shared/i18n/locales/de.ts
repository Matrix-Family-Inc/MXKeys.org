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
    home: 'Startseite',
    about: 'Über',
    howItWorks: 'Funktionsweise',
    api: 'API',
    ecosystem: 'Ökosystem',
    homeAria: 'MXKeys Startseite',
    github: 'MXKeys GitHub-Repository',
    language: 'Sprache',
    openMenu: 'Navigationsmenü öffnen',
    closeMenu: 'Navigationsmenü schließen',
  },
  hero: {
    title: 'MXKeys',
    subtitle: 'Föderations-Vertrauensinfrastruktur',
    tagline: 'Vertrauen. Verifizieren. Föderieren.',
    description: 'Föderations-Schlüsselvertrauensschicht für Matrix: Schlüsselverifizierung, Transparenzprotokollierung, Anomalieerkennung und authentifizierte Clusterkoordination.',
    trust: 'Go-Dienst mit PostgreSQL-Caching, Matrix-Spec-Discovery und operativen Endpunkten.',
    learnMore: 'Mehr erfahren',
    viewAPI: 'API anzeigen',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },
  status: {
    online: 'Infrastruktur online',
  },
  about: {
    title: 'Was ist MXKeys?',
    description: 'MXKeys ist eine Matrix-Föderations-Vertrauensinfrastruktur, die Matrix-Servern hilft, Identitäten zu verifizieren, Schlüsseländerungen zu verfolgen, Anomalien zu erkennen und Vertrauensrichtlinien durchzusetzen.',
    problem: {
      title: 'Das Problem',
      description: 'Matrix-Föderation basiert auf Serverschlüsseln mit eingeschränkter Sichtbarkeit. Schlüsselrotation ist schwer nachzuverfolgen, kompromittierte Server sind schwer zu erkennen, und es gibt keinen Prüfpfad für Schlüsseländerungen. Vertrauen ist implizit.',
    },
    solution: {
      title: 'Die Lösung',
      description: 'MXKeys bietet Schlüsselverifizierung mit Perspektivsignaturen, ein hash-verkettetes Transparenzprotokoll mit Merkle-Beweisen, Anomalieerkennung, konfigurierbare Vertrauensrichtlinien und authentifizierte Clustermodi.',
    },
  },
  features: {
    title: 'Funktionen',
    description: 'Schlüsselverifizierungsfähigkeiten für Matrix-Föderation.',
    caching: {
      title: 'Schlüssel-Caching',
      description: 'Speichert verifizierte Schlüssel in PostgreSQL. Reduziert Latenz und Last auf Ursprungsservern.',
    },
    verification: {
      title: 'Signaturverifizierung',
      description: 'Validiert alle abgerufenen Schlüssel anhand von Serversignaturen vor dem Caching.',
    },
    perspective: {
      title: 'Perspektivsignierung',
      description: 'Fügt eine Notar-Mitsignatur (ed25519:mxkeys) zu verifizierten Schlüsseln hinzu — eine unabhängige Bestätigung.',
    },
    discovery: {
      title: 'Server-Discovery',
      description: 'Matrix-Discovery-Unterstützung für .well-known-Delegation, SRV-Einträge (_matrix-fed._tcp), IP-Literale und Port-Fallback im Rahmen des MXKeys-Schlüsselnotars.',
    },
    fallback: {
      title: 'Fallback-Unterstützung',
      description: 'Wenn der direkte Abruf fehlschlägt, kann MXKeys konfigurierte Fallback-Notare als expliziten operativen Vertrauenspfad abfragen.',
    },
    performance: {
      title: 'Hohe Leistung',
      description: 'Geschrieben in Go. Speicher-Caching, Connection-Pooling, effiziente Bereinigung und Single-Binary-Deployment.',
    },
    opensource: {
      title: 'Open Source',
      description: 'Überprüfbarer Code. Keine versteckte Logik, keine proprietären Abhängigkeiten.',
    },
  },
  howItWorks: {
    title: 'Funktionsweise',
    description: 'Der Schlüsselverifizierungsablauf.',
    steps: {
      request: {
        title: '1. Anfrage',
        description: 'Ein Matrix-Server fragt MXKeys nach den Schlüsseln eines anderen Servers über POST /_matrix/key/v2/query',
      },
      cache: {
        title: '2. Cache-Prüfung',
        description: 'MXKeys prüft den Speicher-Cache, dann PostgreSQL. Wenn ein gültiger gecachter Schlüssel existiert — sofortige Rückgabe.',
      },
      fetch: {
        title: '3. Server-Discovery',
        description: 'Bei Cache-Miss löst MXKeys den Zielserver über .well-known-Delegation, SRV-Einträge und Port-Fallback auf — und ruft dann Schlüssel über /_matrix/key/v2/server ab.',
      },
      verify: {
        title: '4. Verifizierung',
        description: 'MXKeys verifiziert die Selbstsignatur des Servers mit Ed25519. Ungültige Signaturen werden abgelehnt.',
      },
      sign: {
        title: '5. Mitsignierung',
        description: 'MXKeys fügt seine Perspektivsignatur (ed25519:mxkeys) hinzu — als Bestätigung der Schlüsselverifizierung.',
      },
      respond: {
        title: '6. Antwort',
        description: 'Schlüssel mit Original- und Notarsignaturen werden an den anfragenden Server zurückgegeben.',
      },
    },
  },
  api: {
    title: 'API-Endpunkte',
    description: 'MXKeys implementiert die Matrix Key Server API und operative Prüfendpunkte.',
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'Gibt die öffentlichen Schlüssel von MXKeys zurück. Wird zur Signaturverifizierung verwendet.',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'Gibt einen bestimmten MXKeys-Schlüssel nach Schlüssel-ID zurück. Antwortet mit M_NOT_FOUND, wenn der Schlüssel fehlt.',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'Haupt-Notar-Endpunkt. Fragt Schlüssel für Matrix-Server ab und gibt verifizierte Schlüssel mit MXKeys-Mitsignatur zurück.',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'Server-Versionsinformationen.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'Gesundheits-Endpunkt. Gibt Service-Gesundheitsmetadaten zurück.',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'Bereitschafts-Endpunkt. Prüft DB-Konnektivität und aktiven Signaturschlüssel.',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'Liveness-Prüfendpunkt. Gibt den Prozess-Lebenszustand zurück.',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'Detaillierter Servicestatus: Betriebszeit, Cache-Metriken, Datenbankstatistiken und Status der optionalen Subsysteme.',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'Prometheus-Metrikbereitstellung für Service- und Laufzeittelemetrie.',
    },
    errorsTitle: 'Fehlermodell',
    errorsDescription: 'Anfragenvalidierung und Missbrauchskontrollen verwenden Matrix-kompatible Fehlercodes: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE und M_LIMIT_EXCEEDED.',
    protectedTitle: 'Admin-only Operations-Routen',
    protectedDescription: 'Transparenz-, Analyse-, Cluster- und Richtlinienrouten sind admin-only Ops/Debug-Oberflächen. Sie werden durch ein Bearer-Token (security.admin_access_token) abgesichert und liegen außerhalb der stabilen öffentlichen Föderations-API.',
  },
  integration: {
    title: 'Integration',
    description: 'Konfigurieren Sie Ihren Matrix-Server, um MXKeys als vertrauenswürdigen Schlüsselserver zu verwenden.',
    synapse: 'Synapse-Konfiguration',
    mxcore: 'MXCore-Konfiguration',
  },
  ecosystem: {
    title: 'Teil von Matrix Family',
    description: 'MXKeys wird von Matrix Family Inc. entwickelt. Verfügbar für alle Matrix-Server.',
    matrixFamily: { title: 'Matrix Family', description: 'Ökosystem-Hub' },
    hushme: { title: 'HushMe', description: 'Matrix-Client' },
    hushmeStore: { title: 'HushMe Store', description: 'MFOS-Apps' },
    mxcore: { title: 'MXCore', description: 'Matrix-Homeserver' },
    mfos: { title: 'MFOS', description: 'Entwicklerplattform' },
  },
  footer: {
    ecosystem: 'Ökosystem',
    resources: 'Ressourcen',
    contact: 'Kontakt',
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    architecture: 'Architektur',
    apiReference: 'API-Referenz',
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    protocol: 'Protokoll',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',
    copyrightPrefix: '© 2026 Matrix Family Inc. Alle Rechte vorbehalten. Teil des ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: '-Ökosystems.',
    tagline: 'Schlüsselnotar für Matrix-Föderation.',
  },
};
