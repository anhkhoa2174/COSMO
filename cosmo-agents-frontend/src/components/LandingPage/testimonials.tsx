import { Card, CardContent } from '@/components/ui/card';
import { Avatar, AvatarImage } from '@/components/ui/avatar';

export function Testimonials() {
  return (
    <div className="bg-[url(/landing-page/testimonial.png)] py-8">
      <div id="testimonials" className="container mx-auto">
        {/* Top section */}
        <div className="flex flex-col items-center justify-center gap-4">
          {/* Badge */}
          <div className="flex items-center justify-center rounded-md bg-zinc-100 px-4 py-2 text-indigo-500">
            <span className="text-xs font-bold uppercase">Testimonials</span>
          </div>

          {/* Title */}
          <h2 className="text-2xl font-bold">
            A Community of AI Revenue Innovators
          </h2>

          {/* Description */}
        </div>

        {/* Bottom section */}
        <div className="mt-12">
          <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 md:grid-cols-4">
            <AvatarCard
              image="/landing-page/avatar_1.png"
              name="Hoang Linh"
              title="CEO at cirCO"
              quote="Cosmo Agents improved our workflow and revenue. The automation has made our sales process more efficient and effective."
            />
            <AvatarCard
              image="/landing-page/avatar_3.png"
              name="Thanh Huynh Minh"
              title="President at ASOFT"
              quote="Cosmo Agents excels in nurturing conversations. It keeps the dialogue going with potential clients, making our sales funnel more efficient."
            />
            <AvatarCard
              image="/landing-page/avatar_4.png"
              name="Giang Nguyen"
              title="Sales Manager at CMC"
              quote="Implementing Cosmo Agents transformed our sales. Automation and personalized content significantly increased engagement."
            />
            <AvatarCard
              image="/landing-page/avatar_5.png"
              name="Tung Hoang"
              title="SAP Consultant at Citek"
              quote="Cosmo Agents has been a powerful asset to our sales team. The way it drives revenue through automated processes is nothing short of impressive."
            />
            <AvatarCard
              image="/landing-page/avatar_6.png"
              name="Pemi Nguyen"
              title="CEO at Ike Education"
              quote="Cosmo Agents simplifies our sales efforts. The automation and lead segmentation features have dramatically improved our conversion rates and efficiency."
            />
            <AvatarCard
              image="/landing-page/avatar_8.png"
              name="Michael Phan"
              title="Head of Sales at Glenvill Developments"
              quote="Cosmo Agents has transformed how we approach lead management. The nurturing tools have made our sales strategy much more effective."
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
    <Card className="rounded-lg bg-white shadow-md">
      <div className="flex justify-between p-2">
        <div className="h-4 w-4 rounded-full bg-gray-100 shadow-md" />
        <div className="h-4 w-4 rounded-full bg-gray-100 shadow-md" />
      </div>
      <CardContent className="px-6 pb-0">
        <p className="rounded-xl border border-gray-200 p-2 text-gray-600">
          {props.quote}
        </p>
        <div className="mt-4 flex items-center justify-between">
          <div>
            <p className="font-bold">{props.name}</p>
            <p className="mt-1 max-w-[170px] text-sm font-medium">
              {props.title}
            </p>
          </div>
          <Avatar>
            <AvatarImage
              src={props.image}
              alt="avatar"
              className="h-12 w-12 rounded-full"
            />
          </Avatar>
        </div>
      </CardContent>
      <div className="flex justify-between p-2">
        <div className="h-4 w-4 rounded-full bg-gray-100 shadow-md" />
        <div className="h-4 w-4 rounded-full bg-gray-100 shadow-md" />
      </div>
    </Card>
  );
}
