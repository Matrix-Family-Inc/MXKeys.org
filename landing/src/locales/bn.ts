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

export const bn = {
  nav: {
    home: 'হোম',
    about: 'পরিচিতি',
    howItWorks: 'এটি কিভাবে কাজ করে',
    api: 'API',
    ecosystem: 'ইকোসিস্টেম',
    homeAria: 'MXKeys হোম',
    github: 'MXKeys GitHub রিপোজিটরি',
    language: 'ভাষা',
    openMenu: 'নেভিগেশন মেনু খুলুন',
    closeMenu: 'নেভিগেশন মেনু বন্ধ করুন',
  },

  hero: {
    title: 'MXKeys',
    subtitle: 'ফেডারেশন ট্রাস্ট ইনফ্রাস্ট্রাকচার',
    tagline: 'বিশ্বাস। যাচাই। ফেডারেশন।',
    description: 'Matrix-এর জন্য ফেডারেশন কী ট্রাস্ট লেয়ার: কী যাচাই, স্বচ্ছতা লগিং, অসঙ্গতি সনাক্তকরণ, এবং প্রমাণীকৃত ক্লাস্টার সমন্বয়।',
    trust: 'PostgreSQL ক্যাশিং, Matrix-spec ডিসকভারি, এবং অপারেশনাল এন্ডপয়েন্ট সহ Go সার্ভিস।',
    learnMore: 'আরও জানুন',
    viewAPI: 'API দেখুন',
    github: 'GitHub',
    matrixFamilyBadge: 'Matrix Family',
    hushmeBadge: 'HushMe',
  },

  status: {
    online: 'ইনফ্রাস্ট্রাকচার অনলাইন',
  },

  about: {
    title: 'MXKeys কী?',
    description: 'MXKeys হল একটি Matrix ফেডারেশন ট্রাস্ট ইনফ্রাস্ট্রাকচার যা Matrix সার্ভারগুলিকে পরিচয় যাচাই করতে, কী পরিবর্তন ট্র্যাক করতে, অসঙ্গতি সনাক্ত করতে এবং ট্রাস্ট নীতি প্রয়োগ করতে সাহায্য করে।',
    
    problem: {
      title: 'সমস্যা',
      description: 'Matrix ফেডারেশন সীমিত দৃশ্যমানতা সহ সার্ভার কী-এর উপর নির্ভর করে। কী রোটেশন ট্র্যাক করা কঠিন, আপোষকৃত সার্ভার সনাক্ত করা কঠিন, এবং কী পরিবর্তনের জন্য কোনো অডিট ট্রেইল নেই। বিশ্বাস অন্তর্নিহিত।',
    },
    
    solution: {
      title: 'সমাধান',
      description: 'MXKeys পার্সপেক্টিভ স্বাক্ষর সহ কী যাচাই, Merkle প্রমাণ সহ হ্যাশ-চেইনড স্বচ্ছতা লগ, অসঙ্গতি সনাক্তকরণ, কনফিগারযোগ্য ট্রাস্ট নীতি, এবং প্রমাণীকৃত ক্লাস্টার মোড প্রদান করে।',
    },
  },

  features: {
    title: 'বৈশিষ্ট্যসমূহ',
    description: 'Matrix ফেডারেশনের জন্য কী যাচাই সক্ষমতা।',
    
    caching: {
      title: 'কী ক্যাশিং',
      description: 'যাচাইকৃত কী PostgreSQL-এ সংরক্ষণ করে। উৎস সার্ভারে বিলম্ব ও লোড কমায়।',
    },
    verification: {
      title: 'স্বাক্ষর যাচাই',
      description: 'ক্যাশিং-এর আগে সার্ভার স্বাক্ষরের বিপরীতে সমস্ত প্রাপ্ত কী যাচাই করে।',
    },
    perspective: {
      title: 'পার্সপেক্টিভ সাইনিং',
      description: 'যাচাইকৃত কী-তে একটি নোটারি সহ-স্বাক্ষর (ed25519:mxkeys) যোগ করে — একটি স্বাধীন প্রত্যয়ন।',
    },
    discovery: {
      title: 'সার্ভার ডিসকভারি',
      description: 'MXKeys কী-নোটারি পরিসরে .well-known ডেলিগেশন, SRV রেকর্ড (_matrix-fed._tcp), IP লিটারেল, এবং পোর্ট ফলব্যাকের জন্য Matrix ডিসকভারি সমর্থন।',
    },
    fallback: {
      title: 'ফলব্যাক সমর্থন',
      description: 'সরাসরি ফেচ ব্যর্থ হলে, MXKeys একটি স্পষ্ট অপারেশনাল ট্রাস্ট পথ হিসেবে কনফিগারকৃত ফলব্যাক নোটারি থেকে কোয়েরি করতে পারে।',
    },
    performance: {
      title: 'উচ্চ কর্মক্ষমতা',
      description: 'Go-তে লেখা। মেমোরি ক্যাশিং, কানেকশন পুলিং, দক্ষ ক্লিনআপ, এবং একক-বাইনারি ডিপ্লয়মেন্ট।',
    },
    opensource: {
      title: 'ওপেন সোর্স',
      description: 'নিরীক্ষণযোগ্য কোড। কোনো লুকানো লজিক নেই, কোনো মালিকানাধীন নির্ভরতা নেই।',
    },
  },

  howItWorks: {
    title: 'এটি কিভাবে কাজ করে',
    description: 'কী যাচাই প্রবাহ।',
    
    steps: {
      request: {
        title: '1. অনুরোধ',
        description: 'একটি Matrix সার্ভার POST /_matrix/key/v2/query-এর মাধ্যমে অন্য সার্ভারের কী-এর জন্য MXKeys-এ কোয়েরি করে',
      },
      cache: {
        title: '2. ক্যাশ চেক',
        description: 'MXKeys প্রথমে মেমোরি ক্যাশ, তারপর PostgreSQL পরীক্ষা করে। বৈধ ক্যাশড কী থাকলে — তৎক্ষণাৎ ফেরত দেয়।',
      },
      fetch: {
        title: '3. সার্ভার ডিসকভারি',
        description: 'ক্যাশ মিসে, MXKeys .well-known ডেলিগেশন, SRV রেকর্ড, এবং পোর্ট ফলব্যাক ব্যবহার করে লক্ষ্য সার্ভার রিজলভ করে — তারপর /_matrix/key/v2/server-এর মাধ্যমে কী ফেচ করে',
      },
      verify: {
        title: '4. যাচাই',
        description: 'MXKeys Ed25519 ব্যবহার করে সার্ভারের সেলফ-সিগনেচার যাচাই করে। অবৈধ স্বাক্ষর প্রত্যাখ্যাত হয়।',
      },
      sign: {
        title: '5. সহ-স্বাক্ষর',
        description: 'MXKeys তার পার্সপেক্টিভ স্বাক্ষর (ed25519:mxkeys) যোগ করে — প্রমাণ করে যে এটি কী যাচাই করেছে।',
      },
      respond: {
        title: '6. প্রতিক্রিয়া',
        description: 'মূল এবং নোটারি উভয় স্বাক্ষর সহ কী অনুরোধকারী সার্ভারে ফেরত দেওয়া হয়।',
      },
    },
  },

  api: {
    title: 'API এন্ডপয়েন্ট',
    description: 'MXKeys Matrix Key Server API এবং অপারেশনাল প্রোব বাস্তবায়ন করে।',
    
    serverKeys: {
      title: 'GET /_matrix/key/v2/server',
      description: 'MXKeys পাবলিক কী ফেরত দেয়। স্বাক্ষর যাচাইয়ের জন্য ব্যবহৃত।',
    },
    serverKeyByID: {
      title: 'GET /_matrix/key/v2/server/{keyID}',
      description: 'কী ID দ্বারা একটি নির্দিষ্ট MXKeys কী ফেরত দেয়। কী অনুপস্থিত থাকলে M_NOT_FOUND সহ প্রতিক্রিয়া দেয়।',
    },
    query: {
      title: 'POST /_matrix/key/v2/query',
      description: 'প্রধান নোটারি এন্ডপয়েন্ট। Matrix সার্ভারের কী কোয়েরি করে এবং MXKeys সহ-স্বাক্ষর সহ যাচাইকৃত কী ফেরত দেয়।',
    },
    version: {
      title: 'GET /_matrix/federation/v1/version',
      description: 'সার্ভার সংস্করণ তথ্য।',
    },
    health: {
      title: 'GET /_mxkeys/health',
      description: 'হেলথ এন্ডপয়েন্ট। সার্ভিস হেলথ মেটাডেটা ফেরত দেয়।',
    },
    ready: {
      title: 'GET /_mxkeys/ready',
      description: 'রেডিনেস এন্ডপয়েন্ট। DB সংযোগ এবং সক্রিয় সাইনিং কী যাচাই করে।',
    },
    live: {
      title: 'GET /_mxkeys/live',
      description: 'লাইভনেস প্রোব এন্ডপয়েন্ট। প্রসেস অ্যালাইভ স্টেট ফেরত দেয়।',
    },
    status: {
      title: 'GET /_mxkeys/status',
      description: 'বিস্তারিত সার্ভিস স্ট্যাটাস: আপটাইম, ক্যাশ মেট্রিক্স, ডেটাবেস স্ট্যাটস, এবং ঐচ্ছিক এন্টারপ্রাইজ ফিচার স্ট্যাটাস।',
    },
    metrics: {
      title: 'GET /_mxkeys/metrics',
      description: 'সার্ভিস এবং রানটাইম টেলিমেট্রির জন্য Prometheus মেট্রিক্স এক্সপোজিশন।',
    },
    errorsTitle: 'ত্রুটি মডেল',
    errorsDescription: 'অনুরোধ বৈধতা এবং অপব্যবহার নিয়ন্ত্রণ Matrix-সামঞ্জস্যপূর্ণ ত্রুটি কোড ব্যবহার করে: M_BAD_JSON, M_INVALID_PARAM, M_NOT_FOUND, M_TOO_LARGE, এবং M_LIMIT_EXCEEDED।',
    protectedTitle: 'সুরক্ষিত অপারেশনাল রুট',
    protectedDescription: 'স্বচ্ছতা, বিশ্লেষণ, ক্লাস্টার, এবং নীতি রুটগুলির জন্য এন্টারপ্রাইজ অ্যাক্সেস টোকেন প্রয়োজন এবং স্থিতিশীল পাবলিক ফেডারেশন API থেকে আলাদাভাবে নথিভুক্ত।',
  },

  integration: {
    title: 'ইন্টিগ্রেশন',
    description: 'আপনার Matrix সার্ভারকে MXKeys-কে বিশ্বস্ত কী সার্ভার হিসেবে ব্যবহার করতে কনফিগার করুন।',
    
    synapse: 'Synapse কনফিগারেশন',
    mxcore: 'MXCore কনফিগারেশন',
  },

  ecosystem: {
    title: 'Matrix Family-এর অংশ',
    description: 'MXKeys Matrix Family Inc. দ্বারা তৈরি। সমস্ত Matrix সার্ভারের জন্য উপলব্ধ।',
    
    matrixFamily: {
      title: 'Matrix Family',
      description: 'ইকোসিস্টেম হাব',
    },
    hushme: {
      title: 'HushMe',
      description: 'Matrix ক্লায়েন্ট',
    },
    hushmeStore: {
      title: 'HushMe Store',
      description: 'MFOS অ্যাপস',
    },
    mxcore: {
      title: 'MXCore',
      description: 'Matrix হোমসার্ভার',
    },
    mfos: {
      title: 'MFOS',
      description: 'ডেভেলপার প্ল্যাটফর্ম',
    },
  },

  footer: {
    ecosystem: 'ইকোসিস্টেম',
    resources: 'রিসোর্স',
    contact: 'যোগাযোগ',
    
    matrixFamily: 'Matrix Family',
    hushme: 'HushMe Client',
    hushmeStore: 'HushMe Store',
    mxcore: 'MXCore',
    mfos: 'MFOS Docs',
    hushmeWeb: 'HushMe Web',
    appsGateway: 'Apps Gateway',
    
    architecture: 'আর্কিটেকচার',
    apiReference: 'API রেফারেন্স',
    
    support: '@support',
    developer: '@dev',
    devChat: '#dev',
    
    protocol: 'প্রোটোকল',
    matrixSpec: 'Matrix Spec',
    hushmeSpace: 'HushMe Space',

    copyrightPrefix: '© 2026 Matrix Family Inc. সর্বস্বত্ব সংরক্ষিত। ',
    copyrightLink: 'Matrix Family',
    copyrightSuffix: ' ইকোসিস্টেমের অংশ।',
    tagline: 'Matrix ফেডারেশনের জন্য Key Notary।',
  },
};
