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
    home: 'Inicio',
    about: 'Acerca de',
    howItWorks: 'Cómo funciona',
    api: 'API',
    ecosystem: 'Ecosistema',
    homeAria: 'Inicio de MXKeys',
    github: 'Repositorio GitHub de MXKeys',
    language: 'Idioma',
    openMenu: 'Abrir menú de navegación',
    closeMenu: 'Cerrar menú de navegación',
  },
  hero: {
    title: 'MXKeys',
    subtitle: 'Infraestructura de confianza para federación',
    tagline: 'Confiar. Verificar. Federar.',
    description: 'Capa de confianza de claves de federación para Matrix: verificación de claves, registro de transparencia, detección de anomalías y coordinación de clústeres autenticada.',
    trust: 'Servicio Go con caché PostgreSQL, descubrimiento conforme a la spec Matrix y endpoints operativos.',
    learnMore: 'Más información',
    viewAPI: 'Ver API',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },
  status: {
    online: 'Infraestructura en línea',
  },
  about: {
    title: '¿Qué es MXKeys?',
    description: 'MXKeys es una infraestructura de confianza para la federación Matrix que ayuda a los servidores Matrix a verificar identidades, rastrear cambios de claves, detectar anomalías y aplicar políticas de confianza.',
    problem: {
      title: 'El problema',
      description: 'La federación Matrix depende de claves de servidor con visibilidad limitada. La rotación de claves es difícil de rastrear, los servidores comprometidos son difíciles de detectar, y no existe un registro de auditoría para los cambios de claves. La confianza es implícita.',
    },
    solution: {
      title: 'La solución',
      description: 'MXKeys proporciona verificación de claves con firmas de perspectiva, un registro de transparencia encadenado por hash con pruebas de Merkle, detección de anomalías, políticas de confianza configurables y modos de clúster autenticados.',
    },
  },
  features: {
    title: 'Características',
    description: 'Capacidades de verificación de claves para la federación Matrix.',
    caching: {
      title: 'Caché de claves',
      description: 'Almacena claves verificadas en PostgreSQL. Reduce la latencia y la carga en los servidores de origen.',
    },
    verification: {
      title: 'Verificación de firmas',
      description: 'Valida todas las claves obtenidas contra las firmas del servidor antes del almacenamiento en caché.',
    },
    perspective: {
      title: 'Firma de perspectiva',
      description: 'Añade una co-firma notarial (ed25519:mxkeys) a las claves verificadas — una atestación independiente.',
    },
    discovery: {
      title: 'Descubrimiento de servidores',
      description: 'Soporte de descubrimiento Matrix para delegación .well-known, registros SRV (_matrix-fed._tcp), literales IP y respaldo de puerto dentro del ámbito del notario de claves MXKeys.',
    },
    fallback: {
      title: 'Soporte de respaldo',
      description: 'Si la obtención directa falla, MXKeys puede consultar notarios de respaldo configurados como ruta de confianza operativa explícita.',
    },
    performance: {
      title: 'Alto rendimiento',
      description: 'Escrito en Go. Caché en memoria, pool de conexiones, limpieza eficiente y despliegue en binario único.',
    },
    opensource: {
      title: 'Open Source',
      description: 'Código auditable. Sin lógica oculta, sin dependencias propietarias.',
    },
  },
  howItWorks: {
    title: 'Cómo funciona',
    description: 'El flujo de verificación de claves.',
    steps: {
      request: {
        title: '1. Solicitud',
        description: 'Un servidor Matrix consulta a MXKeys las claves de otro servidor mediante POST /_matrix/key/v2/query',
      },
      cache: {
        title: '2. Verificación de caché',
        description: 'MXKeys comprueba la caché en memoria, luego PostgreSQL. Si existe una clave en caché válida — retorno inmediato.',
      },
      fetch: {
        title: '3. Descubrimiento del servidor',
        description: 'Si no hay caché, MXKeys resuelve el servidor destino mediante delegación .well-known, registros SRV y respaldo de puerto — luego obtiene las claves vía /_matrix/key/v2/server',
      },
      verify: {
        title: '4. Verificación',
        description: 'MXKeys verifica la auto-firma del servidor con Ed25519. Las firmas inválidas son rechazadas.',
      },
      sign: {
        title: '5. Co-firma',
        description: 'MXKeys añade su firma de perspectiva (ed25519:mxkeys) — atestando que verificó las claves.',
      },
      respond: {
        title: '6. Respuesta',
        description: 'Las claves con firmas originales y notariales son devueltas al servidor solicitante.',
      },
    },
  },
  api: {
    title: 'Endpoints de API',
    description: 'MXKeys implementa la API de Matrix Key Server y sondas operativas.',
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'Devuelve las claves públicas de MXKeys. Se utiliza para verificar firmas.',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'Devuelve una clave específica de MXKeys por ID de clave. Responde con M_NOT_FOUND cuando la clave está ausente.',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'Endpoint notarial principal. Consulta claves de servidores Matrix y devuelve claves verificadas con co-firma de MXKeys.',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'Información de versión del servidor.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'Endpoint de salud. Devuelve metadatos de salud del servicio.',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'Endpoint de disponibilidad. Verifica la conectividad con la base de datos y la clave de firma activa.',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'Endpoint de vivacidad. Devuelve el estado de vida del proceso.',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'Estado detallado del servicio: tiempo de actividad, métricas de caché, estadísticas de base de datos y estado de los subsistemas opcionales.',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'Exposición de métricas Prometheus para telemetría del servicio y del runtime.',
    },
    errorsTitle: 'Modelo de errores',
    errorsDescription: 'La validación de solicitudes y los controles anti-abuso utilizan códigos de error compatibles con Matrix: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE y M_LIMIT_EXCEEDED.',
    protectedTitle: 'Rutas operativas admin-only',
    protectedDescription: 'Las rutas de transparencia, análisis, clúster y políticas son superficies ops/debug admin-only. Están protegidas por un token bearer (security.admin_access_token) y se encuentran fuera de la API de federación pública estable.',
  },
  integration: {
    title: 'Integración',
    description: 'Configure su servidor Matrix para usar MXKeys como servidor de claves de confianza.',
    synapse: 'Configuración de Synapse',
    mxcore: 'Configuración de MXCore',
  },
  ecosystem: {
    title: 'Parte de Matrix Family',
    description: 'MXKeys es desarrollado por Matrix Family Inc. Disponible para todos los servidores Matrix.',
    matrixFamily: { title: 'Matrix Family', description: 'Hub del ecosistema' },
    hushme: { title: 'HushMe', description: 'Cliente Matrix' },
    hushmeStore: { title: 'HushMe Store', description: 'Aplicaciones MFOS' },
    mxcore: { title: 'MXCore', description: 'Homeserver Matrix' },
    mfos: { title: 'MFOS', description: 'Plataforma de desarrollo' },
  },
  footer: {
    ecosystem: 'Ecosistema',
    resources: 'Recursos',
    contact: 'Contacto',
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    architecture: 'Arquitectura',
    apiReference: 'Referencia API',
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    protocol: 'Protocolo',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',
    copyrightPrefix: '© 2026 Matrix Family Inc. Todos los derechos reservados. Parte del ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: ' ecosistema.',
    tagline: 'Notario de claves para la federación Matrix.',
  },
};
