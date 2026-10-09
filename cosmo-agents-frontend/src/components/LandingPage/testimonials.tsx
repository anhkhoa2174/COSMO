import { Card, CardContent } from '@/components/ui/card';
import { Avatar, AvatarImage } from '@/components/ui/avatar';
import { Quote, Star } from 'lucide-react';

export function Testimonials() {
  return (
    <div className="bg-[url(/landing-page/testimonial.webp)] py-8">
      <div id="testimonials" className="container mx-auto">
        {/* Top section */}
        <div className="flex flex-col items-center justify-center gap-4">
          {/* Badge */}
          <div className="flex items-center justify-center rounded-md bg-zinc-100 px-4 py-2 text-indigo-700">
            <span className="text-xs font-bold uppercase">Testimonials</span>
          </div>

          {/* Title */}
          <h2 className="text-3xl font-bold tracking-tight md:text-4xl">
            A community of AI revenue innovators
          </h2>

          <p className="max-w-2xl text-center text-[1.05rem] text-muted-foreground">
            Teams across SaaS, consulting, education and real estate use Cosmo
            to keep every conversation moving.
          </p>
        </div>

        {/* Bottom section */}
        <div className="mt-12">
          <div className="grid grid-cols-1 gap-6 sm:grid-cols-2 lg:grid-cols-3">
            <AvatarCard
              image="/landing-page/avatar_1.webp"
              name="Hoang Linh"
              title="CEO at cirCO"
              quote="Cosmo improved our workflow and revenue. The automation has made our sales process more efficient and effective."
            />
            <AvatarCard
              image="/landing-page/avatar_3.webp"
              name="Thanh Huynh Minh"
              title="President at ASOFT"
              quote="Cosmo excels in nurturing conversations. It keeps the dialogue going with potential clients, making our sales funnel more efficient."
            />
            <AvatarCard
              image="/landing-page/avatar_4.webp"
              name="Giang Nguyen"
              title="Sales Manager at CMC"
              quote="Implementing Cosmo transformed our sales. Automation and personalized content significantly increased engagement."
            />
            <AvatarCard
              image="/landing-page/avatar_5.webp"
              name="Tung Hoang"
              title="SAP Consultant at Citek"
              quote="Cosmo has been a powerful asset to our sales team. The way it drives revenue through automated processes is nothing short of impressive."
            />
            <AvatarCard
              image="/landing-page/avatar_6.webp"
              name="Pemi Nguyen"
              title="CEO at Ike Education"
              quote="Cosmo simplifies our sales efforts. The lead segmentation features have dramatically improved our conversion rates and efficiency."
            />
            <AvatarCard
              image="/landing-page/avatar_8.webp"
              name="Michael Phan"
              title="Head of Sales at Glenvill Developments"
              quote="Cosmo has transformed how we approach lead management. The nurturing tools have made our sales strategy much more effective."
            />
          </div>
        </div>
      </div>
    </div>
  );
}

interface AvatarProps {
  image: string;
  name: string;
  title: string;
  quote: string;
}

function AvatarCard(props: Readonly<AvatarProps>) {
  return (
    <Card className="group relative flex h-full flex-col rounded-2xl border bg-white/90 shadow-sm backdrop-blur transition-all duration-200 hover:-translate-y-1 hover:shadow-xl">
      <CardContent className="flex flex-1 flex-col p-6">
        <Quote className="size-7 shrink-0 text-violet-300" aria-hidden="true" />

        <p className="mt-3 flex-1 text-[0.95rem] leading-relaxed text-muted-foreground">
          {props.quote}
        </p>

        <div
          className="mt-5 flex gap-0.5"
          role="img"
          aria-label="Rated 5 out of 5"
        >
          {Array.from({ length: 5 }).map((_, i) => (
            <Star
              key={i}
              className="size-3.5 fill-amber-400 text-amber-400"
              aria-hidden="true"
            />
          ))}
        </div>

        <div className="mt-4 flex items-center gap-3 border-t pt-4">
          <Avatar className="size-11">
            <AvatarImage
              src={props.image}
              alt=""
              className="size-11 rounded-full object-cover"
            />
          </Avatar>
          <div className="min-w-0">
            <p className="truncate font-semibold">{props.name}</p>
            <p className="truncate text-[0.85rem] text-muted-foreground">
              {props.title}
            </p>
          </div>
        </div>
      </CardContent>
    </Card>
  );
}
