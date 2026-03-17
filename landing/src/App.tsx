/*
 * Project: MXKeys
 * Company: Matrix.Family Inc. - Delaware C-Corp
 * Dev: Brabus
 * Date: Tue Jan 27 2026 UTC
 * Status: Updated - Unified URLs and Footer
 * Contact: @support:matrix.family
 */

import { useTranslation } from 'react-i18next';
import {
  ArrowRight,
  Shield,
  Globe,
  Code,
  Key,
  CheckCircle,
  Database,
  Zap,
  Lock,
  Server,
  ExternalLink,
  MessageSquare,
  BookOpen,
  Store,
  Github,
} from 'lucide-react';
import { Logo } from './components/Logo';
import { LanguageSwitcher } from './components/LanguageSwitcher';
import { URLS, EXTERNAL, MATRIX_CONTACTS, getLinkProps } from './config/urls';

function App() {
  const { t } = useTranslation();

  const features = [
    { key: 'caching', icon: Database },
    { key: 'verification', icon: CheckCircle },
    { key: 'perspective', icon: Shield },
    { key: 'discovery', icon: Globe },
    { key: 'fallback', icon: Server },
    { key: 'performance', icon: Zap },
    { key: 'opensource', icon: Code },
  ];

  const steps = [
    { key: 'request', icon: Server },
    { key: 'cache', icon: Database },
    { key: 'fetch', icon: Globe },
    { key: 'verify', icon: CheckCircle },
    { key: 'sign', icon: Key },
    { key: 'respond', icon: ArrowRight },
  ];

  const ecosystemItems = [
    { key: 'matrixFamily', icon: Globe, href: URLS.matrixFamily },
    { key: 'hushme', icon: MessageSquare, href: URLS.hushmeApp },
    { key: 'hushmeStore', icon: Store, href: URLS.hushmeStore },
    { key: 'mxcore', icon: Server, href: URLS.mxcore },
    { key: 'mfos', icon: BookOpen, href: URLS.mfos },
  ];

  return (
    <div className="min-h-screen">
      {/* Header */}
      <header className="sticky top-0 z-50 h-16 border-b border-[var(--color-border)] bg-[var(--color-bg)]/95 backdrop-blur-sm">
        <div className="max-w-7xl mx-auto h-full px-6 flex items-center justify-between">
          <a href="/" className="flex items-center gap-3">
            <Logo size={32} />
            <span className="text-lg font-semibold text-[var(--color-text)]">MXKeys</span>
          </a>
          <nav className="hidden md:flex items-center gap-1">
            <a href="#about" className="nav-link">{t('nav.about')}</a>
            <a href="#how-it-works" className="nav-link">{t('nav.howItWorks')}</a>
            <a href="#api" className="nav-link">{t('nav.api')}</a>
            <a href="#ecosystem" className="nav-link">{t('nav.ecosystem')}</a>
          </nav>
          <div className="flex items-center gap-2">
            <a
              href={EXTERNAL.github}
              {...getLinkProps(EXTERNAL.github)}
              aria-label="MXKeys GitHub repository"
              className="inline-flex h-9 w-9 items-center justify-center rounded-lg border border-[var(--color-border)] text-[var(--color-text-secondary)] hover:text-[var(--color-primary)] hover:border-[var(--color-primary)]/40 transition-colors"
            >
              <Github size={18} />
            </a>
            <LanguageSwitcher />
          </div>
        </div>
      </header>

      <main>
        {/* Hero */}
        <section className="relative overflow-hidden">
          <div className="absolute inset-0 opacity-[0.03]">
            <div
              className="absolute inset-0"
              style={{
                backgroundImage: `repeating-linear-gradient(
                  -15deg,
                  transparent,
                  transparent 70px,
                  rgba(61, 153, 112, 0.15) 70px,
                  rgba(61, 153, 112, 0.15) 71px
                )`,
              }}
            />
          </div>

          <div className="relative max-w-7xl mx-auto px-6 py-24 md:py-32">
            <div className="text-center">
              <div className="flex justify-center mb-8">
                <Logo size={120} animated />
              </div>

              <h1 className="text-5xl md:text-6xl font-bold text-[var(--color-text)] mb-4">
                {t('hero.title')}
              </h1>
              <p className="text-2xl md:text-3xl text-[var(--color-primary)] font-medium mb-4">
                {t('hero.subtitle')}
              </p>
              <p className="text-lg text-[var(--color-text-muted)] font-mono mb-6">
                {t('hero.tagline')}
              </p>

              <p className="text-lg text-[var(--color-text-secondary)] max-w-3xl mx-auto mb-4">
                {t('hero.description')}
              </p>

              <p className="text-sm text-[var(--color-text-muted)] font-medium mb-8">
                {t('hero.trust')}
              </p>

              <div className="flex flex-wrap items-center justify-center gap-3 mb-10">
                <span className="badge badge-primary flex items-center gap-2">
                  <span className="w-2 h-2 bg-[var(--color-primary)] rounded-full animate-pulse" />
                  {t('status.online')}
                </span>
                <a 
                  href={URLS.matrixFamily}
                  {...getLinkProps(URLS.matrixFamily)}
                  className="badge badge-accent hover:opacity-80 transition-opacity"
                >
                  Matrix.Family
                </a>
                <a 
                  href={URLS.hushmeApp}
                  {...getLinkProps(URLS.hushmeApp)}
                  className="badge badge-accent hover:opacity-80 transition-opacity"
                >
                  HushMe
                </a>
              </div>

              <div className="flex flex-col sm:flex-row items-center justify-center gap-4">
                <a href="#about" className="btn btn-primary">
                  {t('hero.learnMore')}
                  <ArrowRight size={18} />
                </a>
                <a href="#api" className="btn btn-outline">
                  {t('hero.viewAPI')}
                </a>
                <a 
                  href={EXTERNAL.github}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="btn btn-outline"
                >
                  <Github size={18} />
                  GitHub
                </a>
              </div>
            </div>
          </div>
        </section>

        {/* About */}
        <section id="about" className="py-20 bg-[var(--color-bg-surface)]">
          <div className="max-w-7xl mx-auto px-6">
            <div className="max-w-3xl mx-auto text-center mb-12">
              <h2 className="text-3xl font-bold text-[var(--color-text)] mb-4">
                {t('about.title')}
              </h2>
              <p className="text-lg text-[var(--color-text-secondary)]">
                {t('about.description')}
              </p>
            </div>

            <div className="grid md:grid-cols-2 gap-8 max-w-4xl mx-auto">
              <div className="card">
                <div className="w-12 h-12 rounded-lg bg-[rgba(244,67,54,0.15)] flex items-center justify-center mb-4">
                  <Lock size={24} className="text-[#f44336]" />
                </div>
                <h3 className="text-xl font-semibold text-[var(--color-text)] mb-3">
                  {t('about.problem.title')}
                </h3>
                <p className="text-[var(--color-text-secondary)]">
                  {t('about.problem.description')}
                </p>
              </div>

              <div className="card">
                <div className="w-12 h-12 rounded-lg bg-[var(--color-primary-muted)] flex items-center justify-center mb-4">
                  <Key size={24} className="text-[var(--color-primary)]" />
                </div>
                <h3 className="text-xl font-semibold text-[var(--color-text)] mb-3">
                  {t('about.solution.title')}
                </h3>
                <p className="text-[var(--color-text-secondary)]">
                  {t('about.solution.description')}
                </p>
              </div>
            </div>
          </div>
        </section>

        {/* Features */}
        <section className="py-20">
          <div className="max-w-7xl mx-auto px-6">
            <h2 className="text-3xl font-bold text-center text-[var(--color-text)] mb-4">
              {t('features.title')}
            </h2>
            <p className="text-lg text-[var(--color-text-secondary)] text-center max-w-2xl mx-auto mb-12">
              {t('features.description')}
            </p>

            <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
              {features.map((item) => (
                <div key={item.key} className="card">
                  <div className="w-12 h-12 rounded-lg bg-[var(--color-primary-muted)] flex items-center justify-center mb-4">
                    <item.icon size={24} className="text-[var(--color-primary)]" />
                  </div>
                  <h3 className="text-lg font-semibold text-[var(--color-text)] mb-2">
                    {t(`features.${item.key}.title`)}
                  </h3>
                  <p className="text-sm text-[var(--color-text-secondary)]">
                    {t(`features.${item.key}.description`)}
                  </p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* How It Works */}
        <section id="how-it-works" className="py-20 bg-[var(--color-bg-surface)]">
          <div className="max-w-7xl mx-auto px-6">
            <h2 className="text-3xl font-bold text-center text-[var(--color-text)] mb-4">
              {t('howItWorks.title')}
            </h2>
            <p className="text-lg text-[var(--color-text-secondary)] text-center max-w-2xl mx-auto mb-12">
              {t('howItWorks.description')}
            </p>

            <div className="grid md:grid-cols-2 lg:grid-cols-3 gap-6">
              {steps.map((step, index) => (
                <div key={step.key} className="card relative">
                  <div className="absolute top-4 right-4">
                    <span className="step-number">{index + 1}</span>
                  </div>
                  <div className="w-10 h-10 rounded-lg bg-[var(--color-primary-muted)] flex items-center justify-center mb-4">
                    <step.icon size={20} className="text-[var(--color-primary)]" />
                  </div>
                  <h4 className="text-lg font-semibold text-[var(--color-text)] mb-2">
                    {t(`howItWorks.steps.${step.key}.title`)}
                  </h4>
                  <p className="text-sm text-[var(--color-text-secondary)]">
                    {t(`howItWorks.steps.${step.key}.description`)}
                  </p>
                </div>
              ))}
            </div>
          </div>
        </section>

        {/* API */}
        <section id="api" className="py-20">
          <div className="max-w-7xl mx-auto px-6">
            <h2 className="text-3xl font-bold text-center text-[var(--color-text)] mb-4">
              {t('api.title')}
            </h2>
            <p className="text-lg text-[var(--color-text-secondary)] text-center max-w-2xl mx-auto mb-12">
              {t('api.description')}
            </p>

            <div className="max-w-3xl mx-auto space-y-6">
              <div className="card">
                <div className="flex items-center gap-3 mb-3">
                  <span className="method-get">GET</span>
                  <code className="text-[var(--color-text)] font-mono text-sm">/_matrix/key/v2/server</code>
                </div>
                <p className="text-[var(--color-text-secondary)] text-sm">
                  {t('api.serverKeys.description')}
                </p>
              </div>

              <div className="card">
                <div className="flex items-center gap-3 mb-3">
                  <span className="method-get">GET</span>
                  <code className="text-[var(--color-text)] font-mono text-sm">/_matrix/key/v2/server/{"{keyID}"}</code>
                </div>
                <p className="text-[var(--color-text-secondary)] text-sm">
                  {t('api.serverKeyByID.description')}
                </p>
              </div>

              <div className="card">
                <div className="flex items-center gap-3 mb-3">
                  <span className="method-post">POST</span>
                  <code className="text-[var(--color-text)] font-mono text-sm">/_matrix/key/v2/query</code>
                </div>
                <p className="text-[var(--color-text-secondary)] text-sm">
                  {t('api.query.description')}
                </p>
              </div>

              <div className="card">
                <div className="flex items-center gap-3 mb-3">
                  <span className="method-get">GET</span>
                  <code className="text-[var(--color-text)] font-mono text-sm">/_matrix/federation/v1/version</code>
                </div>
                <p className="text-[var(--color-text-secondary)] text-sm">
                  {t('api.version.description')}
                </p>
              </div>

              <div className="card">
                <div className="flex items-center gap-3 mb-3">
                  <span className="method-get">GET</span>
                  <code className="text-[var(--color-text)] font-mono text-sm">/_mxkeys/health</code>
                </div>
                <p className="text-[var(--color-text-secondary)] text-sm">
                  {t('api.health.description')}
                </p>
              </div>

              <div className="card">
                <div className="flex items-center gap-3 mb-3">
                  <span className="method-get">GET</span>
                  <code className="text-[var(--color-text)] font-mono text-sm">/_mxkeys/ready</code>
                </div>
                <p className="text-[var(--color-text-secondary)] text-sm">
                  {t('api.ready.description')}
                </p>
              </div>

              <div className="card">
                <div className="flex items-center gap-3 mb-3">
                  <span className="method-get">GET</span>
                  <code className="text-[var(--color-text)] font-mono text-sm">/_mxkeys/live</code>
                </div>
                <p className="text-[var(--color-text-secondary)] text-sm">
                  {t('api.live.description')}
                </p>
              </div>

              <div className="card">
                <div className="flex items-center gap-3 mb-3">
                  <span className="method-get">GET</span>
                  <code className="text-[var(--color-text)] font-mono text-sm">/_mxkeys/status</code>
                </div>
                <p className="text-[var(--color-text-secondary)] text-sm">
                  {t('api.status.description')}
                </p>
              </div>

              <div className="card">
                <h4 className="text-base font-semibold text-[var(--color-text)] mb-2">
                  {t('api.errorsTitle')}
                </h4>
                <p className="text-[var(--color-text-secondary)] text-sm">
                  {t('api.errorsDescription')}
                </p>
              </div>
            </div>
          </div>
        </section>

        {/* Integration */}
        <section className="py-20 bg-[var(--color-bg-surface)]">
          <div className="max-w-7xl mx-auto px-6">
            <h2 className="text-3xl font-bold text-center text-[var(--color-text)] mb-4">
              {t('integration.title')}
            </h2>
            <p className="text-lg text-[var(--color-text-secondary)] text-center max-w-2xl mx-auto mb-12">
              {t('integration.description')}
            </p>

            <div className="grid md:grid-cols-2 gap-6 max-w-4xl mx-auto">
              <div className="card">
                <h4 className="font-semibold text-[var(--color-text)] mb-4">{t('integration.synapse')}</h4>
                <div className="code-block">
                  <div className="comment"># homeserver.yaml</div>
                  <div><span className="keyword">trusted_key_servers</span>:</div>
                  <div className="pl-4">- <span className="keyword">server_name</span>: <span className="string">"mxkeys.org"</span></div>
                </div>
              </div>

              <div className="card">
                <h4 className="font-semibold text-[var(--color-text)] mb-4">{t('integration.mxcore')}</h4>
                <div className="code-block">
                  <div className="comment"># config.yaml</div>
                  <div><span className="keyword">federation</span>:</div>
                  <div className="pl-4"><span className="keyword">trusted_key_servers</span>:</div>
                  <div className="pl-8">- <span className="string">"mxkeys.org"</span></div>
                </div>
              </div>
            </div>
          </div>
        </section>

        {/* Ecosystem */}
        <section id="ecosystem" className="py-20">
          <div className="max-w-7xl mx-auto px-6">
            <h2 className="text-3xl font-bold text-center text-[var(--color-text)] mb-4">
              {t('ecosystem.title')}
            </h2>
            <p className="text-lg text-[var(--color-text-secondary)] text-center max-w-2xl mx-auto mb-12">
              {t('ecosystem.description')}
            </p>

            <div className="grid md:grid-cols-2 lg:grid-cols-5 gap-6">
              {ecosystemItems.map((item) => (
                <a
                  key={item.key}
                  href={item.href}
                  {...getLinkProps(item.href)}
                  className="card card-interactive text-center group"
                >
                  <div className="w-12 h-12 rounded-lg bg-[var(--color-bg-hover)] flex items-center justify-center mx-auto mb-4 group-hover:bg-[var(--color-primary-muted)] transition-colors">
                    <item.icon size={24} className="text-[var(--color-text-secondary)] group-hover:text-[var(--color-primary)] transition-colors" />
                  </div>
                  <h4 className="text-lg font-semibold text-[var(--color-text)] mb-2 flex items-center justify-center gap-2">
                    {t(`ecosystem.${item.key}.title`)}
                    <ExternalLink size={14} className="text-[var(--color-text-muted)] opacity-0 group-hover:opacity-100 transition-opacity" />
                  </h4>
                  <p className="text-sm text-[var(--color-text-secondary)]">
                    {t(`ecosystem.${item.key}.description`)}
                  </p>
                </a>
              ))}
            </div>
          </div>
        </section>
      </main>

      {/* Footer */}
      <footer className="border-t border-[var(--color-border)] bg-[var(--color-bg)]">
        <div className="max-w-7xl mx-auto px-6 py-12">
          <div className="grid grid-cols-2 md:grid-cols-5 gap-8">
            {/* Brand */}
            <div className="col-span-2 md:col-span-1">
              <div className="flex items-center gap-2 mb-4">
                <Logo size={28} />
                <span className="font-semibold text-[var(--color-text)]">MXKeys</span>
              </div>
              <p className="text-sm text-[var(--color-text-secondary)]">
                {t('footer.tagline')}
              </p>
            </div>

            {/* Ecosystem */}
            <div>
              <h4 className="font-medium text-[var(--color-text)] mb-3">{t('footer.ecosystem')}</h4>
              <ul className="space-y-2 text-sm">
                <li><a href={URLS.matrixFamily} {...getLinkProps(URLS.matrixFamily)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.matrixFamily')}</a></li>
                <li><a href={URLS.hushmeApp} {...getLinkProps(URLS.hushmeApp)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.hushme')}</a></li>
                <li><a href={URLS.hushmeStore} {...getLinkProps(URLS.hushmeStore)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.hushmeStore')}</a></li>
                <li><a href={URLS.mxcore} {...getLinkProps(URLS.mxcore)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.mxcore')}</a></li>
                <li><a href={URLS.mfos} {...getLinkProps(URLS.mfos)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.mfos')}</a></li>
              </ul>
            </div>

            {/* Resources */}
            <div>
              <h4 className="font-medium text-[var(--color-text)] mb-3">{t('footer.resources')}</h4>
              <ul className="space-y-2 text-sm">
                <li><a href={URLS.hushmeOnline} {...getLinkProps(URLS.hushmeOnline)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.hushmeWeb')}</a></li>
                <li><a href={URLS.appsGateway} {...getLinkProps(URLS.appsGateway)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.appsGateway')}</a></li>
                <li><a href="#about" className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.architecture')}</a></li>
                <li><a href="#api" className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.apiReference')}</a></li>
              </ul>
            </div>

            {/* Community */}
            <div>
              <h4 className="font-medium text-[var(--color-text)] mb-3">{t('footer.contact')}</h4>
              <ul className="space-y-2 text-sm">
                <li><a href={MATRIX_CONTACTS.support.href} {...getLinkProps(MATRIX_CONTACTS.support.href)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.support')}</a></li>
                <li><a href={MATRIX_CONTACTS.developer.href} {...getLinkProps(MATRIX_CONTACTS.developer.href)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.developer')}</a></li>
                <li><a href={MATRIX_CONTACTS.devChat.href} {...getLinkProps(MATRIX_CONTACTS.devChat.href)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">#dev</a></li>
              </ul>
            </div>

            {/* Protocol */}
            <div>
              <h4 className="font-medium text-[var(--color-text)] mb-3">{t('footer.protocol')}</h4>
              <ul className="space-y-2 text-sm">
                <li><a href="https://spec.matrix.org/latest/server-server-api/#querying-keys-through-another-server" target="_blank" rel="noopener noreferrer" className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.matrixSpec')}</a></li>
                <li><a href={URLS.hushmeSpace} {...getLinkProps(URLS.hushmeSpace)} className="text-[var(--color-text-secondary)] hover:text-[var(--color-primary)]">{t('footer.hushmeSpace')}</a></li>
              </ul>
            </div>
          </div>

          <div className="mt-12 pt-8 border-t border-[var(--color-border)]">
            <p className="text-sm text-[var(--color-text-muted)] text-center">
              {t('footer.copyrightPrefix')}
              <a
                href={URLS.matrixFamily}
                {...getLinkProps(URLS.matrixFamily)}
                className="text-[var(--color-text-muted)] underline decoration-[var(--color-primary)]/50 underline-offset-2 hover:text-[var(--color-primary)]"
              >
                {t('footer.copyrightLink')}
              </a>
              {t('footer.copyrightSuffix')}
            </p>
          </div>
        </div>
      </footer>
    </div>
  );
}

export default App;
