'use client';

import { BlockquotePlugin } from '@udecode/plate-block-quote/react';
import { SoftBreakPlugin } from '@udecode/plate-break/react';
import { CodeBlockPlugin } from '@udecode/plate-code-block/react';

export const softBreakPlugin = SoftBreakPlugin.configure({
  options: {
    rules: [
      { hotkey: 'shift+enter' },
      {
        hotkey: 'enter',
        query: {
          allow: [
            CodeBlockPlugin.key,
            BlockquotePlugin.key,
          ],
        },
      },
    ],
  },
});
