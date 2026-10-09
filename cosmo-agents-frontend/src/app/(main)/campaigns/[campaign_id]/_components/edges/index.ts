import type { Edge, EdgeTypes } from '@xyflow/react';

import CustomEdge from './custom-edge';

export const initialEdges = [
  { id: 'a->b', source: 'a', target: 'b' },
  { id: 'b->c', source: 'b', target: 'c' },
  { id: 'c->d', source: 'c', target: 'd', type: 'smoothstep' },
  { id: 'c->e', source: 'c', target: 'e', type: 'smoothstep' },
  { id: 'c->f', source: 'c', target: 'f', type: 'smoothstep' },
  { id: 'c->g', source: 'c', target: 'g', type: 'smoothstep' },
  { id: 'c->h', source: 'c', target: 'h', type: 'smoothstep' },
  { id: 'c->i', source: 'c', target: 'i', type: 'smoothstep' },
  { id: 'c->j', source: 'c', target: 'j', type: 'smoothstep' },
  { id: 'h->k', source: 'h', target: 'k' },
  { id: 'i->l', source: 'i', target: 'l' },
  { id: 'j->m', source: 'j', target: 'm' },
] satisfies Edge[];

export const edgeTypes = {
  // The animated wire. Registered under its own name rather than shadowing
  // 'smoothstep', so the built-in stays available.
  flow: CustomEdge,
} satisfies EdgeTypes;
