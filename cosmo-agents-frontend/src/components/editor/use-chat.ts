'use client';

import { useChat as useBaseChat } from 'ai/react';

export const useChat = () => {
  return useBaseChat({
    id: 'editor',
    api: '/api/ai/command',
    fetch: async (input, init) => {
      const res = await fetch(input, init);

      if (!res.ok) {
        await new Promise((resolve) => setTimeout(resolve, 400));

        return new Response('Oops! Something went wrong', {
          headers: {
            'Content-Type': 'text/plain',
          },
        });
      }

      return res;
    },
  });
};
