import Image from 'next/image';
import howItWorksImage from '../../../public/landing-page/how_it_works.webp';

/** The five pipeline stages, one short line each. */
const STEPS = [
  {
    title: 'Identify',
    desc: 'Pull in leads from Apollo, a CSV or LinkedIn',
  },
  {
    title: 'Research',
    desc: 'Enrich each profile with pain points and buying signals',
  },
  {
    title: 'Outreach',
    desc: 'Send personalized, human-approved emails',
  },
  {
    title: 'Nurture',
    desc: 'Follow up and route every reply by intent',
  },
  {
    title: 'Engage',
    desc: 'Book meetings and track your pipeline',
  },
];

export function HowItWorks() {
  return (
    <div className="py-8">
      <div id="how-it-works" className="container mx-auto">
        {/* Top section */}
        <div className="flex flex-col items-center justify-center gap-4">
          {/* Badge */}
          <div className="flex items-center justify-center rounded-md bg-zinc-100 px-4 py-2 text-indigo-700">
            <span className="text-xs font-bold uppercase">How it works</span>
          </div>

          {/* Title */}
          <h2 className="text-2xl font-bold">
            An AI agent that fits the way your team already sells
          </h2>

          {/* Description */}
          <p className="text-muted-foreground">
            Every morning it tells you who to contact and why — you stay the one
            who approves what gets sent
          </p>
        </div>

        {/* Pipeline steps */}
        <div className="mt-12 grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-5">
          {STEPS.map(({ title, desc }, index) => (
            <div
              key={title}
              className="rounded-xl border bg-background p-4 shadow-sm"
            >
              <div className="flex items-center gap-2.5">
                <span className="grid size-6 shrink-0 place-items-center rounded-md bg-zinc-100 text-xs font-bold text-indigo-700">
                  {index + 1}
                </span>
                <p className="font-semibold">{title}</p>
              </div>
              <p className="mt-2 text-sm leading-snug text-muted-foreground">
                {desc}
              </p>
            </div>
          ))}
        </div>

        {/* Diagram */}
        <div className="mt-6 w-full rounded-2xl border bg-background p-4 shadow-md sm:p-6">
          <Image
            src={howItWorksImage}
            alt="how_it_works"
            className="w-full"
            sizes="100vw"
          />
        </div>
      </div>
    </div>
  );
}
