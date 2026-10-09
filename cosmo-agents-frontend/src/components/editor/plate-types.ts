'use client';

import type React from 'react';

import type { TElement, TText } from '@udecode/plate';
import type { BlockquotePlugin } from '@udecode/plate-block-quote/react';
import type {
  CodeBlockPlugin,
  CodeLinePlugin,
} from '@udecode/plate-code-block/react';
import type { HEADING_KEYS } from '@udecode/plate-heading';
import type { HorizontalRulePlugin } from '@udecode/plate-horizontal-rule/react';
import type { TLinkElement } from '@udecode/plate-link';
import type { LinkPlugin } from '@udecode/plate-link/react';
import type {
  TMentionElement,
  TMentionInputElement,
} from '@udecode/plate-mention';
import type {
  MentionInputPlugin,
  MentionPlugin,
} from '@udecode/plate-mention/react';
import type { ParagraphPlugin } from '@udecode/plate/react';

/** Text */

export type EmptyText = {
  text: '';
};

export interface MyAlignProps {
  align?: React.CSSProperties['textAlign'];
}

export interface MyBlockElement
  extends MyIndentListProps, MyLineHeightProps, TElement {
  id?: string;
}

/** Inline Elements */

export interface MyBlockquoteElement extends MyBlockElement {
  children: MyInlineChildren;
  type: typeof BlockquotePlugin.key;
}

export interface MyCodeBlockElement extends MyBlockElement {
  children: MyCodeLineElement[];
  type: typeof CodeBlockPlugin.key;
}

export interface MyCodeLineElement extends TElement {
  children: PlainText[];
  type: typeof CodeLinePlugin.key;
}

export interface MyH1Element extends MyBlockElement {
  children: MyInlineChildren;
  type: typeof HEADING_KEYS.h1;
}

export interface MyH2Element extends MyBlockElement {
  children: MyInlineChildren;
  type: typeof HEADING_KEYS.h2;
}

/** Block props */

export interface MyH3Element extends MyBlockElement {
  children: MyInlineChildren;
  type: typeof HEADING_KEYS.h3;
}

export interface MyHrElement extends MyBlockElement {
  children: [EmptyText];
  type: typeof HorizontalRulePlugin.key;
}

export interface MyIndentListProps extends MyIndentProps {
  listRestart?: number;
  listStart?: number;
  listStyleType?: string;
}

export interface MyIndentProps {
  indent?: number;
}

/** Blocks */

export type MyInlineChildren = MyInlineDescendant[];

export type MyInlineDescendant = MyInlineElement | RichText;

export type MyInlineElement =
  | MyLinkElement
  | MyMentionElement
  | MyMentionInputElement;

export interface MyLineHeightProps {
  lineHeight?: React.CSSProperties['lineHeight'];
}

export interface MyLinkElement extends TLinkElement {
  children: RichText[];
  type: typeof LinkPlugin.key;
}

export interface MyMentionElement extends TMentionElement {
  children: [EmptyText];
  type: typeof MentionPlugin.key;
}

export interface MyMentionInputElement extends TMentionInputElement {
  children: [PlainText];
  type: typeof MentionInputPlugin.key;
}

export type MyNestableBlock = MyParagraphElement;

export interface MyParagraphElement extends MyBlockElement {
  children: MyInlineChildren;
  type: typeof ParagraphPlugin.key;
}

export type MyRootBlock =
  | MyBlockquoteElement
  | MyCodeBlockElement
  | MyH1Element
  | MyH2Element
  | MyH3Element
  | MyHrElement
  | MyParagraphElement;

export type MyValue = MyRootBlock[];

export type PlainText = {
  text: string;
};

export interface RichText extends TText {
  backgroundColor?: React.CSSProperties['backgroundColor'];
  bold?: boolean;
  code?: boolean;
  color?: React.CSSProperties['color'];
  fontFamily?: React.CSSProperties['fontFamily'];
  fontSize?: React.CSSProperties['fontSize'];
  fontWeight?: React.CSSProperties['fontWeight'];
  italic?: boolean;
  kbd?: boolean;
  strikethrough?: boolean;
  subscript?: boolean;
  underline?: boolean;
}

// export type MyElement = ElementOf<MyEditor>;

// export type MyBlock = Exclude<MyElement, MyInlineElement>;

// export type MyEditor = ReturnType<typeof useCreateEditor>;

// export const useEditor = () => useEditorRef<MyEditor>();
