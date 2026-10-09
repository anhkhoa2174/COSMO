'use client';

import { Card, CardContent } from '@/components/ui/card';
import { gmailRedirectUri } from '@/helpers/env';
import AuthApi from '@/network/client/auth';
import Image from 'next/image';
import { useSearchParams } from 'next/navigation';
import { useEffect, useRef, useState } from 'react';

export default function GmailAuthCallback() {
  const searchParams = useSearchParams();
  const [error, setError] = useState<string | null>(null);
  const [countdown, setCountdown] = useState(5);
  // An authorization code is single-use. This effect re-runs (Strict Mode, and
  // any render that hands back a new searchParams object), and each re-run
  // burned the code — so the retries always failed with invalid_grant and
  // masked whatever the first attempt actually hit.
  const exchangedRef = useRef(false);

  useEffect(() => {
    if (error) {
      const timer = setInterval(() => {
        setCountdown((prev) => {
          if (prev <= 1) {
            window.close();
            return 0;
          }
          return prev - 1;
        });
      }, 1000);

      return () => clearInterval(timer);
    }
  }, [error]);

  useEffect(() => {
    if (!searchParams) return;
    if (exchangedRef.current) return;
    exchangedRef.current = true;

    const execute = async () => {
      const timeout = setTimeout(() => {
        setError('Authentication timeout. Please try again.');
      }, 30000);

      try {
        const params = Object.fromEntries(searchParams.entries());
        const data = {
          ...params,
          redirect_uri: gmailRedirectUri,
        };
        await AuthApi.processGmailAuthCallback(data);

        if (
          window.opener &&
          window.opener.location.origin === window.location.origin
        ) {
          window.opener.postMessage(
            {
              type: 'GMAIL_AUTH',
              data,
              timestamp: Date.now(),
            },
            window.location.origin
          );
        } else {
          throw new Error('Authentication window communication failed');
        }
      } catch (err: any) {
        console.error('Gmail Auth Error:', err);
        const detail = err?.error?.detail || err?.detail;
        setError(
          detail
            ? `${err.message || 'Authentication failed'}: ${detail}`
            : err.message || 'Authentication failed. Please try again.'
        );
      } finally {
        clearTimeout(timeout);
      }
    };

    execute();
  }, [searchParams]);

  return (
    <div className="flex h-screen items-center justify-center bg-[#f7f7f7]">
      <Card>
        <CardContent className="flex h-[320px] w-[320px] flex-col items-center justify-center gap-4 pt-4">
          <Image src="/favicon.svg" alt="LogoImg" width={64} height={64} />
          {error ? (
            <>
              <p className="text-center text-base font-semibold text-destructive">
                {error}
              </p>
              <p className="text text-muted-foreground">
                Window will close in {countdown} seconds...
              </p>
            </>
          ) : (
            <p className="text-2xl font-semibold">Authenticating...</p>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
