import { Avatar, AvatarImage } from "@/components/ui/avatar";
import { cn } from "@/lib/utils";

type ChatMessageProps = {
  message: React.ReactNode;
  fromUser?: boolean;
  avatarUrl?: string;
  fallback?: string;
};

export function ChatMessage({ message, fromUser, avatarUrl, fallback }: ChatMessageProps) {
  return (
    <div className={cn("flex items-end mb-4", fromUser ? "justify-end" : "justify-start")}>
      {!fromUser && (
        <Avatar className="h-8 w-8 mr-2">
          {fallback ? <span>{fallback}</span> : <AvatarImage src={avatarUrl} />}
        </Avatar>
      )}
      <div
        className={cn(
          "max-w-xs rounded-2xl px-4 py-2 text-sm shadow",
          fromUser
            ? "bg-black text-white rounded-br-none"
            : "bg-gray-100 text-black rounded-bl-none"
        )}
      >
        {message}
      </div>
      {fromUser && avatarUrl && (
        <Avatar className="h-5 w-5 ml-2">
          <AvatarImage src={avatarUrl} />
        </Avatar>
      )}
    </div>
  );
}
