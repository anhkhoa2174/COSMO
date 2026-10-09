export default function DescriptionText({
  title,
  description,
}: {
  title: string;
  description: string;
}) {
  return (
    <div className="flex flex-col items-start gap-2">
      <p className="font-semibold">{title}</p>
      <p className="text-start">{description}</p>
    </div>
  );
}
