import React from 'react';
import type { BuiltInNode, Node, NodeTypes } from '@xyflow/react';
import type { EmailIntent } from '@/models/email';
import { ActionNode } from './action-node';
import { BadgeNode } from './badge-node';
import { CustomNode } from './custom-node';
import { IntentNode } from './intent-node';
import { TemplateNode } from './template-node';

export type CustomNode = Node<
  {
    isTrigger?: boolean;
    isEnd?: boolean;
    icon?: React.ReactNode;
    label: string;
    children: React.ReactNode;
    intent?: string;
    isCompleted?: boolean;
    isDelete?: boolean;
    deleteNode?: (nodeId: string) => void;
  },
  'custom-node'
>;

export type TemplateNode = Node<
  {
    label: string;
    children: React.ReactNode;
    isCompleted?: boolean;
  },
  'template-node'
>;

export type BadgeNode = Node<
  {
    icon: React.ReactNode;
    label: string;
  },
  'badge-node'
>;

type IntentNodeData = {
  label: React.ReactNode;
};
export type IntentNode = Node<IntentNodeData, 'intent-node'>;

export type ActionNode = Node<
  {
    actions: Array<() => void>;
    intent?: EmailIntent;
  },
  'action-node'
>;

export type AppNode =
  | BuiltInNode
  | CustomNode
  | BadgeNode
  | IntentNode
  | ActionNode
  | TemplateNode

export const nodeTypes = {
  'custom-node': CustomNode,
  'badge-node': BadgeNode,
  'intent-node': IntentNode,
  'action-node': ActionNode,
  'template-node': TemplateNode,
} satisfies NodeTypes;
