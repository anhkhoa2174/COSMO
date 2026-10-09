'use client';

import { IconBrandGoogle } from '@/assets/icons';
import { googleRedirectUri } from '@/helpers/env';
import AuthApi from '@/network/client/auth';
import { Button } from '@/components/ui/button';
import { CosmoMark } from '@/components/nav/cosmo-mark';
import { MarketingPanel } from '@/app/auth/login/_components/marketing-panel';
import { cn } from '@/lib/utils';
import {
  CheckCircle2,
  Eye,
  EyeOff,
  Lock,
  LogIn,
  Mail,
  ShieldCheck,
  Sparkles,
  UserPlus,
} from 'lucide-react';
import Link from 'next/link';
import { useSearchParams } from 'next/navigation';
import { useState } from 'react';
import { toast } from 'sonner';

const TRUST = [
  'No credit card required — start free',
  'Cancel anytime, your data stays yours',
  'GDPR-compliant, encrypted in transit',
];

export default function AuthLoginPage() {
  const searchParams = useSearchParams();
  const nextUri = searchParams.get('next');

  const [isAuthenticating, setIsAuthenticating] = useState(false);
  const [mode, setMode] = useState<'signin' | 'signup'>('signin');
  const [showPassword, setShowPassword] = useState(false);

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

          window.location.href = nextUri || '/ai-inboxes';
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
    <div className="grid min-h-screen lg:grid-cols-2">
      <div className="flex flex-col bg-gradient-to-br from-white via-white to-violet-50 p-6 md:p-10">
        <header className="flex items-center justify-between gap-4">
          <Link href="/" className="flex items-center gap-2.5">
            <CosmoMark size={34} />
            <span className="text-xl font-bold tracking-tight">Cosmo</span>
          </Link>
          <span className="inline-flex items-center gap-1.5 rounded-full bg-emerald-50 px-3 py-1.5 text-[0.8rem] font-medium text-emerald-700">
            <ShieldCheck className="size-3.5" />
            Verified by Google OAuth
          </span>
        </header>

        <main className="mx-auto flex w-full max-w-[26rem] flex-1 flex-col justify-center py-10">
          <span className="mb-5 inline-flex w-fit items-center gap-2 rounded-full bg-violet-100 px-3.5 py-1.5 text-[0.8rem] font-medium text-violet-700">
            <Sparkles className="size-3.5" />
            AI-Native Sales Outreach
          </span>

          <h1 className="text-3xl font-bold tracking-tight md:text-[2.15rem]">
            Welcome to Cosmo
          </h1>
          <p className="mt-3 text-[0.95rem] leading-relaxed text-muted-foreground">
            Sign in with Google to read, compose, and send emails from your
            Gmail account. Your AI agent gets to work in seconds.
          </p>

          <div className="mt-7 grid grid-cols-2 gap-1 rounded-xl border bg-muted/50 p-1">
            <button
              type="button"
              onClick={() => setMode('signin')}
              className={cn(
                'inline-flex items-center justify-center gap-2 rounded-lg py-2.5 text-[0.9rem] font-semibold transition-colors',
                mode === 'signin'
                  ? 'bg-gradient-to-r from-violet-600 to-blue-600 text-white shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              <LogIn className="size-4" />
              Sign in
            </button>
            <button
              type="button"
              onClick={() => setMode('signup')}
              className={cn(
                'inline-flex items-center justify-center gap-2 rounded-lg py-2.5 text-[0.9rem] font-semibold transition-colors',
                mode === 'signup'
                  ? 'bg-gradient-to-r from-violet-600 to-blue-600 text-white shadow-sm'
                  : 'text-muted-foreground hover:text-foreground'
              )}
            >
              <UserPlus className="size-4" />
              Sign up
            </button>
          </div>

          <form
            className="mt-6 space-y-4"
            onSubmit={(event) => {
              event.preventDefault();
              // No credentials endpoint exists yet — the backend only issues
              // tokens through the Google OAuth exchange.
              toast.info(
                'Email and password sign-in is not enabled yet. Please continue with Google.'
              );
            }}
          >
            <div className="space-y-1.5">
              <label
                htmlFor="email"
                className="text-[0.7rem] font-semibold uppercase tracking-[0.12em] text-muted-foreground"
              >
                Email address
              </label>
              <div className="relative">
                <Mail className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <input
                  id="email"
                  type="email"
                  autoComplete="email"
                  placeholder="you@company.com"
                  className="h-12 w-full rounded-xl border bg-background pl-10 pr-3.5 text-[0.95rem] outline-none ring-offset-background transition-shadow focus-visible:ring-2 focus-visible:ring-ring"
                />
              </div>
            </div>

            <div className="space-y-1.5">
              <div className="flex items-center justify-between">
                <label
                  htmlFor="password"
                  className="text-[0.7rem] font-semibold uppercase tracking-[0.12em] text-muted-foreground"
                >
                  Password
                </label>
                <button
                  type="button"
                  onClick={() =>
                    toast.info(
                      'Password reset is not available — accounts are managed through Google.'
                    )
                  }
                  className="text-[0.8rem] font-medium text-violet-600 hover:underline"
                >
                  Forgot?
                </button>
              </div>
              <div className="relative">
                <Lock className="pointer-events-none absolute left-3.5 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
                <input
                  id="password"
                  type={showPassword ? 'text' : 'password'}
                  autoComplete={
                    mode === 'signin' ? 'current-password' : 'new-password'
                  }
                  placeholder="••••••••"
                  className="h-12 w-full rounded-xl border bg-background pl-10 pr-11 text-[0.95rem] outline-none ring-offset-background transition-shadow focus-visible:ring-2 focus-visible:ring-ring"
                />
                <button
                  type="button"
                  aria-label={showPassword ? 'Hide password' : 'Show password'}
                  onClick={() => setShowPassword((v) => !v)}
                  className="absolute right-3 top-1/2 -translate-y-1/2 text-muted-foreground transition-colors hover:text-foreground"
                >
                  {showPassword ? (
                    <EyeOff className="size-4" />
                  ) : (
                    <Eye className="size-4" />
                  )}
                </button>
              </div>
            </div>

            <Button
              type="submit"
              size="lg"
              className="h-12 w-full bg-gradient-to-r from-violet-600 to-blue-600 text-[0.95rem] font-semibold text-white hover:from-violet-700 hover:to-blue-700"
            >
              {mode === 'signin' ? 'Sign in' : 'Create account'}
            </Button>
          </form>

          <div className="my-6 flex items-center gap-4">
            <span className="h-px flex-1 bg-border" />
            <span className="text-[0.7rem] font-semibold uppercase tracking-[0.12em] text-muted-foreground">
              Or continue with
            </span>
            <span className="h-px flex-1 bg-border" />
          </div>

          <Button
            onClick={getGoogleAuth}
            variant="outline"
            size="lg"
            className="h-12 w-full gap-2 text-[0.95rem] font-medium"
          >
            <IconBrandGoogle />
            Google
          </Button>

          <ul className="mt-7 space-y-2.5">
            {TRUST.map((item) => (
              <li
                key={item}
                className="flex items-center gap-2.5 text-[0.85rem] text-muted-foreground"
              >
                <CheckCircle2 className="size-4 shrink-0 text-emerald-500" />
                {item}
              </li>
            ))}
          </ul>
        </main>

        <footer className="flex flex-wrap items-center justify-between gap-3 text-[0.8rem] text-muted-foreground">
          <span>© {new Date().getFullYear()} Cosmo. All rights reserved.</span>
          <span className="flex gap-5">
            <Link href="/tos" className="hover:text-foreground">
              Terms
            </Link>
            <Link href="/data-processing" className="hover:text-foreground">
              Privacy
            </Link>
          </span>
        </footer>
      </div>

      <MarketingPanel />
    </div>
  );
}
