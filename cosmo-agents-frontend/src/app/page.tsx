import { Header, Product } from '@/components/LandingPage';

import { UseCases } from '@/components/LandingPage/use-cases';
import { KeyFeatures } from '@/components/LandingPage/key-features';
import { HowItWorks } from '@/components/LandingPage/how-it-works';
import { Testimonials } from '@/components/LandingPage/testimonials';
import { Waitlist } from '@/components/LandingPage/waitlist';
import { Footer } from '@/components/LandingPage/footer';

export default function Home() {
  return (
    <div className="relative">
      <Header />
      <Product />
      <UseCases />
      <KeyFeatures />
      <HowItWorks />
      <Testimonials />
      <Waitlist />
      <Footer />
    </div>
  );
}
