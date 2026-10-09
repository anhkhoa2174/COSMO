import { Header, Product } from '@/components/LandingPage';
import { AnnouncementBar } from '@/components/LandingPage/announcement-bar';
import { TrustedBy } from '@/components/LandingPage/trusted-by';

import { UseCases } from '@/components/LandingPage/use-cases';
import { KeyFeatures } from '@/components/LandingPage/key-features';
import { HowItWorks } from '@/components/LandingPage/how-it-works';
import { Testimonials } from '@/components/LandingPage/testimonials';
import { Trust } from '@/components/LandingPage/trust';
import { Waitlist } from '@/components/LandingPage/waitlist';
import { Footer } from '@/components/LandingPage/footer';
import { AskCosmoPublic } from '@/components/LandingPage/ask-cosmo-public';

export default function Home() {
  return (
    <div className="relative">
      <AnnouncementBar />
      <Header />
      {/* The landmark lets a screen reader skip the banner and go straight to
          the content. Header, footer and the chat widget stay outside it: they
          are navigation and furniture, not the page's subject. */}
      <main>
        <Product />
        <TrustedBy />
        <UseCases />
        <KeyFeatures />
        <HowItWorks />
        <Testimonials />
        <Trust />
        <Waitlist />
      </main>
      <Footer />
      <AskCosmoPublic />
    </div>
  );
}
