import { Avatar, AvatarImage } from '@/components/ui/avatar';

export function ChatTypingBubble({
  text,
  fallback,
  avatarUrl,
}: {
  text: string;
  fallback?: string;
  avatarUrl?: string;
}) {
  return (
    <div className="flex items-end">
      <Avatar className="mr-2 h-8 w-8">
        {fallback ? <span>{fallback}</span> : <AvatarImage src={avatarUrl} />}
      </Avatar>

      <div className="max-w-xs rounded-2xl rounded-bl-none bg-gray-100 px-4 py-2 pt-3 text-sm text-black shadow">
        <div className="flex space-x-1">
          {text || (
            <>
              <span className="h-2 w-2 animate-bounce rounded-full bg-gray-500 [animation-delay:.1s]" />
              <span className="h-2 w-2 animate-bounce rounded-full bg-gray-500 [animation-delay:.2s]" />
              <span className="h-2 w-2 animate-bounce rounded-full bg-gray-500 [animation-delay:.3s]" />
            </>
          )}
        </div>
      </div>
    </div>
  );
}
