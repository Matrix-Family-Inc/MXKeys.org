/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Sun Apr 13 2026 UTC
 * Status: Created
 */

export const hi = {
  nav: {
    home: 'होम',
    about: 'परिचय',
    howItWorks: 'यह कैसे काम करता है',
    api: 'API',
    ecosystem: 'इकोसिस्टम',
    homeAria: 'MXKeys होम',
    github: 'MXKeys GitHub रिपॉज़िटरी',
    language: 'भाषा',
    openMenu: 'नेविगेशन मेनू खोलें',
    closeMenu: 'नेविगेशन मेनू बंद करें',
  },

  hero: {
    title: 'MXKeys',
    subtitle: 'फ़ेडरेशन ट्रस्ट इंफ्रास्ट्रक्चर',
    tagline: 'विश्वास। सत्यापन। फ़ेडरेशन।',
    description: 'Matrix के लिए फ़ेडरेशन की ट्रस्ट लेयर: की सत्यापन, पारदर्शिता लॉगिंग, विसंगति पहचान, और प्रमाणित क्लस्टर समन्वय।',
    trust: 'PostgreSQL कैशिंग, Matrix-spec डिस्कवरी, और ऑपरेशनल एंडपॉइंट्स के साथ Go सर्विस।',
    learnMore: 'और जानें',
    viewAPI: 'API देखें',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },

  status: {
    online: 'इंफ्रास्ट्रक्चर ऑनलाइन',
  },

  about: {
    title: 'MXKeys क्या है?',
    description: 'MXKeys एक Matrix फ़ेडरेशन ट्रस्ट इंफ्रास्ट्रक्चर है जो Matrix सर्वर को पहचान सत्यापित करने, की परिवर्तनों को ट्रैक करने, विसंगतियों का पता लगाने, और ट्रस्ट नीतियाँ लागू करने में सहायता करता है।',
    
    problem: {
      title: 'समस्या',
      description: 'Matrix फ़ेडरेशन सर्वर कीज़ पर निर्भर है जिनकी दृश्यता सीमित है। की रोटेशन को ट्रैक करना कठिन है, समझौता किए गए सर्वर का पता लगाना कठिन है, और की परिवर्तनों के लिए कोई ऑडिट ट्रेल नहीं है। ट्रस्ट अंतर्निहित है।',
    },
    
    solution: {
      title: 'समाधान',
      description: 'MXKeys पर्सपेक्टिव सिग्नेचर के साथ की सत्यापन, Merkle प्रमाणों के साथ हैश-चेन्ड पारदर्शिता लॉग, विसंगति पहचान, कॉन्फ़िगर करने योग्य ट्रस्ट नीतियाँ, और प्रमाणित क्लस्टर मोड प्रदान करता है।',
    },
  },

  features: {
    title: 'विशेषताएँ',
    description: 'Matrix फ़ेडरेशन के लिए की सत्यापन क्षमताएँ।',
    
    caching: {
      title: 'की कैशिंग',
      description: 'सत्यापित कीज़ को PostgreSQL में संग्रहीत करता है। ओरिजिन सर्वर पर लेटेंसी और लोड कम करता है।',
    },
    verification: {
      title: 'सिग्नेचर सत्यापन',
      description: 'कैशिंग से पहले सर्वर सिग्नेचर के विरुद्ध सभी प्राप्त कीज़ को मान्य करता है।',
    },
    perspective: {
      title: 'पर्सपेक्टिव साइनिंग',
      description: 'सत्यापित कीज़ में एक नोटरी सह-हस्ताक्षर (ed25519:mxkeys) जोड़ता है — एक स्वतंत्र प्रमाणन।',
    },
    discovery: {
      title: 'सर्वर डिस्कवरी',
      description: 'MXKeys की-नोटरी स्कोप में .well-known डेलिगेशन, SRV रिकॉर्ड्स (_matrix-fed._tcp), IP लिटरल्स, और पोर्ट फ़ॉलबैक के लिए Matrix डिस्कवरी समर्थन।',
    },
    fallback: {
      title: 'फ़ॉलबैक समर्थन',
      description: 'यदि प्रत्यक्ष फ़ेच विफल होता है, तो MXKeys एक स्पष्ट ऑपरेशनल ट्रस्ट पथ के रूप में कॉन्फ़िगर किए गए फ़ॉलबैक नोटरीज़ से क्वेरी कर सकता है।',
    },
    performance: {
      title: 'उच्च प्रदर्शन',
      description: 'Go में लिखा गया। मेमोरी कैशिंग, कनेक्शन पूलिंग, कुशल क्लीनअप, और सिंगल-बाइनरी डिप्लॉयमेंट।',
    },
    opensource: {
      title: 'ओपन सोर्स',
      description: 'ऑडिट करने योग्य कोड। कोई छिपा हुआ लॉजिक नहीं, कोई प्रोप्राइटरी डिपेंडेंसी नहीं।',
    },
  },

  howItWorks: {
    title: 'यह कैसे काम करता है',
    description: 'की सत्यापन प्रवाह।',
    
    steps: {
      request: {
        title: '1. अनुरोध',
        description: 'एक Matrix सर्वर POST /_matrix/key/v2/query के माध्यम से किसी अन्य सर्वर की कीज़ के लिए MXKeys से क्वेरी करता है',
      },
      cache: {
        title: '2. कैश जाँच',
        description: 'MXKeys पहले मेमोरी कैश, फिर PostgreSQL जाँचता है। यदि वैध कैश्ड की मौजूद है — तुरंत लौटाता है।',
      },
      fetch: {
        title: '3. सर्वर डिस्कवरी',
        description: 'कैश मिस पर, MXKeys .well-known डेलिगेशन, SRV रिकॉर्ड्स, और पोर्ट फ़ॉलबैक का उपयोग करके लक्ष्य सर्वर को रिज़ॉल्व करता है — फिर /_matrix/key/v2/server के माध्यम से कीज़ प्राप्त करता है',
      },
      verify: {
        title: '4. सत्यापन',
        description: 'MXKeys Ed25519 का उपयोग करके सर्वर के सेल्फ़-सिग्नेचर को सत्यापित करता है। अमान्य सिग्नेचर अस्वीकार किए जाते हैं।',
      },
      sign: {
        title: '5. सह-हस्ताक्षर',
        description: 'MXKeys अपना पर्सपेक्टिव सिग्नेचर (ed25519:mxkeys) जोड़ता है — प्रमाणित करते हुए कि उसने कीज़ सत्यापित की हैं।',
      },
      respond: {
        title: '6. प्रतिक्रिया',
        description: 'मूल और नोटरी दोनों सिग्नेचर के साथ कीज़ अनुरोध करने वाले सर्वर को लौटाई जाती हैं।',
      },
    },
  },

  api: {
    title: 'API एंडपॉइंट्स',
    description: 'MXKeys Matrix Key Server API और ऑपरेशनल प्रोब्स लागू करता है।',
    
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'MXKeys सार्वजनिक कीज़ लौटाता है। सिग्नेचर सत्यापित करने के लिए उपयोग किया जाता है।',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'की ID द्वारा एक विशिष्ट MXKeys की लौटाता है। जब की अनुपस्थित हो तो M_NOT_FOUND के साथ प्रतिक्रिया करता है।',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'मुख्य नोटरी एंडपॉइंट। Matrix सर्वर की कीज़ क्वेरी करता है और MXKeys सह-हस्ताक्षर के साथ सत्यापित कीज़ लौटाता है।',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'सर्वर संस्करण जानकारी।',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'हेल्थ एंडपॉइंट। सर्विस हेल्थ मेटाडेटा लौटाता है।',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'रेडीनेस एंडपॉइंट। DB कनेक्टिविटी और सक्रिय साइनिंग की सत्यापित करता है।',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'लाइवनेस प्रोब एंडपॉइंट। प्रोसेस अलाइव स्टेट लौटाता है।',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'विस्तृत सर्विस स्टेटस: अपटाइम, कैश मेट्रिक्स, डेटाबेस स्टैट्स, और वैकल्पिक एंटरप्राइज़ फ़ीचर स्टेटस।',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'सर्विस और रनटाइम टेलीमेट्री के लिए Prometheus मेट्रिक्स एक्सपोज़िशन।',
    },
    errorsTitle: 'त्रुटि मॉडल',
    errorsDescription: 'अनुरोध मान्यकरण और दुरुपयोग नियंत्रण Matrix-संगत त्रुटि कोड का उपयोग करते हैं: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE, और M_LIMIT_EXCEEDED।',
    protectedTitle: 'संरक्षित ऑपरेशनल रूट्स',
    protectedDescription: 'पारदर्शिता, एनालिटिक्स, क्लस्टर, और पॉलिसी रूट्स के लिए एंटरप्राइज़ एक्सेस टोकन आवश्यक है और इन्हें स्थिर सार्वजनिक फ़ेडरेशन API से अलग दस्तावेज़ित किया गया है।',
  },

  integration: {
    title: 'इंटीग्रेशन',
    description: 'अपने Matrix सर्वर को MXKeys को विश्वसनीय की सर्वर के रूप में उपयोग करने के लिए कॉन्फ़िगर करें।',
    
    synapse: 'Synapse कॉन्फ़िगरेशन',
    mxcore: 'MXCore कॉन्फ़िगरेशन',
  },

  ecosystem: {
    title: 'Matrix Family का हिस्सा',
    description: 'MXKeys को Matrix Family Inc. द्वारा विकसित किया गया है। सभी Matrix सर्वर के लिए उपलब्ध।',
    
    matrixFamily: {
      title: 'Matrix Family',
      description: 'इकोसिस्टम हब',
    },
    hushme: {
      title: 'HushMe',
      description: 'Matrix क्लाइंट',
    },
    hushmeStore: {
      title: 'HushMe Store',
      description: 'MFOS ऐप्स',
    },
    mxcore: {
      title: 'MXCore',
      description: 'Matrix होमसर्वर',
    },
    mfos: {
      title: 'MFOS',
      description: 'डेवलपर प्लेटफ़ॉर्म',
    },
  },

  footer: {
    ecosystem: 'इकोसिस्टम',
    resources: 'संसाधन',
    contact: 'संपर्क',
    
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    
    architecture: 'आर्किटेक्चर',
    apiReference: 'API संदर्भ',
    
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    
    protocol: 'प्रोटोकॉल',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',

    copyrightPrefix: '© 2026 Matrix Family Inc. सर्वाधिकार सुरक्षित। ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: ' इकोसिस्टम का हिस्सा।',
    tagline: 'Matrix फ़ेडरेशन के लिए Key Notary।',
  },
};
