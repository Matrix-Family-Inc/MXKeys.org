/*
 * Project: MXKeys (mxkeys.org)
 * Company: Matrix.Family Inc. (https://matrix.family)
 * Owner: Matrix.Family Inc.
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Mon 22 Jun 2026 00:50:40 UTC
 * Status: Updated
 */

export default {
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
    description: 'क्लासिक Matrix फ़ेडरेशन के लिए कुंजी नोटरी। होमसर्वर यहाँ सत्यापित सर्वर कुंजियाँ लेते हैं।',
    trust: 'PostgreSQL कैशिंग, Matrix-spec डिस्कवरी, और ऑपरेशनल एंडपॉइंट्स के साथ Go सर्विस।',
    learnMore: 'और जानें',
    viewAPI: 'API देखें',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix.Family',
  },

  status: {
    online: 'इंफ्रास्ट्रक्चर ऑनलाइन',
  },

  about: {
    title: 'MXKeys क्या है?',
    description: 'MXKeys एक Matrix कुंजी नोटरी है। होमसर्वर दूसरे सर्वर की कुंजियाँ माँगता है और परिप्रेक्ष्य हस्ताक्षर वाला सत्यापित दस्तावेज़ पाता है।',

    problem: {
      title: 'समस्या',
      description: 'Matrix फ़ेडरेशन सर्वर कीज़ पर निर्भर है जिनकी दृश्यता सीमित है। की रोटेशन को ट्रैक करना कठिन है, समझौता किए गए सर्वर का पता लगाना कठिन है, और की परिवर्तनों के लिए कोई ऑडिट ट्रेल नहीं है। ट्रस्ट अंतर्निहित है।',
    },

    solution: {
      title: 'समाधान',
      description: 'MXKeys कुंजी दस्तावेज़ लाता है, मूल हस्ताक्षर जाँचता है और अपना जोड़ता है। पारदर्शिता, विश्वास नीति और क्लस्टर मोड तब तक बंद रहते हैं जब तक ऑपरेटर उन्हें चालू न करे।',
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
      description: 'विस्तृत सर्विस स्टेटस: अपटाइम, कैश मेट्रिक्स, डेटाबेस स्टैट्स, और वैकल्पिक सब-सिस्टम की स्थिति।',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'सर्विस और रनटाइम टेलीमेट्री के लिए Prometheus मेट्रिक्स एक्सपोज़िशन।',
    },
    errorsTitle: 'त्रुटि मॉडल',
    errorsDescription: 'अनुरोध मान्यकरण और दुरुपयोग नियंत्रण Matrix-संगत त्रुटि कोड का उपयोग करते हैं: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE, और M_LIMIT_EXCEEDED।',
    protectedTitle: 'केवल एडमिन ऑपरेशनल रूट्स',
    protectedDescription: 'पारदर्शिता, एनालिटिक्स, क्लस्टर, और पॉलिसी रूट्स केवल एडमिन के ops/debug सरफेस हैं। इन्हें bearer टोकन (security.admin_access_token) द्वारा संरक्षित किया गया है और ये स्थिर सार्वजनिक फ़ेडरेशन API के बाहर स्थित हैं।',
  },

  integration: {
    title: 'इंटीग्रेशन',
    description: 'अपने Matrix सर्वर को MXKeys को विश्वसनीय की सर्वर के रूप में उपयोग करने के लिए कॉन्फ़िगर करें।',

    synapse: 'Synapse कॉन्फ़िगरेशन',
    matrixFamily: 'Matrix.Family कॉन्फ़िगरेशन',
  },

  ecosystem: {
    title: 'Matrix.Family का हिस्सा',
    description: 'MXKeys, Matrix.Family के होमसर्वर और क्लाइंट के साथ कुंजी नोटरी है।',

    matrixFamily: {
      title: 'Matrix.Family',
      description: 'होमसर्वर और क्लाइंट',
    },
  },

  footer: {
    ecosystem: 'इकोसिस्टम',
    resources: 'संसाधन',
    contact: 'संपर्क',

    matrixFamily: 'Matrix.Family',

    architecture: 'आर्किटेक्चर',
    apiReference: 'API संदर्भ',

    support: '@support',
    developer: '@dev',
    devChat: '#dev',

    protocol: 'प्रोटोकॉल',
    matrixSpec: 'Matrix Spec',

    copyrightPrefix: '© 2026 Matrix.Family Inc. सर्वाधिकार सुरक्षित। ',
    copyrightLink: 'Matrix.Family',
    copyrightSuffix: ' इकोसिस्टम का हिस्सा।',
    tagline: 'Matrix फ़ेडरेशन के लिए Key Notary।',
  },
};
