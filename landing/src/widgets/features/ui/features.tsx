/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

import { useTranslation } from 'react-i18next';
import {
  CheckCircle,
  Code,
  Database,
  FileCheck,
  Globe,
  Layers,
  Network,
  Server,
  Shield,
  Zap,
} from 'lucide-react';

const features = [
  { key: 'caching', icon: Database },
  { key: 'verification', icon: CheckCircle },
  { key: 'perspective', icon: Shield },
  { key: 'integrity', icon: FileCheck },
  { key: 'transparency', icon: Layers },
  { key: 'cluster', icon: Network },
  { key: 'discovery', icon: Globe },
  { key: 'fallback', icon: Server },
  { key: 'performance', icon: Zap },
  { key: 'opensource', icon: Code },
];

export function FeaturesSection() {
  const { t } = useTranslation();

  return (
    <section className="py-20">
      <div className="max-w-7xl mx-auto px-6">
        <h2 className="text-3xl font-bold text-center text-text mb-4">
          {t('features.title')}
        </h2>
        <p className="text-lg text-text-secondary text-center max-w-2xl mx-auto mb-12">
          {t('features.description')}
        </p>

        <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
          {features.map((item) => (
            <div key={item.key} className="card">
              <div className="w-12 h-12 rounded-lg bg-primary-muted flex items-center justify-center mb-4">
                <item.icon size={24} className="text-primary" aria-hidden="true" />
              </div>
              <h3 className="text-lg font-semibold text-text mb-2">
                {t(`features.${item.key}.title`)}
              </h3>
              <p className="text-sm text-text-secondary">
                {t(`features.${item.key}.description`)}
              </p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}
