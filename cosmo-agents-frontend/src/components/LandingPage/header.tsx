'use client';

import Link from 'next/link';
// import Script from 'next/script';
import { useRouter } from 'next/navigation';
import { CosmoMark } from '@/components/nav/cosmo-mark';
import { MainButton } from '../buttons/main-button';
import { Menu } from 'lucide-react';
import { useState } from 'react';

const Header = () => {
  const [isOpen, setIsOpen] = useState(false);
  const router = useRouter();

  const sections = [
    { id: 'product', label: 'Product' },
    { id: 'key-features', label: 'Key features' },
    { id: 'how-it-works', label: 'How it works' },
    { id: 'use-cases', label: 'Use cases' },
    { id: 'testimonials', label: 'Testimonials' },
    { id: 'policy', label: 'Policy' },
  ];

  const scrollToSection =
    (id: string) => (e: React.MouseEvent<HTMLButtonElement>) => {
      e.preventDefault();
      const target = document.getElementById(id);

      if (!target) {
        // Not on the landing page — go there and let the browser jump to the section
        router.push(`/#${id}`);
        return;
      }

      if (target) {
        const headerOffset = 30;
        const elementPosition = target.getBoundingClientRect().top;
        const offsetPosition = elementPosition + window.scrollY - headerOffset;

        window.scrollTo({
          top: offsetPosition - 80,
          behavior: 'smooth',
        });
      }
    };

  return (
    <div className="container sticky top-4 z-50 mx-auto">
      <div className="space-y-2 rounded-xl border bg-background p-4 shadow-md">
        <div className="flex items-center justify-between gap-2">
          <Link href="/" className="flex items-center gap-2.5">
            <CosmoMark size={40} />
            <span className="text-2xl font-bold tracking-tight">COSMO</span>
          </Link>
          <div className="hidden items-center justify-center gap-8 sm:flex">
            {sections.map(({ id, label }) => (
              <button
                key={id}
                type="button"
                className="font-semibold text-muted-foreground hover:underline hover:underline-offset-4"
                onClick={scrollToSection(id)}
              >
                {label}
              </button>
            ))}
          </div>
          <RequestButton className="hidden sm:inline-flex" />
          <MainButton
            icon={Menu}
            size="icon"
            variant="ghost"
            className="sm:hidden"
            onClick={() => setIsOpen((prev) => !prev)}
          />
        </div>
        {isOpen && (
          <div className="flex flex-col items-center gap-4">
            {sections.map(({ id, label }) => (
              <button
                key={id}
                type="button"
                className="font-semibold text-muted-foreground hover:underline hover:underline-offset-4"
                onClick={scrollToSection(id)}
              >
                {label}
              </button>
            ))}
            <RequestButton />
          </div>
        )}
      </div>
    </div>
  );
};

export default Header;

// declare global {
//   interface Window {
//     Calendly: any;
//   }
// }

const RequestButton = ({ className }: { className?: string }) => {
  // const handleClick = () => {
  //   window.Calendly.initPopupWidget({
  //     url: 'https://calendly.com/ngoc-cosmoagents/30min',
  //   });
  // };

  const router = useRouter();

  return (
    <>
      <MainButton
        text="Try It Now"
        onClick={() => router.push('/auth/login')}
        className={`w-40 ${className}`}
      />
      {/* <Script
        src="https://assets.calendly.com/assets/external/widget.js"
        async
        strategy="afterInteractive"
      />
      <link href="https://assets.calendly.com/assets/external/widget.css" rel="stylesheet" /> */}
    </>
  );
};
