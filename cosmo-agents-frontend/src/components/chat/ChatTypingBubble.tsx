import { Avatar, AvatarImage } from "@/components/ui/avatar";

export function ChatTypingBubble({ text, fallback, avatarUrl }: { text: string, fallback?: string, avatarUrl?: string }) {
    return (
      <div className="flex items-end">
        <Avatar className="h-8 w-8 mr-2">
          {fallback ? <span>{fallback}</span> : <AvatarImage src={avatarUrl} />}
        </Avatar>
  
        <div className="bg-gray-100 text-black text-sm px-4 py-2 rounded-2xl rounded-bl-none shadow max-w-xs pt-3">
          <div className="flex space-x-1">
            {text || <>
            <span className="h-2 w-2 bg-gray-500 rounded-full animate-bounce [animation-delay:.1s]" />
            <span className="h-2 w-2 bg-gray-500 rounded-full animate-bounce [animation-delay:.2s]" />
            <span className="h-2 w-2 bg-gray-500 rounded-full animate-bounce [animation-delay:.3s]" />
            </>}
          </div>
        </div>
      </div>
    );
  }
  