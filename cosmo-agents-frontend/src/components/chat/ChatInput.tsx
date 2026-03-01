import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { BookOpenText, Paperclip, SendHorizonal } from "lucide-react";
import React from "react";

export function ChatInput({ onSend, onUseKnowledge, onAttachFile, placeholder }: { onSend: (msg: string) => void, onUseKnowledge: () => void, onAttachFile: () => void, placeholder?: string }) {
  const [value, setValue] = React.useState("");

  const handleSend = () => {
    if (value.trim()) {
      onSend(value.trim());
      setValue("");
    }
  };

  return (
    <form
      className="relative rounded-lg border bg-background focus-within:ring-1 focus-within:ring-ring p-1 ml-2"
    >
      <div className="flex items-center gap-2 p-2">
        <Input
          className="flex-1"
          placeholder={placeholder || "Type your message here..."}
          value={value}
          onChange={(e) => setValue(e.target.value)}
          onKeyDown={(e) => e.key === "Enter" && handleSend()}
        />
      </div>
      <div className="flex items-center p-3 pt-0">
        <Button variant="ghost" size="icon" onClick={onAttachFile}>
          <Paperclip className="size-4" />
          <span className="sr-only">Attach file</span>
        </Button>

        <Button variant="ghost" size="icon" onClick={onUseKnowledge}>
          <BookOpenText className="size-4" />
          <span className="sr-only">Use Knowledge</span>
        </Button>

        <Button
          size="sm"
          className="ml-auto gap-1.5"
          onClick={handleSend}
        >
          Send
          <SendHorizonal className="size-3.5" />
        </Button>
      </div>
    </form>

  );
}
