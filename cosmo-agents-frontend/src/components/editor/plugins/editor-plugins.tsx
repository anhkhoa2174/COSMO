'use client';

import {
  FontBackgroundColorPlugin,
  FontColorPlugin,
  FontSizePlugin,
} from '@udecode/plate-font/react';
import { HighlightPlugin } from '@udecode/plate-highlight/react';
import { HorizontalRulePlugin } from '@udecode/plate-horizontal-rule/react';
import { JuicePlugin } from '@udecode/plate-juice';
import { KbdPlugin } from '@udecode/plate-kbd/react';
import { MarkdownPlugin } from '@udecode/plate-markdown';
import { TrailingBlockPlugin } from '@udecode/plate-trailing-block';

import { FixedToolbarPlugin } from '@/components/editor/plugins/fixed-toolbar-plugin';
import { FloatingToolbarPlugin } from '@/components/editor/plugins/floating-toolbar-plugin';

import { aiPlugins } from './ai-plugins';
import { alignPlugin } from './align-plugin';
import { autoformatPlugin } from './autoformat-plugin';
import { basicNodesPlugins } from './basic-nodes-plugins';
import { blockMenuPlugins } from './block-menu-plugins';
import { cursorOverlayPlugin } from './cursor-overlay-plugin';
import { deletePlugins } from './delete-plugins';
import { dndPlugins } from './dnd-plugins';
import { exitBreakPlugin } from './exit-break-plugin';
import { indentListPlugins } from './indent-list-plugins';
import { linkPlugin } from './link-plugin';
import { mentionPlugin } from './mention-plugin';
import { resetBlockTypePlugin } from './reset-block-type-plugin';
import { skipMarkPlugin } from './skip-mark-plugin';
import { softBreakPlugin } from './soft-break-plugin';
import { tocPlugin } from './toc-plugin';

export const viewPlugins = [
  ...basicNodesPlugins,
  HorizontalRulePlugin,
  linkPlugin,
  mentionPlugin,
  tocPlugin,

  // Marks
  FontColorPlugin,
  FontBackgroundColorPlugin,
  FontSizePlugin,
  HighlightPlugin,
  KbdPlugin,
  skipMarkPlugin,

  // Block Style
  alignPlugin,
  ...indentListPlugins,
] as const;

export const editorPlugins = [
  // AI
  ...aiPlugins,

  // Nodes
  ...viewPlugins,

  // Functionality
  autoformatPlugin,
  cursorOverlayPlugin,
  ...blockMenuPlugins,
  ...dndPlugins,
  exitBreakPlugin,
  resetBlockTypePlugin,
  ...deletePlugins,
  softBreakPlugin,
  TrailingBlockPlugin,

  // Deserialization
  MarkdownPlugin.configure({ options: { indentList: true } }),
  JuicePlugin,

  // UI
  FixedToolbarPlugin,
  FloatingToolbarPlugin,
];
