import { ColorSchemeScript, MantineProvider } from '@mantine/core';
import { Open_Sans } from 'next/font/google';
import React, { Suspense } from 'react';

import './globals.css';

import { Toaster } from '@/components/ui/sonner';
import theme from '@/helpers/theme';
import { ReactQueryProvider } from '@/providers/react-query-provider';

const openSans = Open_Sans({
  display: 'swap',
  variable: '--font-primary',
  subsets: ['latin'],
});

export const metadata = {
  title: 'COSMO',
  description: 'Convert marketing leads into revenue with an AI workforce',
};

export default function RootLayout({ children }: React.PropsWithChildren) {
  return (
    <html lang="en">
      <head>
        <link rel="shortcut icon" href="/favicon.svg" />
        <meta
          name="viewport"
          // user-scalable=no blocked pinch-zoom, which Lighthouse flags as an
          // accessibility failure and which anybody who needs to enlarge text
          // simply cannot work around. It buys nothing here: the layout is
          // responsive, so there is no mis-zoom to prevent.
          content="minimum-scale=1, initial-scale=1, width=device-width"
        />
        <ColorSchemeScript />
      </head>
      <body className={openSans.className}>
        <ReactQueryProvider>
          <MantineProvider theme={theme} forceColorScheme="light">
            <Suspense>{children}</Suspense>
            <Toaster richColors position="top-center" />
          </MantineProvider>
        </ReactQueryProvider>
      </body>
    </html>
  );
}
