'use client';

import { BasicMarksPlugin } from '@udecode/plate-basic-marks/react';
import { BlockquotePlugin } from '@udecode/plate-block-quote/react';
import { CodeBlockPlugin } from '@udecode/plate-code-block/react';
import { HeadingPlugin } from '@udecode/plate-heading/react';
import { createLowlight } from 'lowlight';
import bash from 'highlight.js/lib/languages/bash';
import css from 'highlight.js/lib/languages/css';
import htmlLang from 'highlight.js/lib/languages/xml';
import javascript from 'highlight.js/lib/languages/javascript';
import json from 'highlight.js/lib/languages/json';
import python from 'highlight.js/lib/languages/python';
import sql from 'highlight.js/lib/languages/sql';
import typescript from 'highlight.js/lib/languages/typescript';

/**
 * Syntax highlighting for code blocks in the editor.
 *
 * This used to be `createLowlight(all)`, which registers every language
 * highlight.js ships — around 190 grammars, including asciidoc, LiveScript and
 * nginx config. That produced a 1.4 MB chunk loaded by the AI Inbox, the
 * campaign builder and the agent editor, and it was the single largest cost in
 * their first load.
 *
 * The editor writes sales emails. A code block in one is rare, and when it
 * appears it is a snippet, a query, or a payload — so the languages below are
 * registered and the rest are not. An unregistered language still renders as a
 * code block, just without colouring, which is the right way for this to
 * degrade.
 */
const lowlight = createLowlight({
  bash,
  css,
  html: htmlLang,
  javascript,
  json,
  python,
  sql,
  typescript,
});

export const basicNodesPlugins = [
  HeadingPlugin.configure({ options: { levels: 3 } }),
  BlockquotePlugin,
  CodeBlockPlugin.configure({ options: { lowlight } }),
  BasicMarksPlugin,
] as const;
