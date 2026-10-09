import { Avatar, AvatarImage } from '@/components/ui/avatar';
import { cn } from '@/lib/utils';

type ChatMessageProps = {
  message: React.ReactNode;
  fromUser?: boolean;
  avatarUrl?: string;
  fallback?: string;
};

export function ChatMessage({
  message,
  fromUser,
  avatarUrl,
  fallback,
}: ChatMessageProps) {
  return (
    <div
      className={cn(
        'mb-4 flex items-end',
        fromUser ? 'justify-end' : 'justify-start'
      )}
    >
      {!fromUser && (
        <Avatar className="mr-2 h-8 w-8">
          {fallback ? <span>{fallback}</span> : <AvatarImage src={avatarUrl} />}
        </Avatar>
      )}
      <div
        className={cn(
          'max-w-xs rounded-2xl px-4 py-2 text-sm shadow',
          fromUser
            ? 'rounded-br-none bg-black text-white'
            : 'rounded-bl-none bg-gray-100 text-black'
        )}
      >
        {message}
      </div>
      {fromUser && avatarUrl && (
        <Avatar className="ml-2 h-5 w-5">
          <AvatarImage src={avatarUrl} />
        </Avatar>
      )}
    </div>
  );
}
