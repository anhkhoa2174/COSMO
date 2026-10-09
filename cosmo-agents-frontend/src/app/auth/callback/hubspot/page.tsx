'use client';

import Image from 'next/image';

import { useSearchParams } from 'next/navigation';
import { useEffect } from 'react';
import { toast } from 'sonner';

import { Card, CardContent } from '@/components/ui/card';
import { hubspotRedirectUri } from '@/helpers/env';
import { HubspotApi } from '@/network/client/hubspot';

export default function HubSpotAuthCallback() {
  const searchParams = useSearchParams();

  useEffect(() => {
    if (!searchParams) return;

    const execute = async () => {
      try {
        const params = Object.fromEntries(searchParams.entries());
        await HubspotApi.processCallback({
          code: params.code,
          redirect_uri: hubspotRedirectUri,
        });

        if (window.opener) {
          window.opener.postMessage({ type: 'HUBSPOT_AUTH' }, '*');
        }
      } catch (err: any) {
        toast.error(err.message);
      }
    };

    execute();
  }, [searchParams]);

  return (
    <div className="flex h-screen items-center justify-center bg-[#f7f7f7]">
      <Card>
        <CardContent className="flex h-[320px] w-[320px] flex-col items-center justify-center gap-4 pt-4">
          <Image src="/favicon.svg" alt="LogoImg" width={64} height={64} />
          <p className="text-2xl font-semibold">Authenticating...</p>
        </CardContent>
      </Card>
    </div>
  );
}
