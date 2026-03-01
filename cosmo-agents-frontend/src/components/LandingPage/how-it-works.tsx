export function HowItWorks() {
  return (
    <div className="py-8">
      <div id="how-it-works" className="container mx-auto">
        {/* Top section */}
        <div className="flex flex-col items-center justify-center gap-4">
          {/* Badge */}
          <div className="flex items-center justify-center rounded-md bg-zinc-100 px-4 py-2 text-indigo-500">
            <span className="text-xs font-bold uppercase">How it works</span>
          </div>

          {/* Title */}
          <h2 className="text-2xl font-bold">
            AI agents that seamlessly fits your workflow
          </h2>

          {/* Description */}
          <p className="text-muted-foreground">
            Nurture customers and book meetings for sales with our Agents
          </p>
        </div>

        {/* Bottom section */}
        <div className="mt-12">
          <img
            src="/landing-page/how_it_works.png"
            alt="how_it_works"
            className="w-full"
          />
        </div>
      </div>
    </div>
  );
}
