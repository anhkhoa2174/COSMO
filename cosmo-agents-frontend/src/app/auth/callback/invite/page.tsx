'use client';

import { Card, CardContent } from '@/components/ui/card';
import { memberRedirectUri } from '@/helpers/env';
import AuthApi from '@/network/client/auth';
import { cookieService } from '@/services/cookieService';
import { authService } from '@/services/authService';
import { setToken } from '@/helpers/client/cookies';
import { useUpdateUserCache } from '@/hooks/use-user';
import Image from 'next/image';
import { useSearchParams } from 'next/navigation';
import { useEffect } from 'react';
import { toast } from 'sonner';

export default function InviteCallback() {
  const searchParams = useSearchParams();
  const { mutateAsync: updateUserCache } = useUpdateUserCache();

  useEffect(() => {
    if (!searchParams) return;

    const execute = async () => {
      try {
        const params = Object.fromEntries(searchParams.entries());
        const credentials = {
          ...params,
          redirect_uri: memberRedirectUri,
        };
        const { data } = await AuthApi.processInviteCallback(credentials);
        if (typeof data === 'string') {
          window.location.href = data;
        } else {
          // Set tokens in cookies
          await cookieService.setTokens({
            accessToken: data.access_token,
            refreshToken: data.refresh_token,
            expiresIn: data.expires_in,
          });

          // Also set agent_access_token (unencrypted) for Chrome extension
          setToken({ agent_access_token: data.access_token });

          // Fetch and update user data
          const { data: user } = await authService.getMe();
          await updateUserCache(user);

          window.location.href = '/campaigns';
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
