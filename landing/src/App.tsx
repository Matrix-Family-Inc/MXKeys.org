/*
 * Project: MXKeys
 * Company: Matrix Family Inc. (https://matrix.family)
 * Owner: Matrix Family Inc.
 * Maintainer: Brabus
 * Role: Lead Architect
 * Contact: dev@matrix.family
 * Support: support@matrix.family
 * Matrix: @support:matrix.family
 * Date: Tue Jan 27 2026 UTC
 * Status: Updated
 */

import { LandingHeader } from './components/LandingHeader';
import { HeroSection } from './components/HeroSection';
import { AboutSection } from './components/AboutSection';
import { FeaturesSection } from './components/FeaturesSection';
import { HowItWorksSection } from './components/HowItWorksSection';
import { ApiSection } from './components/ApiSection';
import { IntegrationSection } from './components/IntegrationSection';
import { EcosystemSection } from './components/EcosystemSection';
import { LandingFooter } from './components/LandingFooter';

function App() {
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

export default App;
