'use client';

import ReactMarkdown from 'react-markdown';

interface TextBlockProps {
  text: string;
}

export function TextBlock({ text }: TextBlockProps) {
  return (
    <div className="prose prose-sm prose-invert max-w-none text-muted-foreground prose-strong:text-foreground prose-a:text-indigo-600 dark:prose-a:text-indigo-400">
      <ReactMarkdown>{text}</ReactMarkdown>
    </div>
  );
}
