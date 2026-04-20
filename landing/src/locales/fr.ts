/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Sun Apr 13 2026 UTC
 * Status: Created
 */

export const fr = {
  nav: {
    home: 'Accueil',
    about: 'À propos',
    howItWorks: 'Fonctionnement',
    api: 'API',
    ecosystem: 'Écosystème',
    homeAria: 'Accueil MXKeys',
    github: 'Dépôt GitHub MXKeys',
    language: 'Langue',
    openMenu: 'Ouvrir le menu de navigation',
    closeMenu: 'Fermer le menu de navigation',
  },
  hero: {
    title: 'MXKeys',
    subtitle: 'Infrastructure de confiance pour la fédération',
    tagline: 'Confiance. Vérification. Fédération.',
    description: 'Couche de confiance des clés de fédération pour Matrix : vérification des clés, journalisation de transparence, détection d\'anomalies et coordination de cluster authentifiée.',
    trust: 'Service Go avec cache PostgreSQL, découverte conforme à la spec Matrix et points de terminaison opérationnels.',
    learnMore: 'En savoir plus',
    viewAPI: 'Voir l\'API',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },
  status: {
    online: 'Infrastructure en ligne',
  },
  about: {
    title: 'Qu\'est-ce que MXKeys ?',
    description: 'MXKeys est une infrastructure de confiance pour la fédération Matrix qui aide les serveurs Matrix à vérifier les identités, suivre les changements de clés, détecter les anomalies et appliquer les politiques de confiance.',
    problem: {
      title: 'Le problème',
      description: 'La fédération Matrix repose sur des clés serveur avec une visibilité limitée. La rotation des clés est difficile à suivre, les serveurs compromis sont difficiles à détecter, et il n\'existe aucune piste d\'audit pour les changements de clés. La confiance est implicite.',
    },
    solution: {
      title: 'La solution',
      description: 'MXKeys fournit la vérification des clés avec des signatures de perspective, un journal de transparence chaîné par hachage avec des preuves de Merkle, la détection d\'anomalies, des politiques de confiance configurables et des modes de cluster authentifiés.',
    },
  },
  features: {
    title: 'Fonctionnalités',
    description: 'Capacités de vérification des clés pour la fédération Matrix.',
    caching: {
      title: 'Cache de clés',
      description: 'Stocke les clés vérifiées dans PostgreSQL. Réduit la latence et la charge sur les serveurs d\'origine.',
    },
    verification: {
      title: 'Vérification des signatures',
      description: 'Valide toutes les clés récupérées par rapport aux signatures du serveur avant la mise en cache.',
    },
    perspective: {
      title: 'Signature de perspective',
      description: 'Ajoute une co-signature notariale (ed25519:mxkeys) aux clés vérifiées — une attestation indépendante.',
    },
    discovery: {
      title: 'Découverte de serveurs',
      description: 'Support de découverte Matrix pour la délégation .well-known, les enregistrements SRV (_matrix-fed._tcp), les littéraux IP et le repli de port dans le cadre du notaire de clés MXKeys.',
    },
    fallback: {
      title: 'Support de repli',
      description: 'Si la récupération directe échoue, MXKeys peut interroger des notaires de repli configurés comme chemin de confiance opérationnel explicite.',
    },
    performance: {
      title: 'Haute performance',
      description: 'Écrit en Go. Cache mémoire, pool de connexions, nettoyage efficace et déploiement en binaire unique.',
    },
    opensource: {
      title: 'Open Source',
      description: 'Code auditable. Aucune logique cachée, aucune dépendance propriétaire.',
    },
  },
  howItWorks: {
    title: 'Fonctionnement',
    description: 'Le flux de vérification des clés.',
    steps: {
      request: {
        title: '1. Requête',
        description: 'Un serveur Matrix interroge MXKeys pour obtenir les clés d\'un autre serveur via POST /_matrix/key/v2/query',
      },
      cache: {
        title: '2. Vérification du cache',
        description: 'MXKeys vérifie le cache mémoire, puis PostgreSQL. Si une clé en cache valide existe — retour immédiat.',
      },
      fetch: {
        title: '3. Découverte du serveur',
        description: 'En cas d\'absence en cache, MXKeys résout le serveur cible via la délégation .well-known, les enregistrements SRV et le repli de port — puis récupère les clés via /_matrix/key/v2/server',
      },
      verify: {
        title: '4. Vérification',
        description: 'MXKeys vérifie l\'auto-signature du serveur avec Ed25519. Les signatures invalides sont rejetées.',
      },
      sign: {
        title: '5. Co-signature',
        description: 'MXKeys ajoute sa signature de perspective (ed25519:mxkeys) — attestant qu\'il a vérifié les clés.',
      },
      respond: {
        title: '6. Réponse',
        description: 'Les clés avec les signatures originales et notariales sont retournées au serveur demandeur.',
      },
    },
  },
  api: {
    title: 'Points de terminaison API',
    description: 'MXKeys implémente l\'API Matrix Key Server et les sondes opérationnelles.',
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'Retourne les clés publiques de MXKeys. Utilisé pour vérifier les signatures.',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'Retourne une clé MXKeys spécifique par identifiant de clé. Répond avec M_NOT_FOUND lorsque la clé est absente.',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'Point de terminaison notarial principal. Interroge les clés des serveurs Matrix et retourne les clés vérifiées avec la co-signature MXKeys.',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'Informations de version du serveur.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'Point de terminaison de santé. Retourne les métadonnées de santé du service.',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'Point de terminaison de disponibilité. Vérifie la connectivité à la base de données et la clé de signature active.',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'Point de terminaison de vivacité. Retourne l\'état de vie du processus.',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'Statut détaillé du service : temps de fonctionnement, métriques de cache, statistiques de base de données et statut optionnel des fonctionnalités entreprise.',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'Exposition des métriques Prometheus pour la télémétrie du service et du runtime.',
    },
    errorsTitle: 'Modèle d\'erreur',
    errorsDescription: 'La validation des requêtes et les contrôles anti-abus utilisent des codes d\'erreur compatibles Matrix : M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE et M_LIMIT_EXCEEDED.',
    protectedTitle: 'Routes opérationnelles protégées',
    protectedDescription: 'Les routes de transparence, d\'analyse, de cluster et de politique nécessitent un jeton d\'accès entreprise et sont documentées séparément de l\'API de fédération publique stable.',
  },
  integration: {
    title: 'Intégration',
    description: 'Configurez votre serveur Matrix pour utiliser MXKeys comme serveur de clés de confiance.',
    synapse: 'Configuration Synapse',
    mxcore: 'Configuration MXCore',
  },
  ecosystem: {
    title: 'Membre de Matrix Family',
    description: 'MXKeys est développé par Matrix Family Inc. Disponible pour tous les serveurs Matrix.',
    matrixFamily: { title: 'Matrix Family', description: 'Hub de l\'écosystème' },
    hushme: { title: 'HushMe', description: 'Client Matrix' },
    hushmeStore: { title: 'HushMe Store', description: 'Applications MFOS' },
    mxcore: { title: 'MXCore', description: 'Homeserver Matrix' },
    mfos: { title: 'MFOS', description: 'Plateforme développeur' },
  },
  footer: {
    ecosystem: 'Écosystème',
    resources: 'Ressources',
    contact: 'Contact',
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    architecture: 'Architecture',
    apiReference: 'Référence API',
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    protocol: 'Protocole',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',
    copyrightPrefix: '© 2026 Matrix Family Inc. Tous droits réservés. Membre de l\'',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: ' écosystème.',
    tagline: 'Notaire de clés pour la fédération Matrix.',
  },
};
