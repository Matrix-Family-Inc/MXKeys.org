/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Fri 03 Apr 2026 UTC
 * Status: Created
 */

import { useTranslation } from 'react-i18next';

export function IntegrationSection() {
  const { t } = useTranslation();

  return (
    <section className="py-20 bg-bg-surface">
      <div className="max-w-7xl mx-auto px-6">
        <h2 className="text-3xl font-bold text-center text-text mb-4">
          {t('integration.title')}
        </h2>
        <p className="text-lg text-text-secondary text-center max-w-2xl mx-auto mb-12">
          {t('integration.description')}
        </p>

        <div className="grid md:grid-cols-2 gap-6 max-w-4xl mx-auto">
          <div className="card">
            <h3 className="font-semibold text-text mb-4">{t('integration.mxcore')}</h3>
            <div className="code-block">
              <div className="comment"># config.yaml</div>
              <div><span className="keyword">federation</span>:</div>
              <div className="pl-4"><span className="keyword">trusted_key_servers</span>:</div>
              <div className="pl-8">- <span className="string">&quot;notary.example.org&quot;</span></div>
            </div>
          </div>

          <div className="card">
            <h3 className="font-semibold text-text mb-4">{t('integration.synapse')}</h3>
            <div className="code-block">
              <div className="comment"># homeserver.yaml</div>
              <div><span className="keyword">trusted_key_servers</span>:</div>
              <div className="pl-4">- <span className="keyword">server_name</span>: <span className="string">&quot;notary.example.org&quot;</span></div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
