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
    home: 'Strona główna',
    about: 'O projekcie',
    howItWorks: 'Jak to działa',
    api: 'API',
    ecosystem: 'Ekosystem',
    homeAria: 'Strona główna MXKeys',
    github: 'Repozytorium GitHub MXKeys',
    language: 'Język',
    openMenu: 'Otwórz menu nawigacji',
    closeMenu: 'Zamknij menu nawigacji',
  },

  hero: {
    title: 'MXKeys',
    subtitle: 'Infrastruktura zaufania federacji',
    tagline: 'Zaufanie. Weryfikacja. Federacja.',
    description: 'Warstwa zaufania kluczy federacji dla Matrix: weryfikacja kluczy, rejestrowanie przejrzystości, wykrywanie anomalii i uwierzytelniona koordynacja klastrów.',
    trust: 'Usługa Go z buforowaniem PostgreSQL, wykrywaniem Matrix-spec i punktami końcowymi operacyjnymi.',
    learnMore: 'Dowiedz się więcej',
    viewAPI: 'Zobacz API',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },

  status: {
    online: 'Infrastruktura online',
  },

  about: {
    title: 'Czym jest MXKeys?',
    description: 'MXKeys to infrastruktura zaufania federacji Matrix, która pomaga serwerom Matrix weryfikować tożsamości, śledzić zmiany kluczy, wykrywać anomalie i egzekwować polityki zaufania.',
    
    problem: {
      title: 'Problem',
      description: 'Federacja Matrix opiera się na kluczach serwerów o ograniczonej widoczności. Rotacja kluczy jest trudna do śledzenia, skompromitowane serwery trudne do wykrycia, a zmiany kluczy nie mają ścieżki audytu. Zaufanie jest niejawne.',
    },
    
    solution: {
      title: 'Rozwiązanie',
      description: 'MXKeys zapewnia weryfikację kluczy z podpisami perspektywicznymi, łańcuchowany haszowo dziennik przejrzystości z dowodami Merkle, wykrywanie anomalii, konfigurowalne polityki zaufania i uwierzytelnione tryby klastrów.',
    },
  },

  features: {
    title: 'Funkcje',
    description: 'Możliwości weryfikacji kluczy dla federacji Matrix.',
    
    caching: {
      title: 'Buforowanie kluczy',
      description: 'Przechowuje zweryfikowane klucze w PostgreSQL. Zmniejsza opóźnienia i obciążenie serwerów źródłowych.',
    },
    verification: {
      title: 'Weryfikacja podpisów',
      description: 'Waliduje wszystkie pobrane klucze względem podpisów serwera przed buforowaniem.',
    },
    perspective: {
      title: 'Podpisywanie perspektywiczne',
      description: 'Dodaje współpodpis notarialny (ed25519:mxkeys) do zweryfikowanych kluczy — niezależne poświadczenie.',
    },
    discovery: {
      title: 'Wykrywanie serwerów',
      description: 'Obsługa wykrywania Matrix dla delegacji .well-known, rekordów SRV (_matrix-fed._tcp), literałów IP i awaryjnego portu w zakresie notariatu kluczy MXKeys.',
    },
    fallback: {
      title: 'Obsługa awaryjna',
      description: 'Jeśli bezpośrednie pobieranie się nie powiedzie, MXKeys może wysłać zapytanie do skonfigurowanych notariuszy awaryjnych jako jawna ścieżka zaufania operacyjnego.',
    },
    performance: {
      title: 'Wysoka wydajność',
      description: 'Napisany w Go. Buforowanie pamięci, pule połączeń, efektywne czyszczenie i wdrażanie pojedynczego pliku binarnego.',
    },
    opensource: {
      title: 'Otwarte źródło',
      description: 'Kod podlegający audytowi. Bez ukrytej logiki, bez zależności zastrzeżonych.',
    },
  },

  howItWorks: {
    title: 'Jak to działa',
    description: 'Przepływ weryfikacji kluczy.',
    
    steps: {
      request: {
        title: '1. Żądanie',
        description: 'Serwer Matrix wysyła zapytanie do MXKeys o klucze innego serwera przez POST /_matrix/key/v2/query',
      },
      cache: {
        title: '2. Sprawdzenie bufora',
        description: 'MXKeys sprawdza bufor pamięci, następnie PostgreSQL. Jeśli istnieje prawidłowy zbuforowany klucz — zwraca natychmiast.',
      },
      fetch: {
        title: '3. Wykrywanie serwera',
        description: 'Przy braku w buforze, MXKeys rozwiązuje serwer docelowy za pomocą delegacji .well-known, rekordów SRV i awaryjnego portu — następnie pobiera klucze przez /_matrix/key/v2/server',
      },
      verify: {
        title: '4. Weryfikacja',
        description: 'MXKeys weryfikuje samopodpis serwera za pomocą Ed25519. Nieprawidłowe podpisy są odrzucane.',
      },
      sign: {
        title: '5. Współpodpis',
        description: 'MXKeys dodaje swój podpis perspektywiczny (ed25519:mxkeys) — poświadczając, że zweryfikował klucze.',
      },
      respond: {
        title: '6. Odpowiedź',
        description: 'Klucze z oryginalnymi i notarialnymi podpisami są zwracane do serwera żądającego.',
      },
    },
  },

  api: {
    title: 'Punkty końcowe API',
    description: 'MXKeys implementuje Matrix Key Server API i sondy operacyjne.',
    
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'Zwraca klucze publiczne MXKeys. Używane do weryfikacji podpisów.',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'Zwraca konkretny klucz MXKeys według identyfikatora klucza. Odpowiada M_NOT_FOUND, gdy klucz jest nieobecny.',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'Główny punkt końcowy notariatu. Wysyła zapytania o klucze serwerów Matrix i zwraca zweryfikowane klucze ze współpodpisem MXKeys.',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'Informacje o wersji serwera.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'Punkt końcowy zdrowia. Zwraca metadane zdrowia usługi.',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'Punkt końcowy gotowości. Weryfikuje łączność z bazą danych i aktywny klucz podpisu.',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'Punkt końcowy sondy żywotności. Zwraca stan procesu.',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'Szczegółowy status usługi: czas działania, metryki bufora, statystyki bazy danych i opcjonalny status funkcji enterprise.',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'Ekspozycja metryk Prometheus dla telemetrii usługi i środowiska uruchomieniowego.',
    },
    errorsTitle: 'Model błędów',
    errorsDescription: 'Walidacja żądań i kontrole nadużyć używają kodów błędów kompatybilnych z Matrix: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE i M_LIMIT_EXCEEDED.',
    protectedTitle: 'Chronione trasy operacyjne',
    protectedDescription: 'Trasy przejrzystości, analityki, klastrów i polityk wymagają tokenu dostępu enterprise i są dokumentowane oddzielnie od stabilnego publicznego API federacji.',
  },

  integration: {
    title: 'Integracja',
    description: 'Skonfiguruj swój serwer Matrix, aby używał MXKeys jako zaufanego serwera kluczy.',
    
    synapse: 'Konfiguracja Synapse',
    mxcore: 'Konfiguracja MXCore',
  },

  ecosystem: {
    title: 'Część Matrix Family',
    description: 'MXKeys jest rozwijany przez Matrix Family Inc. Dostępny dla wszystkich serwerów Matrix.',
    
    matrixFamily: {
      title: 'Matrix Family',
      description: 'Centrum ekosystemu',
    },
    hushme: {
      title: 'HushMe',
      description: 'Klient Matrix',
    },
    hushmeStore: {
      title: 'HushMe Store',
      description: 'Aplikacje MFOS',
    },
    mxcore: {
      title: 'MXCore',
      description: 'Serwer domowy Matrix',
    },
    mfos: {
      title: 'MFOS',
      description: 'Platforma deweloperska',
    },
  },

  footer: {
    ecosystem: 'Ekosystem',
    resources: 'Zasoby',
    contact: 'Kontakt',
    
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    
    architecture: 'Architektura',
    apiReference: 'Referencja API',
    
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    
    protocol: 'Protokół',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',

    copyrightPrefix: '© 2026 Matrix Family Inc. Wszelkie prawa zastrzeżone. Część ekosystemu ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: '.',
    tagline: 'Key Notary dla federacji Matrix.',
  },
};
