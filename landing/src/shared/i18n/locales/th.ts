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
    home: 'หน้าแรก',
    about: 'เกี่ยวกับ',
    howItWorks: 'วิธีการทำงาน',
    api: 'API',
    ecosystem: 'ระบบนิเวศ',
    homeAria: 'หน้าแรก MXKeys',
    github: 'คลัง GitHub ของ MXKeys',
    language: 'ภาษา',
    openMenu: 'เปิดเมนูนำทาง',
    closeMenu: 'ปิดเมนูนำทาง',
  },

  hero: {
    title: 'MXKeys',
    subtitle: 'โครงสร้างพื้นฐานความน่าเชื่อถือสหพันธ์',
    tagline: 'เชื่อถือ ตรวจสอบ สหพันธ์',
    description: 'ชั้นความน่าเชื่อถือคีย์สหพันธ์สำหรับ Matrix: การตรวจสอบคีย์ การบันทึกความโปร่งใส การตรวจจับความผิดปกติ และการประสานงานคลัสเตอร์ที่รับรองแล้ว',
    trust: 'บริการ Go พร้อมแคช PostgreSQL การค้นหา Matrix-spec และ endpoint ปฏิบัติการ',
    learnMore: 'เรียนรู้เพิ่มเติม',
    viewAPI: 'ดู API',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },

  status: {
    online: 'โครงสร้างพื้นฐานออนไลน์',
  },

  about: {
    title: 'MXKeys คืออะไร?',
    description: 'MXKeys คือโครงสร้างพื้นฐานความน่าเชื่อถือสหพันธ์ Matrix ที่ช่วยเซิร์ฟเวอร์ Matrix ตรวจสอบตัวตน ติดตามการเปลี่ยนแปลงคีย์ ตรวจจับความผิดปกติ และบังคับใช้นโยบายความน่าเชื่อถือ',
    
    problem: {
      title: 'ปัญหา',
      description: 'สหพันธ์ Matrix พึ่งพาคีย์เซิร์ฟเวอร์ที่มีทัศนวิสัยจำกัด การหมุนเวียนคีย์ติดตามได้ยาก เซิร์ฟเวอร์ที่ถูกบุกรุกตรวจจับได้ยาก และไม่มีเส้นทางการตรวจสอบสำหรับการเปลี่ยนแปลงคีย์ ความน่าเชื่อถือเป็นนัย',
    },
    
    solution: {
      title: 'ทางออก',
      description: 'MXKeys ให้การตรวจสอบคีย์ด้วยลายเซ็นมุมมอง บันทึกความโปร่งใสแบบเชนแฮชพร้อมหลักฐาน Merkle การตรวจจับความผิดปกติ นโยบายความน่าเชื่อถือที่กำหนดค่าได้ และโหมดคลัสเตอร์ที่รับรองแล้ว',
    },
  },

  features: {
    title: 'คุณสมบัติ',
    description: 'ความสามารถในการตรวจสอบคีย์สำหรับสหพันธ์ Matrix',
    
    caching: {
      title: 'การแคชคีย์',
      description: 'จัดเก็บคีย์ที่ตรวจสอบแล้วใน PostgreSQL ลดเวลาแฝงและภาระบนเซิร์ฟเวอร์ต้นทาง',
    },
    verification: {
      title: 'การตรวจสอบลายเซ็น',
      description: 'ตรวจสอบความถูกต้องของคีย์ที่ดึงมาทั้งหมดกับลายเซ็นเซิร์ฟเวอร์ก่อนการแคช',
    },
    perspective: {
      title: 'การลงนามมุมมอง',
      description: 'เพิ่มลายเซ็นร่วมจากผู้รับรอง (ed25519:mxkeys) ลงในคีย์ที่ตรวจสอบแล้ว — การรับรองที่เป็นอิสระ',
    },
    discovery: {
      title: 'การค้นหาเซิร์ฟเวอร์',
      description: 'รองรับการค้นหา Matrix สำหรับการมอบหมาย .well-known ระเบียน SRV (_matrix-fed._tcp) IP ตรงตัว และ port fallback ภายในขอบเขตผู้รับรองคีย์ MXKeys',
    },
    fallback: {
      title: 'รองรับ Fallback',
      description: 'หากการดึงโดยตรงล้มเหลว MXKeys สามารถสอบถามผู้รับรองสำรองที่กำหนดค่าไว้เป็นเส้นทางความน่าเชื่อถือปฏิบัติการที่ชัดเจน',
    },
    performance: {
      title: 'ประสิทธิภาพสูง',
      description: 'เขียนด้วย Go แคชหน่วยความจำ การรวมการเชื่อมต่อ การล้างที่มีประสิทธิภาพ และการปรับใช้ไบนารีเดี่ยว',
    },
    opensource: {
      title: 'โอเพนซอร์ส',
      description: 'โค้ดที่ตรวจสอบได้ ไม่มีลอจิกซ่อน ไม่มีการพึ่งพาที่เป็นกรรมสิทธิ์',
    },
  },

  howItWorks: {
    title: 'วิธีการทำงาน',
    description: 'ขั้นตอนการตรวจสอบคีย์',
    
    steps: {
      request: {
        title: '1. คำขอ',
        description: 'เซิร์ฟเวอร์ Matrix สอบถาม MXKeys เพื่อขอคีย์ของเซิร์ฟเวอร์อื่นผ่าน POST /_matrix/key/v2/query',
      },
      cache: {
        title: '2. ตรวจสอบแคช',
        description: 'MXKeys ตรวจสอบแคชหน่วยความจำ จากนั้น PostgreSQL หากมีคีย์แคชที่ถูกต้อง — คืนค่าทันที',
      },
      fetch: {
        title: '3. การค้นหาเซิร์ฟเวอร์',
        description: 'เมื่อแคชพลาด MXKeys จะแก้ไขเซิร์ฟเวอร์เป้าหมายโดยใช้การมอบหมาย .well-known ระเบียน SRV และ port fallback — จากนั้นดึงคีย์ผ่าน /_matrix/key/v2/server',
      },
      verify: {
        title: '4. ตรวจสอบ',
        description: 'MXKeys ตรวจสอบลายเซ็นตนเองของเซิร์ฟเวอร์โดยใช้ Ed25519 ลายเซ็นที่ไม่ถูกต้องจะถูกปฏิเสธ',
      },
      sign: {
        title: '5. ลงนามร่วม',
        description: 'MXKeys เพิ่มลายเซ็นมุมมองของตน (ed25519:mxkeys) — รับรองว่าได้ตรวจสอบคีย์แล้ว',
      },
      respond: {
        title: '6. ตอบกลับ',
        description: 'คีย์ที่มีทั้งลายเซ็นดั้งเดิมและลายเซ็นผู้รับรองจะถูกส่งกลับไปยังเซิร์ฟเวอร์ที่ร้องขอ',
      },
    },
  },

  api: {
    title: 'Endpoint API',
    description: 'MXKeys ใช้งาน Matrix Key Server API และโพรบปฏิบัติการ',
    
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'คืนคีย์สาธารณะของ MXKeys ใช้สำหรับตรวจสอบลายเซ็น',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'คืนคีย์ MXKeys เฉพาะตาม ID คีย์ ตอบกลับด้วย M_NOT_FOUND เมื่อไม่มีคีย์',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'Endpoint ผู้รับรองหลัก สอบถามคีย์สำหรับเซิร์ฟเวอร์ Matrix และคืนคีย์ที่ตรวจสอบแล้วพร้อมลายเซ็นร่วม MXKeys',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'ข้อมูลเวอร์ชันเซิร์ฟเวอร์',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'Endpoint สุขภาพ คืนข้อมูลเมตาสุขภาพของบริการ',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'Endpoint ความพร้อม ตรวจสอบการเชื่อมต่อ DB และคีย์ลงนามที่ใช้งานอยู่',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'Endpoint โพรบความมีชีวิต คืนสถานะกระบวนการที่ทำงานอยู่',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'สถานะบริการโดยละเอียด: เวลาทำงาน เมทริกซ์แคช สถิติฐานข้อมูล และสถานะของระบบย่อยที่เป็นทางเลือก',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'การแสดงเมทริกซ์ Prometheus สำหรับเทเลเมทรีบริการและรันไทม์',
    },
    errorsTitle: 'โมเดลข้อผิดพลาด',
    errorsDescription: 'การตรวจสอบคำขอและการควบคุมการใช้งานในทางที่ผิดใช้รหัสข้อผิดพลาดที่เข้ากันได้กับ Matrix: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE และ M_LIMIT_EXCEEDED',
    protectedTitle: 'เส้นทางปฏิบัติการสำหรับผู้ดูแลเท่านั้น',
    protectedDescription: 'เส้นทางความโปร่งใส การวิเคราะห์ คลัสเตอร์ และนโยบายเป็นพื้นผิว ops/debug สำหรับผู้ดูแลเท่านั้น ปกป้องด้วยโทเค็น bearer (security.admin_access_token) และอยู่นอก API สหพันธ์สาธารณะที่เสถียร',
  },

  integration: {
    title: 'การรวมระบบ',
    description: 'กำหนดค่าเซิร์ฟเวอร์ Matrix ของคุณเพื่อใช้ MXKeys เป็นเซิร์ฟเวอร์คีย์ที่น่าเชื่อถือ',
    
    synapse: 'การกำหนดค่า Synapse',
    mxcore: 'การกำหนดค่า MXCore',
  },

  ecosystem: {
    title: 'ส่วนหนึ่งของ Matrix Family',
    description: 'MXKeys พัฒนาโดย Matrix Family Inc. ใช้งานได้กับเซิร์ฟเวอร์ Matrix ทั้งหมด',
    
    matrixFamily: {
      title: 'Matrix Family',
      description: 'ศูนย์กลางระบบนิเวศ',
    },
    hushme: {
      title: 'HushMe',
      description: 'ไคลเอนต์ Matrix',
    },
    hushmeStore: {
      title: 'HushMe Store',
      description: 'แอป MFOS',
    },
    mxcore: {
      title: 'MXCore',
      description: 'โฮมเซิร์ฟเวอร์ Matrix',
    },
    mfos: {
      title: 'MFOS',
      description: 'แพลตฟอร์มนักพัฒนา',
    },
  },

  footer: {
    ecosystem: 'ระบบนิเวศ',
    resources: 'ทรัพยากร',
    contact: 'ติดต่อ',
    
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    
    architecture: 'สถาปัตยกรรม',
    apiReference: 'อ้างอิง API',
    
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    
    protocol: 'โปรโตคอล',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',

    copyrightPrefix: '© 2026 Matrix Family Inc. สงวนลิขสิทธิ์ ส่วนหนึ่งของระบบนิเวศ ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: '',
    tagline: 'Key Notary สำหรับสหพันธ์ Matrix',
  },
};
