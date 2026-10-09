import Image from 'next/image';

import crmImage from '../../../public/landing-page/crm.webp';
import trainedImage from '../../../public/landing-page/company-trained-ai.webp';
import engagementImage from '../../../public/landing-page/engagement.webp';

import { Card, CardContent } from '@/components/ui/card';

export function KeyFeatures() {
  return (
    <div className="py-8">
      <div id="key-features" className="container mx-auto">
        {/* Top section */}
        <div className="flex flex-col items-center justify-center gap-4">
          {/* Badge */}
          <div className="flex items-center justify-center rounded-md bg-zinc-100 px-4 py-2 text-indigo-700">
            <span className="text-xs font-bold uppercase">Key features</span>
          </div>

          {/* Title */}
          <h2 className="text-2xl font-bold">Put your outreach on autopilot</h2>

          {/* Description */}
          <p className="text-muted-foreground">
            Prospect, personalise and follow up — with a human check before
            anything leaves your inbox
          </p>
        </div>

        {/* Bottom section */}
        <div className="mt-12">
          <FeatureCards />
        </div>
      </div>
    </div>
  );
}

function FeatureCards() {
  return (
    <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
      <div className="col-span-1">
        <Card>
          <CardContent className="p-0">
            <Image
              src={crmImage}
              alt="Contact record showing enriched company and role data"
              className="h-auto w-full"
              sizes="(min-width: 768px) 33vw, 100vw"
            />
            <div className="h-[2px] bg-[#AA95FF]" />
            <div className="p-4">
              <p className="mb-2 text-xl font-bold">
                Prospects from every channel
              </p>
              <p className="text-muted-foreground">
                Import from Apollo, a CSV, the LinkedIn extension or your own
                lead forms — then let the AI enrich each profile with pain
                points, goals and buying signals you can confirm or reject
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
      <div className="col-span-1">
        <Card>
          <CardContent className="p-0">
            <Image
              src={trainedImage}
              alt="Knowledge base documents used to ground an AI reply"
              className="h-auto w-full"
              sizes="(min-width: 768px) 33vw, 100vw"
            />
            <div className="h-[2px] bg-[#AA95FF]" />
            <div className="p-4">
              <p className="mb-2 text-xl font-bold">
                Grounded in your knowledge base
              </p>
              <p className="text-muted-foreground">
                Upload your product docs and pricing. They are chunked, embedded
                and searchable, so drafts speak in your voice with your facts
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
      <div className="col-span-1 md:col-span-2">
        <Card>
          <CardContent className="p-0">
            <Image
              src={engagementImage}
              alt="Engagement timeline for a contact across several emails"
              className="h-auto w-full"
              sizes="(min-width: 768px) 66vw, 100vw"
            />
            <div className="h-[2px] bg-[#AA95FF]" />
            <div className="p-4">
              <p className="mb-2 text-xl font-bold">
                Replies routed by intent, not by rules
              </p>
              <p className="text-muted-foreground">
                Every inbound reply is classified into one of nine intents and
                routed automatically — draft a response, halt the sequence,
                reschedule around an out-of-office, or hand it to a teammate
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
