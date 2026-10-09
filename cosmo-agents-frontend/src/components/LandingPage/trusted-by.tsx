/**
 * Wordmarks for the companies that already appear by name in the testimonials
 * below. Nothing here is invented — every entry maps to a quoted customer, so
 * the band stays an honest summary of that section rather than borrowed logos.
 */
const CUSTOMERS = [
  'cirCO',
  'ASOFT',
  'CMC',
  'Citek',
  'Ike Education',
  'Glenvill Developments',
];

export function TrustedBy() {
  return (
    <section className="border-y bg-muted/30 py-10">
      <div className="container mx-auto px-4">
        <p className="text-center text-[0.7rem] font-semibold uppercase tracking-[0.18em] text-muted-foreground">
          Sales teams already running on Cosmo
        </p>
        <ul className="mt-6 flex flex-wrap items-center justify-center gap-x-10 gap-y-5">
          {CUSTOMERS.map((customer) => (
            <li
              key={customer}
              className="text-lg font-semibold tracking-tight text-muted-foreground/70 transition-colors hover:text-foreground"
            >
              {customer}
            </li>
          ))}
        </ul>
      </div>
    </section>
  );
}
