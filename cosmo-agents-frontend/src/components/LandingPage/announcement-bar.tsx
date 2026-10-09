import { Sparkles } from 'lucide-react';

const MESSAGES = [
  '🏢 Built for B2B SaaS, agencies and consulting firms',
  '🤖 60+ AI tools, 9 intent classes, real-time reply routing',
  '📈 Teams report 3-5x more booked meetings in their first month',
  '🎯 Join 100+ teams already on the waitlist',
];

function Track({ ariaHidden }: { ariaHidden?: boolean }) {
  return (
    <div
      aria-hidden={ariaHidden}
      className="flex shrink-0 items-center gap-10 pr-10"
    >
      {MESSAGES.map((message) => (
        <span key={message} className="flex items-center gap-2.5 text-nowrap">
          <Sparkles className="size-3.5 shrink-0 opacity-80" />
          {message}
        </span>
      ))}
    </div>
  );
}

/** The scrolling proof bar that sits above the landing header. */
export function AnnouncementBar() {
  return (
    <div className="overflow-hidden bg-gradient-to-r from-violet-600 via-purple-600 to-blue-600 py-2.5 text-[0.85rem] font-medium text-white">
      {/* Two identical tracks scroll as one loop; the duplicate is hidden from
          assistive tech so the messages are not announced twice. */}
      <div className="flex w-max animate-marquee motion-reduce:animate-none">
        <Track />
        <Track ariaHidden />
      </div>
    </div>
  );
}
