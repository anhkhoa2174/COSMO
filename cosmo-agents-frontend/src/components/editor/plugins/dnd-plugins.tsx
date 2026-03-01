'use client';

import { DndPlugin } from '@udecode/plate-dnd';
import { NodeIdPlugin } from '@udecode/plate-node-id';

import { DraggableAboveNodes } from '@/components/plate-ui/draggable';

export const dndPlugins = [
  NodeIdPlugin,
  DndPlugin.configure({
    options: {
      enableScroller: true,
    },
    render: {
      aboveNodes: DraggableAboveNodes,
    },
  }),
] as const;
