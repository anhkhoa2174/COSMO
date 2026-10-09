'use client';

import { useState, type PropsWithChildren } from 'react';
import {
  QueryClient,
  QueryClientProvider,
  type QueryClientConfig,
} from '@tanstack/react-query';

export function ReactQueryProvider({ children }: PropsWithChildren) {
  const config: QueryClientConfig = {
    defaultOptions: {
      queries: {
        refetchOnWindowFocus: false,
        retry: false,
      },
    },
  };

  const [queryClient] = useState(() => new QueryClient(config));

  return (
    <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>
  );
}
