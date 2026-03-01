import { Card, CardContent } from '@/components/ui/card';

export function KeyFeatures() {
  return (
    <div className="py-8">
      <div id="key-features" className="container mx-auto">
        {/* Top section */}
        <div className="flex flex-col items-center justify-center gap-4">
          {/* Badge */}
          <div className="flex items-center justify-center rounded-md bg-zinc-100 px-4 py-2 text-indigo-500">
            <span className="text-xs font-bold uppercase">Key features</span>
          </div>

          {/* Title */}
          <h2 className="text-2xl font-bold">
            Put your lead nurturing on autopilot
          </h2>

          {/* Description */}
          <p className="text-muted-foreground">
            Autonomously handle conversations with your AI Marketing Workforce
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
            <img src="/landing-page/crm.png" alt="crm.png" />
            <div className="h-[2px] bg-[#AA95FF]" />
            <div className="p-4">
              <p className="mb-2 text-xl font-bold">Seamless CRM Integration</p>
              <p className="text-muted-foreground">
                Connect your agents to your CRM for personalized content
                creation tailored to each contact
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
      <div className="col-span-1">
        <Card>
          <CardContent className="p-0">
            <img
              src="/landing-page/company-trained-ai.png"
              alt="company-trained-ai.png"
            />
            <div className="h-[2px] bg-[#AA95FF]" />
            <div className="p-4">
              <p className="mb-2 text-xl font-bold">
                Trained using company content
              </p>
              <p className="text-muted-foreground">
                Our agents are trained on your company&apos;s marketing content,
                ensuring consistent brand voice and messaging
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
      <div className="col-span-1 md:col-span-2">
        <Card>
          <CardContent className="p-0">
            <img src="/landing-page/engagement.png" alt="engagement.png" />
            <div className="h-[2px] bg-[#AA95FF]" />
            <div className="p-4">
              <p className="mb-2 text-xl font-bold">
                Orchestrate personalized conversations at scale
              </p>
              <p className="text-muted-foreground">
                Let our agents write personalized emails, handle replies and
                book meetings for sales team at scale
              </p>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
