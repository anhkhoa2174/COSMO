import type { Metadata } from 'next';
import Link from 'next/link';


export const metadata: Metadata = {
  title: 'Trust & Data Policy | COSMO',
  description:
    'How COSMO handles your data and keeps AI under control: scope of processing, prompt-injection defense, human approval, and opt-out enforcement.',
};

const sections = [
  {
    title: 'Scope of processing',
    body: (
      <p>
        We process customer data to deliver the outreach features your
        organization signs up for. Beyond that, we look at usage and
        performance data — and at your data only in de-identified, aggregated
        form — to keep the service working and improve it. Nothing is processed
        for purposes we have not disclosed here or in the Terms of Service.
      </p>
    ),
  },
  {
    title: 'Your data is not for sale',
    body: (
      <p>
        We do not sell or rent customer data. We never use one
        organization&apos;s data to train models that are shared with other
        organizations.
      </p>
    ),
  },
  {
    title: 'AI safety & prompt-injection defense',
    body: (
      <>
        <p>
          Inbound emails and retrieved documents are treated as untrusted data,
          not as instructions to the AI:
        </p>
        <ul className="mt-3 list-disc space-y-2 pl-6">
          <li>
            In the reply pipeline — the path that reads inbound mail and drafts
            answers — external content is isolated in delimited context blocks
            with an explicit instruction to the model that everything inside is
            data, never a command. Extending the same wrapper to the remaining
            generation prompts is in progress.
          </li>
          <li>
            Classifier output is validated against a fixed set of nine intents.
            Anything the model returns outside that set falls back to
            &quot;unknown&quot; and is routed to a person — the AI cannot
            invent an action of its own.
          </li>
          <li>
            An instruction injected inside an email cannot cause a message to
            be sent. Two intents do write to your records automatically —
            suppressing a contact who asks to be left alone, and creating a
            lead when someone refers a colleague — and both are visible in the
            contact&apos;s activity history.
          </li>
        </ul>
      </>
    ),
  },
  {
    title: 'You decide what goes out',
    body: (
      <>
        <p>
          AI-drafted replies to inbound mail are never sent on their own. Each
          one is held as a draft for review, and a person sends it — we call
          this supervised autonomy: the AI does the work, you sign off.
        </p>
        <p className="mt-3">
          Campaign and playbook sequences are different by design: once you
          build and activate a sequence, it sends on the schedule you set,
          because that is the automation you asked for. You stay in control by
          approving the sequence, pausing it, or stopping it at any time.
        </p>
      </>
    ),
  },
  {
    title: 'Opt-out is honored',
    body: (
      <>
        <p>
          When someone replies asking not to be contacted, the classifier flags
          that contact as do-not-contact as soon as the reply is processed, and
          every automated send path — campaigns, playbooks and scheduled
          follow-ups — skips them from that point on.
        </p>
        <p className="mt-3">
          Two limits worth stating plainly: the flag is applied when the reply
          is processed rather than the instant it arrives, and a person on your
          team can still send a one-off manual email to a suppressed contact.
          Blocking suppressed addresses on manual sends as well is on the
          roadmap.
        </p>
      </>
    ),
  },
  {
    title: 'No advertising on our own behalf',
    body: (
      <p>
        COSMO sends no advertising for itself through your channels. Your
        inboxes and your audience are yours.
      </p>
    ),
  },
  {
    title: 'Compliance responsibility',
    body: (
      <p>
        Operators remain responsible for the anti-spam and data-protection
        rules of their jurisdiction (for example, GDPR and CAN-SPAM).
        COSMO&apos;s controls — opt-out enforcement, per-contact activity
        history, and review of AI replies before sending — are built to support
        that compliance. A full administrative audit log is on the roadmap.
      </p>
    ),
  },
];

export default function TrustAndDataPolicyPage() {
  return (
    <div className="relative">

      <main>
        {/* Hero */}
        <div className="py-16 md:py-20">
          <div className="container mx-auto">
            <div className="flex flex-col items-center justify-center gap-4 text-center">
              <div className="flex items-center justify-center rounded-md bg-zinc-100 px-4 py-2 text-indigo-500">
                <span className="text-xs font-bold uppercase">Trust</span>
              </div>
              <h1 className="text-3xl font-bold tracking-tight md:text-4xl">
                Trust &amp; Data Policy
              </h1>
              <p className="max-w-xl text-muted-foreground">
                How COSMO handles your data and keeps AI under control
              </p>
              <p className="text-sm text-muted-foreground">
                Last updated: August 2026
              </p>
            </div>
          </div>
        </div>

        {/* Policy sections */}
        <div className="container mx-auto pb-16 md:pb-24">
          <div className="mx-auto max-w-3xl">
            <div className="mb-10 rounded-xl border border-amber-200 bg-amber-50 p-5 text-sm leading-relaxed text-amber-900 dark:border-amber-900/40 dark:bg-amber-950/30 dark:text-amber-200">
              <strong className="font-semibold">Prototype notice.</strong>{' '}
              COSMO is a research prototype built as a university project. The
              commitments below describe how the system behaves today and what
              is still on the roadmap; they are published for transparency, not
              as a contract with any entity.
            </div>

            <ol className="space-y-12">
              {sections.map(({ title, body }, index) => (
                <li key={title} className="flex gap-5">
                  <span
                    aria-hidden="true"
                    className="mt-0.5 flex h-8 w-8 shrink-0 items-center justify-center rounded-md bg-zinc-100 text-sm font-bold text-indigo-500"
                  >
                    {index + 1}
                  </span>
                  <div>
                    <h2 className="text-xl font-bold">{title}</h2>
                    <div className="mt-2 leading-relaxed text-muted-foreground">
                      {body}
                    </div>
                  </div>
                </li>
              ))}
            </ol>

            <div className="mt-16 border-t pt-8">
              <h2 className="text-lg font-bold">The rest of the paperwork</h2>
              <p className="mt-2 text-sm text-muted-foreground">
                This page is the plain-language summary. Two further documents
                carry the legal detail:
              </p>
              <ul className="mt-4 space-y-3 text-sm">
                <li>
                  <Link
                    href="/tos"
                    className="font-medium underline underline-offset-4"
                  >
                    Terms of Service
                  </Link>
                  <span className="text-muted-foreground">
                    {' '}
                    — the contract governing use of COSMO, including the
                    privacy section that lists which AI providers process your
                    data and for what purpose.
                  </span>
                </li>
                <li>
                  <Link
                    href="/data-processing"
                    className="font-medium underline underline-offset-4"
                  >
                    Data Processing Addendum
                  </Link>
                  <span className="text-muted-foreground">
                    {' '}
                    — the processor terms your legal team will ask for:
                    sub-processors, transfer mechanisms, retention and deletion.
                  </span>
                </li>
              </ul>
              <p className="mt-6 text-sm text-muted-foreground">
                Where the summary and the legal documents differ, the legal
                documents govern. Questions about how your data is handled?
                Reach out to your COSMO contact and we will walk you through it.
              </p>
            </div>
          </div>
        </div>
      </main>

    </div>
  );
}
