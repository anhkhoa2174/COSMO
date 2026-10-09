import Link from 'next/link';
import { ShieldCheck, UserCheck, FileLock2, BellOff } from 'lucide-react';

const COMMITMENTS = [
  {
    icon: ShieldCheck,
    title: 'Your data is never sold or shared',
    body: 'We process customer data only to run the outreach features you sign up for, and one organization’s data never trains a model shared with anyone else.',
  },
  {
    icon: FileLock2,
    title: 'Prospect emails are data, not instructions',
    body: 'Inbound emails and retrieved documents sit in isolated context blocks and model output is limited to a fixed set of intents, so an instruction hidden inside an email cannot make the system act.',
  },
  {
    icon: UserCheck,
    title: 'You decide what goes out',
    body: 'AI-drafted replies to inbound mail are held for review and sent by a person. Campaign sequences run on the schedule you set, and you can pause or stop them at any time.',
  },
  {
    icon: BellOff,
    title: 'Opt-out is honoured',
    body: 'A do-not-contact reply flags the contact as soon as it is processed, and every automated send path skips them from that point on.',
  },
];

export function Trust() {
  return (
    <div className="py-8">
      <div id="policy" className="container mx-auto scroll-mt-28">
        {/* Top section */}
        <div className="flex flex-col items-center justify-center gap-4">
          <div className="flex items-center justify-center rounded-md bg-zinc-100 px-4 py-2 text-indigo-700">
            <span className="text-xs font-bold uppercase">Policy</span>
          </div>

          <h2 className="text-center text-3xl font-bold tracking-tight md:text-4xl">
            Autonomy where it helps, control where it counts
          </h2>

          <p className="max-w-2xl text-center text-[1.05rem] text-muted-foreground">
            COSMO acts on your pipeline every day — so the limits it works
            within are part of the product, not fine print.
          </p>
        </div>

        {/* Commitments */}
        <div className="mt-12 grid grid-cols-1 gap-5 md:grid-cols-2">
          {COMMITMENTS.map(({ icon: Icon, title, body }) => (
            <div
              key={title}
              className="flex gap-4 rounded-xl border bg-background p-6 shadow-sm"
            >
              <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-lg bg-gradient-to-br from-violet-500 to-indigo-600">
                <Icon className="h-5 w-5 text-white" strokeWidth={2} />
              </div>
              <div>
                <h3 className="font-semibold">{title}</h3>
                <p className="mt-1 text-sm leading-relaxed text-muted-foreground">
                  {body}
                </p>
              </div>
            </div>
          ))}
        </div>

        <div className="mt-8 flex flex-col items-center gap-2">
          <Link
            href="/policy"
            className="inline-flex items-center gap-2 rounded-md bg-zinc-900 px-6 py-2.5 text-sm font-medium text-white transition-colors hover:bg-zinc-800"
          >
            Read the full Trust &amp; Data Policy
          </Link>
          <p className="text-xs text-muted-foreground">
            Operators remain responsible for the anti-spam and data-protection
            rules of their jurisdiction.
          </p>
        </div>
      </div>
    </div>
  );
}
