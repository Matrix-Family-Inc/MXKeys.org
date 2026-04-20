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
    home: 'Início',
    about: 'Sobre',
    howItWorks: 'Como funciona',
    api: 'API',
    ecosystem: 'Ecossistema',
    homeAria: 'Página inicial MXKeys',
    github: 'Repositório GitHub MXKeys',
    language: 'Idioma',
    openMenu: 'Abrir menu de navegação',
    closeMenu: 'Fechar menu de navegação',
  },
  hero: {
    title: 'MXKeys',
    subtitle: 'Infraestrutura de confiança para federação',
    tagline: 'Confiar. Verificar. Federar.',
    description: 'Camada de confiança de chaves de federação para Matrix: verificação de chaves, registro de transparência, detecção de anomalias e coordenação de clusters autenticada.',
    trust: 'Serviço Go com cache PostgreSQL, descoberta conforme à spec Matrix e endpoints operacionais.',
    learnMore: 'Saiba mais',
    viewAPI: 'Ver API',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },
  status: {
    online: 'Infraestrutura online',
  },
  about: {
    title: 'O que é o MXKeys?',
    description: 'MXKeys é uma infraestrutura de confiança para federação Matrix que ajuda servidores Matrix a verificar identidades, rastrear alterações de chaves, detectar anomalias e aplicar políticas de confiança.',
    problem: {
      title: 'O problema',
      description: 'A federação Matrix depende de chaves de servidor com visibilidade limitada. A rotação de chaves é difícil de rastrear, servidores comprometidos são difíceis de detectar e não existe trilha de auditoria para alterações de chaves. A confiança é implícita.',
    },
    solution: {
      title: 'A solução',
      description: 'MXKeys fornece verificação de chaves com assinaturas de perspectiva, um log de transparência encadeado por hash com provas de Merkle, detecção de anomalias, políticas de confiança configuráveis e modos de cluster autenticados.',
    },
  },
  features: {
    title: 'Funcionalidades',
    description: 'Capacidades de verificação de chaves para federação Matrix.',
    caching: {
      title: 'Cache de chaves',
      description: 'Armazena chaves verificadas no PostgreSQL. Reduz latência e carga nos servidores de origem.',
    },
    verification: {
      title: 'Verificação de assinaturas',
      description: 'Valida todas as chaves obtidas contra as assinaturas do servidor antes do cache.',
    },
    perspective: {
      title: 'Assinatura de perspectiva',
      description: 'Adiciona uma co-assinatura notarial (ed25519:mxkeys) às chaves verificadas — uma atestação independente.',
    },
    discovery: {
      title: 'Descoberta de servidores',
      description: 'Suporte a descoberta Matrix para delegação .well-known, registros SRV (_matrix-fed._tcp), literais IP e fallback de porta no escopo do notário de chaves MXKeys.',
    },
    fallback: {
      title: 'Suporte a fallback',
      description: 'Se a obtenção direta falhar, MXKeys pode consultar notários de fallback configurados como caminho de confiança operacional explícito.',
    },
    performance: {
      title: 'Alto desempenho',
      description: 'Escrito em Go. Cache em memória, pool de conexões, limpeza eficiente e implantação em binário único.',
    },
    opensource: {
      title: 'Open Source',
      description: 'Código auditável. Sem lógica oculta, sem dependências proprietárias.',
    },
  },
  howItWorks: {
    title: 'Como funciona',
    description: 'O fluxo de verificação de chaves.',
    steps: {
      request: {
        title: '1. Requisição',
        description: 'Um servidor Matrix consulta o MXKeys pelas chaves de outro servidor via POST /_matrix/key/v2/query',
      },
      cache: {
        title: '2. Verificação de cache',
        description: 'MXKeys verifica o cache em memória, depois o PostgreSQL. Se existir uma chave em cache válida — retorno imediato.',
      },
      fetch: {
        title: '3. Descoberta do servidor',
        description: 'Em caso de cache miss, MXKeys resolve o servidor alvo usando delegação .well-known, registros SRV e fallback de porta — então obtém as chaves via /_matrix/key/v2/server',
      },
      verify: {
        title: '4. Verificação',
        description: 'MXKeys verifica a auto-assinatura do servidor usando Ed25519. Assinaturas inválidas são rejeitadas.',
      },
      sign: {
        title: '5. Co-assinatura',
        description: 'MXKeys adiciona sua assinatura de perspectiva (ed25519:mxkeys) — atestando que verificou as chaves.',
      },
      respond: {
        title: '6. Resposta',
        description: 'Chaves com assinaturas originais e notariais são retornadas ao servidor solicitante.',
      },
    },
  },
  api: {
    title: 'Endpoints de API',
    description: 'MXKeys implementa a API Matrix Key Server e sondas operacionais.',
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'Retorna as chaves públicas do MXKeys. Usado para verificar assinaturas.',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'Retorna uma chave MXKeys específica por ID de chave. Responde com M_NOT_FOUND quando a chave está ausente.',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'Endpoint notarial principal. Consulta chaves de servidores Matrix e retorna chaves verificadas com co-assinatura MXKeys.',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'Informações de versão do servidor.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'Endpoint de saúde. Retorna metadados de saúde do serviço.',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'Endpoint de prontidão. Verifica a conectividade com o banco de dados e a chave de assinatura ativa.',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'Endpoint de vivacidade. Retorna o estado de vida do processo.',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'Status detalhado do serviço: tempo de atividade, métricas de cache, estatísticas do banco de dados e status opcional de funcionalidades enterprise.',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'Exposição de métricas Prometheus para telemetria do serviço e do runtime.',
    },
    errorsTitle: 'Modelo de erros',
    errorsDescription: 'A validação de requisições e os controles anti-abuso utilizam códigos de erro compatíveis com Matrix: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE e M_LIMIT_EXCEEDED.',
    protectedTitle: 'Rotas operacionais protegidas',
    protectedDescription: 'Rotas de transparência, análise, cluster e políticas requerem um token de acesso enterprise e são documentadas separadamente da API de federação pública estável.',
  },
  integration: {
    title: 'Integração',
    description: 'Configure seu servidor Matrix para usar MXKeys como servidor de chaves confiável.',
    synapse: 'Configuração Synapse',
    mxcore: 'Configuração MXCore',
  },
  ecosystem: {
    title: 'Parte da Matrix Family',
    description: 'MXKeys é desenvolvido pela Matrix Family Inc. Disponível para todos os servidores Matrix.',
    matrixFamily: { title: 'Matrix Family', description: 'Hub do ecossistema' },
    hushme: { title: 'HushMe', description: 'Cliente Matrix' },
    hushmeStore: { title: 'HushMe Store', description: 'Aplicativos MFOS' },
    mxcore: { title: 'MXCore', description: 'Homeserver Matrix' },
    mfos: { title: 'MFOS', description: 'Plataforma de desenvolvimento' },
  },
  footer: {
    ecosystem: 'Ecossistema',
    resources: 'Recursos',
    contact: 'Contato',
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    architecture: 'Arquitetura',
    apiReference: 'Referência API',
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    protocol: 'Protocolo',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',
    copyrightPrefix: '© 2026 Matrix Family Inc. Todos os direitos reservados. Parte do ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: ' ecossistema.',
    tagline: 'Notário de chaves para federação Matrix.',
  },
};
