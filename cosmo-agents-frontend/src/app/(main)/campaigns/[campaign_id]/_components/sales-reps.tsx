import { SelectMenu } from '@/components/SelectMenu';
import { Button } from '@/components/ui/button';
import { SalesRep } from '@/models/sales-rep';
import { useEffect, useState } from 'react';

export interface AvatarListProps {
  avatars: { src: string; alt: string }[];
  max?: number;
  onMoreClick?: () => void;
}

// Helper to get a random color based on idx or alt to keep it stable for each user
const bgColors = [
  '#F87171', // red-400
  '#60A5FA', // blue-400
  '#34D399', // green-400
  '#FBBF24', // yellow-400
  '#A78BFA', // purple-400
  '#F472B6', // pink-400
  '#38BDF8', // sky-400
  '#FCD34D', // amber-300
  '#4ADE80', // emerald-400
  '#818CF8', // indigo-400
];
// Track used color indexes to avoid duplicates in a single render
const usedColorIndexes: Set<number> = new Set();
export function getUniqueColor(key: string | number) {
  // Deterministic hash for stable color
  let hash = 0;
  for (let i = 0; i < String(key).length; i++) {
    hash = String(key).charCodeAt(i) + ((hash << 5) - hash);
  }
  let idx = Math.abs(hash) % bgColors.length;
  // If already used, find next available color
  let startIdx = idx;
  while (usedColorIndexes.has(idx)) {
    idx = (idx + 1) % bgColors.length;
    if (idx === startIdx) break; // fallback to allow reuse if all used
  }
  usedColorIndexes.add(idx);
  return bgColors[idx];
}

export function AvatarItem({
  src,
  alt,
  idx,
}: {
  src: string;
  alt: string;
  idx: number;
}) {
  const isImg = /^(https?:\/\/|\/)/.test(src);
  return (
    <div
      className="flex items-center justify-center rounded-full text-xs font-bold uppercase ring-1 ring-background"
      style={{
        width: 26,
        height: 26,
        backgroundColor: getUniqueColor(idx || alt || src),
      }}
    >
      {isImg ? (
        <img src={src} width={26} height={26} alt={alt} />
      ) : (
        <span className="text-white">{alt?.[0] || '?'}</span>
      )}
    </div>
  );
}

export function AvatarList({ avatars, max, onMoreClick }: AvatarListProps) {
  const remaining = avatars.length - (max || avatars.length);
  return (
    <div className="flex -space-x-1.5">
      {(max ? avatars.slice(0, max) : avatars).map((avatar, idx) => (
        <AvatarItem key={idx} src={avatar.src} alt={avatar.alt} idx={idx} />
      ))}
      {remaining > 0 && (
        <Button
          variant="ghost"
          className="flex h-[26px] w-[26px] items-center justify-center rounded-full bg-black text-xs text-white ring-1 ring-background hover:bg-secondary hover:text-foreground"
          size="icon"
          onClick={onMoreClick}
        >
          +{remaining}
        </Button>
      )}
    </div>
  );
}

export default function SalesReps({
  onClick,
  value,
  data,
  onChange,
}: {
  onClick: (value: string[]) => void;
  value: string[];
  data: SalesRep[];
  onChange?: (value: string[]) => void;
}) {
  const [salesRepsIds, setSalesRepsIds] = useState<string[]>([]);

  const handleSubmit = (value: string[]) => {
    setSalesRepsIds(value);
    onClick?.(value);
  };

  const handleChange = (value: string[]) => {
    setSalesRepsIds(value);
    onChange?.(value);
  };

  const renderAvatarList = () => {
    return (
      <AvatarList
        avatars={data
          .filter((s) => salesRepsIds.includes(s.id))
          .map((s) => ({ src: '', alt: s.first_name + ' ' + s.last_name }))}
        max={3}
        onMoreClick={() => handleSubmit(data.map((s) => s.id))}
      />
    );
  };

  useEffect(() => {
    if (value.length > 0) {
      setSalesRepsIds(value);
    }
  }, [value]);
  return (
    <SelectMenu
      value={salesRepsIds}
      data={data?.map((s) => ({
        label: s.first_name + ' ' + s.last_name,
        value: s.id,
        email: s.email,
      }))}
      isCustomSubmit
      title="Round Robin"
      textSubmit="Add a sale rep"
      renderItem={(
        item: { label: string; value: string; email: string },
        idx
      ) => (
        <div className="flex items-center gap-2">
          <AvatarItem src={item.value} alt={item.label} idx={idx} />
          <div className="flex flex-col gap-1">
            <span className="font-inter text-[14px] font-semibold leading-[100%] tracking-normal text-[#252525]">
              {item.label}
            </span>
            <span className="font-inter text-[14px] font-normal leading-[100%] tracking-[0%] text-[#6A6A6A]">{`<${item.email}>`}</span>
          </div>
        </div>
      )}
      onSubmit={handleSubmit}
      onChange={handleChange}
    >
      <Button
        variant="outline"
        onClick={() => {}}
        className="flex items-center gap-2"
      >
        <span className="font-inter text-[14px] font-semibold leading-[100%] tracking-normal text-[#C5C5C5]">
          CC:
        </span>
        {salesRepsIds?.length > 0 && data?.length > 0 ? (
          renderAvatarList()
        ) : (
          <span className="font-inter text-[14px] font-semibold leading-[100%] tracking-normal text-[#252525] underline">
            Choose sales reps
          </span>
        )}
      </Button>
    </SelectMenu>
  );
}
