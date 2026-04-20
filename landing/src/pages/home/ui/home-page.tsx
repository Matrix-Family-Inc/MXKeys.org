/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Maintainer: Brabus
 * Contact: dev@matrix.family
 * Date: Mon Apr 20 2026 UTC
 * Status: Created
 */

import { AboutSection } from '../../../widgets/about';
import { ApiSection } from '../../../widgets/api';
import { EcosystemSection } from '../../../widgets/ecosystem';
import { FeaturesSection } from '../../../widgets/features';
import { HeroSection } from '../../../widgets/hero';
import { HowItWorksSection } from '../../../widgets/how-it-works';
import { IntegrationSection } from '../../../widgets/integration';
import { LandingFooter } from '../../../widgets/landing-footer';
import { LandingHeader } from '../../../widgets/landing-header';

/**
 * HomePage composes every marketing widget in the canonical order the
 * operator's stakeholders reviewed. Adding a new widget means adding it
 * here and registering its barrel in widgets/<name>/index.ts.
 */
export function HomePage() {
  return (
    <div id="home" className="min-h-screen">
      <LandingHeader />
      <main>
        <HeroSection />
        <AboutSection />
        <FeaturesSection />
        <HowItWorksSection />
        <ApiSection />
        <IntegrationSection />
        <EcosystemSection />
      </main>
      <LandingFooter />
    </div>
  );
}
