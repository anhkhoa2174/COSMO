import Link from 'next/link';
import Image from 'next/image';
import { MainButton } from '@/components/buttons/main-button';

export default function NotFound() {
  return (
    <div className="ca-grid-line flex h-screen flex-col items-center justify-center">
      <div className="flex items-center gap-4">
        <Image
          src="/404-error.png"
          alt="404 error image"
          width={370}
          height={377}
        />
        <div className="ml-8 flex flex-col gap-4">
          <h2 className="text-2xl font-bold">Page not found</h2>
          <p className="text-base text-muted-foreground">
            The page you're searching for isn't available
          </p>
          <Link href="/" passHref>
            <MainButton text="Go home" />
          </Link>
        </div>
      </div>
    </div>
  );
}
