/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

import { useTranslation } from 'react-i18next';

const apiEndpoints = [
  { method: 'GET', path: '/_matrix/key/v2/server', i18nKey: 'api.serverKeys.description' },
  { method: 'GET', path: '/_matrix/key/v2/server/{keyID}', i18nKey: 'api.serverKeyByID.description' },
  { method: 'POST', path: '/_matrix/key/v2/query', i18nKey: 'api.query.description' },
  { method: 'GET', path: '/_matrix/federation/v1/version', i18nKey: 'api.version.description' },
  { method: 'GET', path: '/_mxkeys/health', i18nKey: 'api.health.description' },
  { method: 'GET', path: '/_mxkeys/ready', i18nKey: 'api.ready.description' },
  { method: 'GET', path: '/_mxkeys/live', i18nKey: 'api.live.description' },
  { method: 'GET', path: '/_mxkeys/status', i18nKey: 'api.status.description' },
  { method: 'GET', path: '/_mxkeys/metrics', i18nKey: 'api.metrics.description' },
];

export function ApiSection() {
  const { t } = useTranslation();

  return (
    <section id="api" className="py-20">
      <div className="max-w-7xl mx-auto px-6">
        <h2 className="text-3xl font-bold text-center text-text mb-4">
          {t('api.title')}
        </h2>
        <p className="text-lg text-text-secondary text-center max-w-2xl mx-auto mb-12">
          {t('api.description')}
        </p>

        <div className="max-w-3xl mx-auto space-y-6">
          {apiEndpoints.map((endpoint) => (
            <div key={`${endpoint.method}-${endpoint.path}`} className="card">
              <div className="flex items-center gap-3 mb-3">
                <span className={endpoint.method === 'POST' ? 'method-post' : 'method-get'}>
                  {endpoint.method}
                </span>
                <code className="text-text font-mono text-sm">{endpoint.path}</code>
              </div>
              <p className="text-text-secondary text-sm">
                {t(endpoint.i18nKey)}
              </p>
            </div>
          ))}

          <div className="card">
            <h3 className="text-base font-semibold text-text mb-2">
              {t('api.errorsTitle')}
            </h3>
            <p className="text-text-secondary text-sm">
              {t('api.errorsDescription')}
            </p>
          </div>

          <div className="card">
            <h3 className="text-base font-semibold text-text mb-2">
              {t('api.protectedTitle')}
            </h3>
            <p className="text-text-secondary text-sm mb-3">
              {t('api.protectedDescription')}
            </p>
            <code className="block text-text font-mono text-sm">
              /_mxkeys/transparency/*, /_mxkeys/analytics/*, /_mxkeys/cluster/*, /_mxkeys/policy/*
            </code>
          </div>
        </div>
      </div>
    </section>
  );
}
