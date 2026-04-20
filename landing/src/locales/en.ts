/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Tue Jan 27 2026 UTC
 * Status: Created
 */

export const en = {
  nav: {
    home: 'Home',
    about: 'About',
    howItWorks: 'How It Works',
    api: 'API',
    ecosystem: 'Ecosystem',
    homeAria: 'MXKeys home',
    github: 'MXKeys GitHub repository',
    language: 'Language',
    openMenu: 'Open navigation menu',
    closeMenu: 'Close navigation menu',
  },

  hero: {
    title: 'MXKeys',
    subtitle: 'Federation Trust Infrastructure',
    tagline: 'Trust. Verify. Federate.',
    description: 'Federation key trust layer for Matrix: key verification, transparency logging, anomaly detection, and authenticated cluster coordination.',
    trust: 'Go service with PostgreSQL caching, Matrix-spec discovery, and operational endpoints.',
    learnMore: 'Learn More',
    viewAPI: 'View API',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },

  status: {
    online: 'Infrastructure Online',
  },

  about: {
    title: 'What is MXKeys?',
    description: 'MXKeys is a Matrix Federation Trust Infrastructure that helps Matrix servers verify identities, track key changes, detect anomalies, and enforce trust policies.',
    
    problem: {
      title: 'The Problem',
      description: 'Matrix federation relies on server keys with limited visibility. Key rotation is hard to track, compromised servers are hard to detect, and there\'s no audit trail for key changes. Trust is implicit.',
    },
    
    solution: {
      title: 'The Solution',
      description: 'MXKeys provides key verification with perspective signatures, a hash-chained transparency log with Merkle proofs, anomaly detection, configurable trust policies, and authenticated cluster modes.',
    },
  },

  features: {
    title: 'Features',
    description: 'Key verification capabilities for Matrix federation.',
    
    caching: {
      title: 'Key Caching',
      description: 'Stores verified keys in PostgreSQL. Reduces latency and load on origin servers.',
    },
    verification: {
      title: 'Signature Verification',
      description: 'Validates all fetched keys against server signatures before caching.',
    },
    perspective: {
      title: 'Perspective Signing',
      description: 'Adds a notary co-signature (ed25519:mxkeys) to verified keys — an independent attestation.',
    },
    discovery: {
      title: 'Server Discovery',
      description: 'Matrix discovery support for .well-known delegation, SRV records (_matrix-fed._tcp), IP literals, and port fallback within the MXKeys key-notary scope.',
    },
    fallback: {
      title: 'Fallback Support',
      description: 'If direct fetch fails, MXKeys can query configured fallback notaries as an explicit operational trust path.',
    },
    performance: {
      title: 'High Performance',
      description: 'Written in Go. Memory caching, connection pooling, efficient cleanup, and single-binary deployment.',
    },
    opensource: {
      title: 'Open Source',
      description: 'Auditable code. No hidden logic, no proprietary dependencies.',
    },
  },

  howItWorks: {
    title: 'How It Works',
    description: 'The key verification flow.',
    
    steps: {
      request: {
        title: '1. Request',
        description: 'A Matrix server queries MXKeys for another server\'s keys via POST /_matrix/key/v2/query',
      },
      cache: {
        title: '2. Cache Check',
        description: 'MXKeys checks memory cache, then PostgreSQL. If valid cached key exists — returns immediately.',
      },
      fetch: {
        title: '3. Server Discovery',
        description: 'On cache miss, MXKeys resolves the target server using .well-known delegation, SRV records, and port fallback — then fetches keys via /_matrix/key/v2/server',
      },
      verify: {
        title: '4. Verify',
        description: 'MXKeys verifies the server\'s self-signature using Ed25519. Invalid signatures are rejected.',
      },
      sign: {
        title: '5. Co-Sign',
        description: 'MXKeys adds its perspective signature (ed25519:mxkeys) — attesting that it verified the keys.',
      },
      respond: {
        title: '6. Respond',
        description: 'Keys with both original and notary signatures are returned to the requesting server.',
      },
    },
  },

  api: {
    title: 'API Endpoints',
    description: 'MXKeys implements Matrix Key Server API and operational probes.',
    
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'Returns MXKeys public keys. Used to verify signatures.',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'Returns a specific MXKeys key by key ID. Responds with M_NOT_FOUND when key is absent.',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'Main notary endpoint. Queries keys for Matrix servers and returns verified keys with MXKeys co-signature.',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'Server version information.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'Health endpoint. Returns service health metadata.',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'Readiness endpoint. Verifies DB connectivity and active signing key.',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'Liveness probe endpoint. Returns process alive state.',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'Detailed service status: uptime, cache metrics, database stats, and optional enterprise feature status.',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'Prometheus metrics exposition for service and runtime telemetry.',
    },
    errorsTitle: 'Error model',
    errorsDescription: 'Request validation and abuse controls use Matrix-compatible error codes: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE, and M_LIMIT_EXCEEDED.',
    protectedTitle: 'Protected operational routes',
    protectedDescription: 'Transparency, analytics, cluster, and policy routes require an enterprise access token and are documented separately from the stable public federation API.',
  },

  integration: {
    title: 'Integration',
    description: 'Configure your Matrix server to use MXKeys as a trusted key server.',
    
    synapse: 'Synapse Configuration',
    mxcore: 'MXCore Configuration',
  },

  ecosystem: {
    title: 'Part of Matrix Family',
    description: 'MXKeys is developed by Matrix Family Inc. Available for all Matrix servers.',
    
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
    ecosystem: 'Ecosystem',
    resources: 'Resources',
    contact: 'Contact',
    
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
    
    protocol: 'Protocol',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',

    copyrightPrefix: '© 2026 Matrix Family Inc. All rights reserved. Part of the ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: ' ecosystem.',
    tagline: 'Key Notary for Matrix federation.',
  },
};
