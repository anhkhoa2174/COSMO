import { IconSparkles } from '@/assets/icons';
import type { CampaignAction } from '@/models/campaign';
import type { EmailIntent } from '@/models/email';
import { Braces, FolderUp, Forward } from 'lucide-react';
import { type Edge, type SmoothStepPathOptions } from '@xyflow/react';
import { AppNode } from './_components/nodes';

export const X_OFFSET = 25;
export const Y_OFFSET = 50;
export const NODE_WIDTH = 254;
export const NODE_GAP = 64;

export const CAMPAIGN_NODE_ID = {
  ENTRY_RULES: 'ENTRY_RULES',
  OUTREACH_EMAIL: 'OUTREACH_EMAIL',
  CLASSIFY: 'CLASSIFY',
  ACTION_HOLD: 'ACTION_HOLD',
};

const INTENT_NODE_ID = {
  INTERESTED: 'INTERESTED',
  NOT_INTERESTED: 'NOT_INTERESTED',
  PRICING: 'PRICING',
  INFORMATION: 'INFORMATION',
  DO_NOT_CONTACT: 'DO_NOT_CONTACT',
  OUT_OF_OFFICE: 'OUT_OF_OFFICE',
  UNKNOWN: 'UNKNOWN',
};
const AI_ACTION_NODE_ID = {
  DO_NOT_CONTACT: 'DO_NOT_CONTACT-AI-ACTION',
  OUT_OF_OFFICE: 'OUT_OF_OFFICE-AI-ACTION',
  UNKNOWN: 'UNKNOWN-AI-ACTION',
};
export const INTENT_NODE_IDS = Object.values(INTENT_NODE_ID);

export const ACTION_NODE_ID = {
  AI_REPLY: 'AI_REPLY',
  DRAFT_EMAIL: 'DRAFT_EMAIL',
  ASSIGN_PERSON: 'ASSIGN_PERSON',
};
export const ACTION_NODE_IDS = Object.values(ACTION_NODE_ID);

export const INTENT_TYPE: Record<string, EmailIntent> = {
  [INTENT_NODE_ID.INTERESTED]: 'Interested',
  [INTENT_NODE_ID.NOT_INTERESTED]: 'Not interested',
  [INTENT_NODE_ID.PRICING]: 'Request for pricing',
  [INTENT_NODE_ID.INFORMATION]: 'Request for information',
  [INTENT_NODE_ID.DO_NOT_CONTACT]: 'Do not contact',
  [INTENT_NODE_ID.OUT_OF_OFFICE]: 'Out of office',
  [INTENT_NODE_ID.UNKNOWN]: 'Unknown intent',
};
export const getIntentNodeIdByType = (type: EmailIntent): string | undefined =>
  Object.keys(INTENT_TYPE).find((key) => INTENT_TYPE[key] === type);

export const ACTION_TYPE: Record<string, CampaignAction> = {
  [ACTION_NODE_ID.AI_REPLY]: 'Let AI reply',
  [ACTION_NODE_ID.DRAFT_EMAIL]: 'Draft an email',
  [ACTION_NODE_ID.ASSIGN_PERSON]: 'Assign to a person',
};
export const getActionNodeIdByType = (type: CampaignAction): string | undefined =>
  Object.keys(ACTION_TYPE).find((key) => ACTION_TYPE[key] === type);

export const actionNodeData = {
  [ACTION_NODE_ID.AI_REPLY]: {
    label: 'Let AI reply',
    icon: <IconSparkles width={20} height={20} />,
    children: 'Let AI sends a personalized reply',
    isEnd: true,
  },
  [ACTION_NODE_ID.DRAFT_EMAIL]: {
    label: 'Draft an email',
    icon: <Forward width={20} height={20} />,
    children: 'Draft an email to send',
    isEnd: true,
  },
  [ACTION_NODE_ID.ASSIGN_PERSON]: {
    label: 'Assign to a person',
    icon: <Braces width={20} height={20} />,
    children: 'Assign to a person to follow up',
    isEnd: true,
  },
};

export const INTENT_NODE_POS = {
  [INTENT_NODE_ID.INTERESTED]: { x: X_OFFSET + 2.5 * NODE_WIDTH + 3 * NODE_GAP, y: Y_OFFSET + 105 },
  [INTENT_NODE_ID.NOT_INTERESTED]: {
    x: X_OFFSET + 2.5 * NODE_WIDTH + 3 * NODE_GAP,
    y: Y_OFFSET + 185,
  },
  [INTENT_NODE_ID.PRICING]: { x: X_OFFSET + 2.5 * NODE_WIDTH + 3 * NODE_GAP, y: Y_OFFSET + 265 },
  [INTENT_NODE_ID.INFORMATION]: {
    x: X_OFFSET + 2.5 * NODE_WIDTH + 3 * NODE_GAP,
    y: Y_OFFSET + 345,
  },
  [INTENT_NODE_ID.DO_NOT_CONTACT]: {
    x: X_OFFSET + 2.5 * NODE_WIDTH + 3 * NODE_GAP,
    y: Y_OFFSET + 425,
  },
  [INTENT_NODE_ID.OUT_OF_OFFICE]: {
    x: X_OFFSET + 2.5 * NODE_WIDTH + 3 * NODE_GAP,
    y: Y_OFFSET + 554,
  },
  [INTENT_NODE_ID.UNKNOWN]: { x: X_OFFSET + 2.5 * NODE_WIDTH + 3 * NODE_GAP, y: Y_OFFSET + 684 },
};

export const initialNodes: AppNode[] = [
  {
    id: CAMPAIGN_NODE_ID.ENTRY_RULES,
    type: 'custom-node',
    position: { x: X_OFFSET, y: Y_OFFSET },
    data: {
      isTrigger: true,
      icon: <FolderUp width={20} height={20} />,
      label: 'Entry Rules',
      children: 'Closed lost contact list',
    },
  },
];

export const afterOutreachEmailNodes: AppNode[] = [
  {
    id: CAMPAIGN_NODE_ID.CLASSIFY,
    type: 'badge-node',
    position: { x: X_OFFSET + 2 * NODE_WIDTH + 3 * NODE_GAP, y: Y_OFFSET + 25 },
    data: {
      icon: <IconSparkles />,
      label: 'AI classified replies',
    },
  },
  {
    id: INTENT_NODE_ID.INTERESTED,
    type: 'intent-node',
    position: INTENT_NODE_POS[INTENT_NODE_ID.INTERESTED],
    data: {
      label: <p>👌 Interested</p>,
    },
  },
  {
    id: INTENT_NODE_ID.NOT_INTERESTED,
    type: 'intent-node',
    position: INTENT_NODE_POS[INTENT_NODE_ID.NOT_INTERESTED],
    data: {
      label: <p>😐 Not interested</p>,
    },
  },
  {
    id: INTENT_NODE_ID.PRICING,
    type: 'intent-node',
    position: INTENT_NODE_POS[INTENT_NODE_ID.PRICING],
    data: {
      label: <p>👌 Request for pricing</p>,
    },
  },
  {
    id: INTENT_NODE_ID.INFORMATION,
    type: 'intent-node',
    position: INTENT_NODE_POS[INTENT_NODE_ID.INFORMATION],
    data: {
      label: <p>🗂️ Request for information</p>,
    },
  },
  {
    id: INTENT_NODE_ID.DO_NOT_CONTACT,
    type: 'intent-node',
    position: INTENT_NODE_POS[INTENT_NODE_ID.DO_NOT_CONTACT],
    data: {
      label: (
        <div className='flex flex-col items-start gap-2'>
          <p>
            ❌ Do not contact <span className="text-muted">default</span>
          </p>
          <div className='flex items-start gap-1 italic text-[#798088]'>
            <span className='w-5 h-5'>🤖</span>
            <div>AI marks contact as DNC status = yes</div>
          </div>
        </div>
      ),
    },
  },
  {
    id: INTENT_NODE_ID.OUT_OF_OFFICE,
    type: 'intent-node',
    position: INTENT_NODE_POS[INTENT_NODE_ID.OUT_OF_OFFICE],
    data: {
      label: (
        <div className='flex flex-col items-start gap-2'>
          <p>
            ✈️ Out of office <span className="text-muted">default</span>
          </p>
          <div className='flex items-start gap-1 italic text-[#798088]'>
            <span className='w-5 h-5'>🤖</span> 
            <div>AI extracts dates then schedules follow up</div>
          </div>
        </div>
      ),
    },
  },
  {
    id: INTENT_NODE_ID.UNKNOWN,
    type: 'intent-node',
    position: INTENT_NODE_POS[INTENT_NODE_ID.UNKNOWN],
    data: {
      label: (
        <div className='flex flex-col items-start gap-2'>
          <p>
            ❓ Unknown intent <span className="text-muted">default</span>
          </p>
          <div className='flex items-start gap-1 italic text-[#798088]'>
            <span className='w-5 h-5'>🤖</span> 
            <div>AI routes to campaign owner</div>
          </div>
        </div>

      ),
    },
  },
  {
    id: AI_ACTION_NODE_ID.DO_NOT_CONTACT,
    type: 'custom-node',
    position: {
      x: INTENT_NODE_POS[INTENT_NODE_ID.DO_NOT_CONTACT].x + NODE_WIDTH + NODE_GAP,
      y: INTENT_NODE_POS[INTENT_NODE_ID.DO_NOT_CONTACT].y - 0.5,
    },
    data: {
      icon: <span className='w-5 h-5'>🤖</span>,
      label: 'AI Action',
      children: 'Mark contact as Do Not Contact',
      isCompleted: true,
      isDelete: false
    },
  },
  {
    id: AI_ACTION_NODE_ID.OUT_OF_OFFICE,
    type: 'custom-node',
    position: {
      x: INTENT_NODE_POS[INTENT_NODE_ID.OUT_OF_OFFICE].x + NODE_WIDTH + NODE_GAP,
      y: INTENT_NODE_POS[INTENT_NODE_ID.OUT_OF_OFFICE].y - 0.5,
    },
    data: {
      icon: <span className='w-5 h-5'>🤖</span>,
      label: 'AI Action',
      children: 'Schedule follow-up after the extracted date',
      isCompleted: true,
      isDelete: false
    },
  },
  {
    id: AI_ACTION_NODE_ID.UNKNOWN,
    type: 'custom-node',
    position: {
      x: INTENT_NODE_POS[INTENT_NODE_ID.UNKNOWN].x + NODE_WIDTH + NODE_GAP,
      y: INTENT_NODE_POS[INTENT_NODE_ID.UNKNOWN].y - 11,
    },
    data: {
      icon: <span className='w-5 h-5'>🤖</span>,
      label: 'AI Action',
      children: 'Route to campaign owner',
      isCompleted: true,
      isDelete: false
    },
  },
];

type IntentEdgeType = Edge & { pathOptions?: SmoothStepPathOptions };

export const afterOutreachEmailEdges: IntentEdgeType[] = [
  {
    id: `${CAMPAIGN_NODE_ID.OUTREACH_EMAIL}->${CAMPAIGN_NODE_ID.CLASSIFY}`,
    source: CAMPAIGN_NODE_ID.OUTREACH_EMAIL,
    target: CAMPAIGN_NODE_ID.CLASSIFY,
  },
  {
    id: `${CAMPAIGN_NODE_ID.CLASSIFY}->${INTENT_NODE_ID.INTERESTED}`,
    source: CAMPAIGN_NODE_ID.CLASSIFY,
    target: INTENT_NODE_ID.INTERESTED,
    type: 'smoothstep',
    pathOptions: {
      borderRadius: 12,
    },
  },
  {
    id: `${CAMPAIGN_NODE_ID.CLASSIFY}->${INTENT_NODE_ID.NOT_INTERESTED}`,
    source: CAMPAIGN_NODE_ID.CLASSIFY,
    target: INTENT_NODE_ID.NOT_INTERESTED,
    type: 'smoothstep',
    pathOptions: {
      borderRadius: 12,
    },
  },
  {
    id: `${CAMPAIGN_NODE_ID.CLASSIFY}->${INTENT_NODE_ID.PRICING}`,
    source: CAMPAIGN_NODE_ID.CLASSIFY,
    target: INTENT_NODE_ID.PRICING,
    type: 'smoothstep',
    pathOptions: {
      borderRadius: 12,
    },
  },
  {
    id: `${CAMPAIGN_NODE_ID.CLASSIFY}->${INTENT_NODE_ID.INFORMATION}`,
    source: CAMPAIGN_NODE_ID.CLASSIFY,
    target: INTENT_NODE_ID.INFORMATION,
    type: 'smoothstep',
    pathOptions: {
      borderRadius: 12,
    },
  },
  {
    id: `${CAMPAIGN_NODE_ID.CLASSIFY}->${INTENT_NODE_ID.DO_NOT_CONTACT}`,
    source: CAMPAIGN_NODE_ID.CLASSIFY,
    target: INTENT_NODE_ID.DO_NOT_CONTACT,
    type: 'smoothstep',
    pathOptions: {
      borderRadius: 12,
    },
  },
  {
    id: `${CAMPAIGN_NODE_ID.CLASSIFY}->${INTENT_NODE_ID.OUT_OF_OFFICE}`,
    source: CAMPAIGN_NODE_ID.CLASSIFY,
    target: INTENT_NODE_ID.OUT_OF_OFFICE,
    type: 'smoothstep',
    pathOptions: {
      borderRadius: 12,
    },
  },
  {
    id: `${CAMPAIGN_NODE_ID.CLASSIFY}->${INTENT_NODE_ID.UNKNOWN}`,
    source: CAMPAIGN_NODE_ID.CLASSIFY,
    target: INTENT_NODE_ID.UNKNOWN,
    type: 'smoothstep',
    pathOptions: {
      borderRadius: 12,
    },
  },
  {
    id: `${INTENT_NODE_ID.DO_NOT_CONTACT}->${AI_ACTION_NODE_ID.DO_NOT_CONTACT}`,
    source: INTENT_NODE_ID.DO_NOT_CONTACT,
    target: AI_ACTION_NODE_ID.DO_NOT_CONTACT,
    type: 'smoothstep',
    pathOptions: {
      borderRadius: 12,
    },
  },
  {
    id: `${INTENT_NODE_ID.OUT_OF_OFFICE}->${AI_ACTION_NODE_ID.OUT_OF_OFFICE}`,
    source: INTENT_NODE_ID.OUT_OF_OFFICE,
    target: AI_ACTION_NODE_ID.OUT_OF_OFFICE,
    type: 'smoothstep',
    pathOptions: {
      borderRadius: 12,
    },
  },
  {
    id: `${INTENT_NODE_ID.UNKNOWN}->${AI_ACTION_NODE_ID.UNKNOWN}`,
    source: INTENT_NODE_ID.UNKNOWN,
    target: AI_ACTION_NODE_ID.UNKNOWN,
    type: 'smoothstep',
    pathOptions: {
      borderRadius: 12,
    },
  },
];
