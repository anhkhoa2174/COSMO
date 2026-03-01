'use client';
import { CodePlugin, SkipMarkPlugin } from '@udecode/plate-basic-marks/react';

export const skipMarkPlugin = SkipMarkPlugin.configure({
  options: {
    query: {
      allow: [CodePlugin.key],
    },
  },
});
