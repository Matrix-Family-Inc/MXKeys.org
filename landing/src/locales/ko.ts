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

export const ko = {
  nav: {
    home: '홈',
    about: '소개',
    howItWorks: '작동 방식',
    api: 'API',
    ecosystem: '에코시스템',
    homeAria: 'MXKeys 홈',
    github: 'MXKeys GitHub 저장소',
    language: '언어',
    openMenu: '내비게이션 메뉴 열기',
    closeMenu: '내비게이션 메뉴 닫기',
  },
  hero: {
    title: 'MXKeys',
    subtitle: '페더레이션 신뢰 인프라',
    tagline: '신뢰. 검증. 페더레이션.',
    description: 'Matrix 페더레이션 키 신뢰 레이어: 키 검증, 투명성 로깅, 이상 탐지 및 인증된 클러스터 조정.',
    trust: 'PostgreSQL 캐싱, Matrix 사양 디스커버리 및 운영 엔드포인트를 갖춘 Go 서비스.',
    learnMore: '자세히 보기',
    viewAPI: 'API 보기',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },
  status: {
    online: '인프라 온라인',
  },
  about: {
    title: 'MXKeys란?',
    description: 'MXKeys는 Matrix 페더레이션 신뢰 인프라로, Matrix 서버가 신원을 검증하고, 키 변경을 추적하고, 이상을 탐지하고, 신뢰 정책을 시행하도록 지원합니다.',
    problem: {
      title: '문제점',
      description: 'Matrix 페더레이션은 가시성이 제한된 서버 키에 의존합니다. 키 교체는 추적이 어렵고, 침해된 서버는 탐지가 어려우며, 키 변경에 대한 감사 기록이 없습니다. 신뢰는 암묵적입니다.',
    },
    solution: {
      title: '해결책',
      description: 'MXKeys는 퍼스펙티브 서명을 통한 키 검증, Merkle 증명이 포함된 해시 체인 투명성 로그, 이상 탐지, 구성 가능한 신뢰 정책 및 인증된 클러스터 모드를 제공합니다.',
    },
  },
  features: {
    title: '기능',
    description: 'Matrix 페더레이션을 위한 키 검증 기능.',
    caching: {
      title: '키 캐싱',
      description: '검증된 키를 PostgreSQL에 저장합니다. 지연 시간과 원본 서버의 부하를 줄입니다.',
    },
    verification: {
      title: '서명 검증',
      description: '캐싱 전에 모든 가져온 키를 서버 서명에 대해 검증합니다.',
    },
    perspective: {
      title: '퍼스펙티브 서명',
      description: '검증된 키에 공증 공동 서명 (ed25519:mxkeys)을 추가합니다 — 독립적인 증명입니다.',
    },
    discovery: {
      title: '서버 디스커버리',
      description: 'MXKeys 키 공증 범위 내에서 Matrix 디스커버리 지원: .well-known 위임, SRV 레코드 (_matrix-fed._tcp), IP 리터럴 및 포트 폴백.',
    },
    fallback: {
      title: '폴백 지원',
      description: '직접 가져오기에 실패하면 MXKeys는 구성된 폴백 공증인에게 명시적 운영 신뢰 경로로 쿼리할 수 있습니다.',
    },
    performance: {
      title: '고성능',
      description: 'Go로 작성. 메모리 캐시, 커넥션 풀링, 효율적인 정리 및 단일 바이너리 배포.',
    },
    opensource: {
      title: '오픈 소스',
      description: '감사 가능한 코드. 숨겨진 로직 없음, 독점 의존성 없음.',
    },
  },
  howItWorks: {
    title: '작동 방식',
    description: '키 검증 플로우.',
    steps: {
      request: {
        title: '1. 요청',
        description: 'Matrix 서버가 POST /_matrix/key/v2/query를 통해 다른 서버의 키를 MXKeys에 쿼리합니다',
      },
      cache: {
        title: '2. 캐시 확인',
        description: 'MXKeys가 메모리 캐시를 확인한 다음 PostgreSQL을 확인합니다. 유효한 캐시된 키가 있으면 — 즉시 반환합니다.',
      },
      fetch: {
        title: '3. 서버 디스커버리',
        description: '캐시 미스 시, MXKeys는 .well-known 위임, SRV 레코드, 포트 폴백을 사용하여 대상 서버를 확인한 다음 /_matrix/key/v2/server를 통해 키를 가져옵니다',
      },
      verify: {
        title: '4. 검증',
        description: 'MXKeys가 Ed25519를 사용하여 서버의 자체 서명을 검증합니다. 유효하지 않은 서명은 거부됩니다.',
      },
      sign: {
        title: '5. 공동 서명',
        description: 'MXKeys가 퍼스펙티브 서명 (ed25519:mxkeys)을 추가합니다 — 키를 검증했음을 증명합니다.',
      },
      respond: {
        title: '6. 응답',
        description: '원본 서명과 공증 서명이 모두 포함된 키가 요청 서버로 반환됩니다.',
      },
    },
  },
  api: {
    title: 'API 엔드포인트',
    description: 'MXKeys는 Matrix Key Server API와 운영 프로브를 구현합니다.',
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'MXKeys 공개 키를 반환합니다. 서명 검증에 사용됩니다.',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: '키 ID로 특정 MXKeys 키를 반환합니다. 키가 없으면 M_NOT_FOUND로 응답합니다.',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: '메인 공증 엔드포인트. Matrix 서버의 키를 쿼리하고 MXKeys 공동 서명이 포함된 검증된 키를 반환합니다.',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: '서버 버전 정보.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: '헬스 엔드포인트. 서비스 헬스 메타데이터를 반환합니다.',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: '레디니스 엔드포인트. DB 연결 및 활성 서명 키를 확인합니다.',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: '라이브니스 프로브 엔드포인트. 프로세스 생존 상태를 반환합니다.',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: '상세 서비스 상태: 가동 시간, 캐시 메트릭, 데이터베이스 통계 및 선택적 엔터프라이즈 기능 상태.',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: '서비스 및 런타임 텔레메트리를 위한 Prometheus 메트릭 노출.',
    },
    errorsTitle: '오류 모델',
    errorsDescription: '요청 검증 및 악용 제어는 Matrix 호환 오류 코드를 사용합니다: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE, M_LIMIT_EXCEEDED.',
    protectedTitle: '보호된 운영 라우트',
    protectedDescription: '투명성, 분석, 클러스터 및 정책 라우트에는 엔터프라이즈 액세스 토큰이 필요하며, 안정적인 공개 페더레이션 API와 별도로 문서화되어 있습니다.',
  },
  integration: {
    title: '통합',
    description: 'Matrix 서버에서 MXKeys를 신뢰할 수 있는 키 서버로 사용하도록 구성하세요.',
    synapse: 'Synapse 구성',
    mxcore: 'MXCore 구성',
  },
  ecosystem: {
    title: 'Matrix Family의 일원',
    description: 'MXKeys는 Matrix Family Inc.에서 개발합니다. 모든 Matrix 서버에서 사용 가능합니다.',
    matrixFamily: { title: 'Matrix Family', description: '에코시스템 허브' },
    hushme: { title: 'HushMe', description: 'Matrix 클라이언트' },
    hushmeStore: { title: 'HushMe Store', description: 'MFOS 앱' },
    mxcore: { title: 'MXCore', description: 'Matrix 홈서버' },
    mfos: { title: 'MFOS', description: '개발자 플랫폼' },
  },
  footer: {
    ecosystem: '에코시스템',
    resources: '리소스',
    contact: '연락처',
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    architecture: '아키텍처',
    apiReference: 'API 레퍼런스',
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    protocol: '프로토콜',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',
    copyrightPrefix: '© 2026 Matrix Family Inc. All rights reserved. ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: ' 에코시스템의 일원입니다.',
    tagline: 'Matrix 페더레이션을 위한 키 공증 서비스.',
  },
};
