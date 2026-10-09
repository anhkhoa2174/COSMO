import Header from '@/components/LandingPage/header';
import { Footer } from '@/components/LandingPage/footer';

/**
 * Shared shell for the public legal pages (/policy, /tos, /data-processing):
 * the same sticky navbar and footer as the landing page, so a visitor who
 * follows a footer link stays inside the same site rather than landing on a
 * bare document.
 */
export default function LegalLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="relative min-h-screen">
      <Header />
      {children}
      <Footer />
    </div>
  );
}
