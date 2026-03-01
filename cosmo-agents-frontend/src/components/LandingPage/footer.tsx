import Link from 'next/link';

export function Footer() {
  return (
    <footer className="bg-zinc-100 py-4">
      <div className="container mx-auto">
        <div className="flex flex-col items-center justify-center gap-2">
          <p className="text-center text-muted-foreground">
            © 2025 Cosmo Agents. Empowering marketing teams with
            revenue-driving AI workforces
          </p>
          <div className="flex gap-4">
            <Link
              href="/tos#privacy"
              className="text-muted-foreground hover:underline"
            >
              Privacy Policy
            </Link>
            <Link href="/tos" className="text-muted-foreground hover:underline">
              Terms of Service
            </Link>
            <Link
              href="/data-processing"
              className="text-muted-foreground hover:underline"
            >
              Data Processing
            </Link>
          </div>
        </div>
      </div>
    </footer>
  );
}
