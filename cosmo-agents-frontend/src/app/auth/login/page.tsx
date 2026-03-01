'use client';

import { IconBrandGoogle } from '@/assets/icons';
import { googleRedirectUri } from '@/helpers/env';
import Logo from '@/images/logo.png';
import Background from '@/images/signin.png';
import AuthApi from '@/network/client/auth';
import { Button } from '@/components/ui/button';
import Image from 'next/image';
import { useSearchParams } from 'next/navigation';
import { useState } from 'react';
import { toast } from 'sonner';

export default function AuthLoginPage() {
  const searchParams = useSearchParams();
  const nextUri = searchParams.get('next');

  const [isAuthenticating, setIsAuthenticating] = useState(false);

  if (isAuthenticating) {
    return (
      <div className="flex h-screen flex-col items-center justify-center">
        <span className="loader" />
      </div>
    );
  }

  const getGoogleAuth = async () => {
    try {
      setIsAuthenticating(true);
      const response = await AuthApi.getGoogleAuthURL(googleRedirectUri);
      const width = 600;
      const height = 600;
      const left = (window.screen.width - width) / 2;
      const top = (window.screen.height - height) / 2;
      const feature = `width=${width},height=${height},left=${left},top=${top},status=yes,toolbar=no,menubar=no,location=yes`;
      const authWindow = window.open(response.data.url, '_blank', feature);

      if (!authWindow) {
        throw new Error(
          'Popup was blocked. Please allow popups for this site.'
        );
      }

      const timeout = setTimeout(() => {
        authWindow.close();
        setIsAuthenticating(false);
        toast.error('Authentication timeout. Please try again.');
      }, 120000);

      const handleMessage = (
        event: MessageEvent<{ type: string; data: any }>
      ) => {
        if (event.origin !== window.location.origin) return;

        if (event.data?.type === 'GOOGLE_AUTH') {
          const data = event.data.data;
          clearTimeout(timeout);
          authWindow.close();
          window.removeEventListener('message', handleMessage);

          if (!data) {
            toast.error('Authentication failed. Please try again.');
            setIsAuthenticating(false);
            return;
          }

          window.location.href = nextUri || '/campaigns';
        }
      };

      window.addEventListener('message', handleMessage);

      const checkWindow = setInterval(() => {
        if (authWindow.closed) {
          clearInterval(checkWindow);
          clearTimeout(timeout);
          window.removeEventListener('message', handleMessage);
          setIsAuthenticating(false);
        }
      }, 500);
    } catch (error: any) {
      setIsAuthenticating(false);
      toast.error(error.message || 'Failed to authenticate with Google');
    }
  };

  return (
    <div className="h-screen">
      <div className="grid grid-cols-2">
        <div className="flex flex-col items-center justify-center">
          <div className="flex max-w-[400px] flex-col items-center justify-center p-4">
            <Image src={Logo} alt="logo" />
            <h2 className="my-4 text-center text-2xl font-bold">
              Welcome to Cosmo Agents
            </h2>
            <p className="mb-8 text-center text-base text-muted-foreground">
              Sign in with Google to read, compose, and send emails from your
              Gmail account
            </p>
            <Button
              onClick={getGoogleAuth}
              variant="outline"
              className="mb-2 w-full bg-gradient-to-r from-purple-600 to-blue-600 text-base text-white shadow-sm transition-all duration-200 hover:bg-gradient-to-r hover:from-purple-800 hover:to-blue-800 hover:text-white hover:shadow-md"
              size="lg"
            >
              <IconBrandGoogle />
              Sign up with Google
            </Button>

            <Button
              onClick={getGoogleAuth}
              variant="outline"
              className="h-[38px] w-full text-base"
              size="lg"
            >
              <IconBrandGoogle />
              Sign in with Google
            </Button>
          </div>
        </div>
        <div
          style={{
            height: '100vh',
            backgroundImage: `url(${Background.src})`,
            backgroundSize: 'cover',
            backgroundPosition: 'center',
          }}
        />
      </div>
    </div>
  );
}
