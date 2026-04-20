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
    home: 'Trang chủ',
    about: 'Giới thiệu',
    howItWorks: 'Cách hoạt động',
    api: 'API',
    ecosystem: 'Hệ sinh thái',
    homeAria: 'Trang chủ MXKeys',
    github: 'Kho lưu trữ GitHub MXKeys',
    language: 'Ngôn ngữ',
    openMenu: 'Mở menu điều hướng',
    closeMenu: 'Đóng menu điều hướng',
  },

  hero: {
    title: 'MXKeys',
    subtitle: 'Hạ tầng tin cậy liên bang',
    tagline: 'Tin cậy. Xác minh. Liên bang.',
    description: 'Tầng tin cậy khóa liên bang cho Matrix: xác minh khóa, ghi nhật ký minh bạch, phát hiện bất thường và phối hợp cụm được xác thực.',
    trust: 'Dịch vụ Go với bộ nhớ đệm PostgreSQL, khám phá Matrix-spec và các endpoint vận hành.',
    learnMore: 'Tìm hiểu thêm',
    viewAPI: 'Xem API',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },

  status: {
    online: 'Hạ tầng trực tuyến',
  },

  about: {
    title: 'MXKeys là gì?',
    description: 'MXKeys là Hạ tầng tin cậy liên bang Matrix giúp các máy chủ Matrix xác minh danh tính, theo dõi thay đổi khóa, phát hiện bất thường và thực thi chính sách tin cậy.',
    
    problem: {
      title: 'Vấn đề',
      description: 'Liên bang Matrix dựa trên khóa máy chủ với khả năng hiển thị hạn chế. Việc xoay khóa khó theo dõi, máy chủ bị xâm phạm khó phát hiện và không có dấu vết kiểm toán cho các thay đổi khóa. Sự tin cậy là ngầm định.',
    },
    
    solution: {
      title: 'Giải pháp',
      description: 'MXKeys cung cấp xác minh khóa với chữ ký phối cảnh, nhật ký minh bạch chuỗi băm với bằng chứng Merkle, phát hiện bất thường, chính sách tin cậy có thể cấu hình và các chế độ cụm được xác thực.',
    },
  },

  features: {
    title: 'Tính năng',
    description: 'Khả năng xác minh khóa cho liên bang Matrix.',
    
    caching: {
      title: 'Bộ nhớ đệm khóa',
      description: 'Lưu trữ khóa đã xác minh trong PostgreSQL. Giảm độ trễ và tải trên máy chủ gốc.',
    },
    verification: {
      title: 'Xác minh chữ ký',
      description: 'Xác thực tất cả khóa thu được đối chiếu với chữ ký máy chủ trước khi lưu đệm.',
    },
    perspective: {
      title: 'Ký phối cảnh',
      description: 'Thêm chữ ký đồng công chứng (ed25519:mxkeys) vào khóa đã xác minh — một chứng thực độc lập.',
    },
    discovery: {
      title: 'Khám phá máy chủ',
      description: 'Hỗ trợ khám phá Matrix cho ủy quyền .well-known, bản ghi SRV (_matrix-fed._tcp), IP trực tiếp và dự phòng cổng trong phạm vi công chứng khóa MXKeys.',
    },
    fallback: {
      title: 'Hỗ trợ dự phòng',
      description: 'Nếu truy xuất trực tiếp thất bại, MXKeys có thể truy vấn các công chứng viên dự phòng đã cấu hình như một đường dẫn tin cậy vận hành rõ ràng.',
    },
    performance: {
      title: 'Hiệu suất cao',
      description: 'Viết bằng Go. Bộ nhớ đệm bộ nhớ, tổng hợp kết nối, dọn dẹp hiệu quả và triển khai tệp nhị phân đơn.',
    },
    opensource: {
      title: 'Mã nguồn mở',
      description: 'Mã có thể kiểm toán. Không có logic ẩn, không có phụ thuộc độc quyền.',
    },
  },

  howItWorks: {
    title: 'Cách hoạt động',
    description: 'Luồng xác minh khóa.',
    
    steps: {
      request: {
        title: '1. Yêu cầu',
        description: 'Một máy chủ Matrix truy vấn MXKeys để lấy khóa của máy chủ khác qua POST /_matrix/key/v2/query',
      },
      cache: {
        title: '2. Kiểm tra bộ nhớ đệm',
        description: 'MXKeys kiểm tra bộ nhớ đệm bộ nhớ, sau đó PostgreSQL. Nếu khóa đệm hợp lệ tồn tại — trả về ngay lập tức.',
      },
      fetch: {
        title: '3. Khám phá máy chủ',
        description: 'Khi không có trong bộ đệm, MXKeys phân giải máy chủ đích bằng ủy quyền .well-known, bản ghi SRV và dự phòng cổng — sau đó truy xuất khóa qua /_matrix/key/v2/server',
      },
      verify: {
        title: '4. Xác minh',
        description: 'MXKeys xác minh chữ ký tự ký của máy chủ bằng Ed25519. Chữ ký không hợp lệ bị từ chối.',
      },
      sign: {
        title: '5. Đồng ký',
        description: 'MXKeys thêm chữ ký phối cảnh của mình (ed25519:mxkeys) — chứng thực rằng đã xác minh các khóa.',
      },
      respond: {
        title: '6. Phản hồi',
        description: 'Khóa với cả chữ ký gốc và công chứng được trả về cho máy chủ yêu cầu.',
      },
    },
  },

  api: {
    title: 'Endpoint API',
    description: 'MXKeys triển khai Matrix Key Server API và các đầu dò vận hành.',
    
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'Trả về khóa công khai MXKeys. Dùng để xác minh chữ ký.',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'Trả về khóa MXKeys cụ thể theo ID khóa. Phản hồi M_NOT_FOUND khi khóa không tồn tại.',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'Endpoint công chứng chính. Truy vấn khóa cho các máy chủ Matrix và trả về khóa đã xác minh với chữ ký đồng MXKeys.',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'Thông tin phiên bản máy chủ.',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'Endpoint sức khỏe. Trả về siêu dữ liệu sức khỏe dịch vụ.',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'Endpoint sẵn sàng. Xác minh kết nối DB và khóa ký hoạt động.',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'Endpoint đầu dò sống. Trả về trạng thái tiến trình hoạt động.',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'Trạng thái dịch vụ chi tiết: thời gian hoạt động, chỉ số bộ đệm, thống kê cơ sở dữ liệu và trạng thái tính năng doanh nghiệp tùy chọn.',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'Trình bày chỉ số Prometheus cho đo lường từ xa dịch vụ và thời gian chạy.',
    },
    errorsTitle: 'Mô hình lỗi',
    errorsDescription: 'Xác thực yêu cầu và kiểm soát lạm dụng sử dụng mã lỗi tương thích Matrix: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE và M_LIMIT_EXCEEDED.',
    protectedTitle: 'Tuyến vận hành được bảo vệ',
    protectedDescription: 'Các tuyến minh bạch, phân tích, cụm và chính sách yêu cầu mã thông báo truy cập doanh nghiệp và được ghi tài liệu riêng biệt với API liên bang công khai ổn định.',
  },

  integration: {
    title: 'Tích hợp',
    description: 'Cấu hình máy chủ Matrix của bạn để sử dụng MXKeys làm máy chủ khóa tin cậy.',
    
    synapse: 'Cấu hình Synapse',
    mxcore: 'Cấu hình MXCore',
  },

  ecosystem: {
    title: 'Thuộc Matrix Family',
    description: 'MXKeys được phát triển bởi Matrix Family Inc. Khả dụng cho mọi máy chủ Matrix.',
    
    matrixFamily: {
      title: 'Matrix Family',
      description: 'Trung tâm hệ sinh thái',
    },
    hushme: {
      title: 'HushMe',
      description: 'Ứng dụng khách Matrix',
    },
    hushmeStore: {
      title: 'HushMe Store',
      description: 'Ứng dụng MFOS',
    },
    mxcore: {
      title: 'MXCore',
      description: 'Máy chủ Matrix',
    },
    mfos: {
      title: 'MFOS',
      description: 'Nền tảng nhà phát triển',
    },
  },

  footer: {
    ecosystem: 'Hệ sinh thái',
    resources: 'Tài nguyên',
    contact: 'Liên hệ',
    
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    
    architecture: 'Kiến trúc',
    apiReference: 'Tham chiếu API',
    
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    
    protocol: 'Giao thức',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',

    copyrightPrefix: '© 2026 Matrix Family Inc. Mọi quyền được bảo lưu. Thuộc hệ sinh thái ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: '.',
    tagline: 'Key Notary cho liên bang Matrix.',
  },
};
