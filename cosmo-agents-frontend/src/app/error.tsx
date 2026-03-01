'use client';

import Link from 'next/link';

import { AlertCircle } from 'lucide-react';
import { Alert, AlertDescription, AlertTitle } from '@/components/ui/alert';
import { MainButton } from '@/components/buttons/main-button';

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  return (
    <div className="ca-grid-line flex h-screen flex-col items-center justify-center">
      <div className="container">
        <div className="ml-8 flex flex-col gap-4">
          <h2 className="text-2xl font-bold uppercase">Something went wrong</h2>
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertTitle>{error.name}</AlertTitle>
            <AlertDescription>
              <p className="line-clamp-5 whitespace-pre-wrap">{error.stack}</p>
            </AlertDescription>
          </Alert>
          <div className="flex gap-4">
            <Link href="/" passHref>
              <MainButton text="Go home" variant="outline" />
            </Link>
            <MainButton onClick={reset} text="Reset" />
          </div>
        </div>
      </div>
    </div>
  );
}
