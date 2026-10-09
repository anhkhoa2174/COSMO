'use client';

import Link from 'next/link';
import { usePathname } from 'next/navigation';

import { CosmoMark } from '@/components/nav/cosmo-mark';

const LEGAL_LINKS = [
  { href: '/policy', label: 'Trust & Data Policy' },
  { href: '/tos', label: 'Terms of Service' },
  { href: '/data-processing', label: 'Data Processing' },
];

export function Footer() {
  const pathname = usePathname();

  return (
    <footer className="bg-zinc-100 py-4">
      <div className="container mx-auto">
        <div className="flex flex-col items-center justify-center gap-2">
          <p className="text-center text-zinc-600">
            © 2026 Cosmo. An AI-native outreach system for B2B sales teams
          </p>
          <div className="flex flex-wrap items-center justify-center gap-4">
            <Link href="/" className="flex items-center gap-1.5">
              <CosmoMark size={20} />
              <span className="text-sm font-bold tracking-tight text-[#3C1988]">
                COSMO
              </span>
            </Link>
            {LEGAL_LINKS.map(({ href, label }) =>
              // The page you are already on is shown as plain text, so the
              // list reads as "where you are" rather than four look-alike links.
              pathname === href ? (
                <span
                  key={href}
                  aria-current="page"
                  className="font-medium text-foreground underline underline-offset-4"
                >
                  {label}
                </span>
              ) : (
                <Link
                  key={href}
                  href={href}
                  className="text-zinc-600 hover:underline"
                >
                  {label}
                </Link>
              )
            )}
          </div>
        </div>
      </div>
    </footer>
  );
}
