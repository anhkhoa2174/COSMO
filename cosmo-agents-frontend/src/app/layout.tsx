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
  title: 'Cosmo Agents',
  description: 'Convert marketing leads into revenue with an AI workforce',
};

export default function RootLayout({ children }: React.PropsWithChildren) {
  return (
    <html lang="en">
      <head>
        <link rel="shortcut icon" href="/favicon.svg" />
        <meta
          name="viewport"
          content="minimum-scale=1, initial-scale=1, width=device-width, user-scalable=no"
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
