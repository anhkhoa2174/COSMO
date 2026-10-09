'use client';

import {
  addEdge,
  Background,
  BackgroundVariant,
  Controls,
  getOutgoers,
  MarkerType,
  MiniMap,
  ReactFlow,
  ReactFlowProvider,
  useEdgesState,
  useNodesState,
  type Edge,
  type OnConnect,
} from '@xyflow/react';
import React, { useCallback, useEffect, useRef, useState } from 'react';

import '@xyflow/react/dist/style.css';

import { BackButton } from '@/components/buttons/back-button';
import { MainButton } from '@/components/buttons/main-button';
import { Editable } from '@/components/editable';
import { ContentLayout } from '@/components/nav/content-layout';
import DescriptionText from '@/components/tour/DescriptionText';
import WelcomeDialog from '@/components/tour/WelcomeDialog';
import { Button } from '@/components/ui/button';
import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from '@/components/ui/sheet';
import { useDisclosure } from '@/hooks/use-disclosure';
import { UpdateOnboarding } from '@/hooks/use-tour';
import { cn, merge } from '@/lib/utils';
import CampaignApi, { useGetCampaignQuery } from '@/network/client/campaign';
import { useUser } from '@/hooks/use-user';
import classes from '@/styles/Campaign.module.css';
import { VisuallyHidden } from '@radix-ui/react-visually-hidden';
import { Bolt, Eye, Inbox, Mails, MessageCircle, Sparkles } from 'lucide-react';
import { toast } from 'sonner';
import AIReplyV2 from './_components/actions/ai-reply-v2';
import AssignPerson from './_components/actions/assign-person';
import DraftEmail from './_components/actions/draft-email';
import ActivateButton from './_components/activate-button';
import AgentSelector from './_components/agent-selector';
import ContactListSelector from './_components/contact-list-selector';
import { edgeTypes } from './_components/edges';
import { AppNode, nodeTypes, TemplateNode } from './_components/nodes';
import ScenarioSimulation from './_components/scenario-simulation';
import Settings from './_components/settings';
import TemplateComposer from './_components/template-composer';
import {
  ACTION_NODE_ID,
  ACTION_NODE_IDS,
  actionNodeData,
  afterOutreachEmailEdges,
  afterOutreachEmailNodes,
  CAMPAIGN_NODE_ID,
  getActionNodeIdByType,
  getIntentNodeIdByType,
  initialNodes,
  INTENT_NODE_IDS,
  INTENT_NODE_POS,
  INTENT_TYPE,
  NODE_GAP,
  NODE_WIDTH,
  X_OFFSET,
  Y_OFFSET,
} from './data';
import { useCampaign, useCampaignSupport } from './use-campaign';

const initialEdges: Edge[] = [];
const outreachNode: AppNode = {
  id: CAMPAIGN_NODE_ID.OUTREACH_EMAIL,
  type: 'custom-node',
  position: { x: X_OFFSET + NODE_WIDTH + NODE_GAP, y: Y_OFFSET },
  data: {
    icon: <Mails className="h-5 w-5" />,
    label: 'Outreach Email',
    children: <p>Includes first and list follow-up email</p>,
  },
};
const outreachEdge = {
  id: `${CAMPAIGN_NODE_ID.ENTRY_RULES}->${CAMPAIGN_NODE_ID.OUTREACH_EMAIL}`,
  source: CAMPAIGN_NODE_ID.ENTRY_RULES,
  target: CAMPAIGN_NODE_ID.OUTREACH_EMAIL,
};

export default function CampaignDetailPage({
  params,
}: {
  params: { campaign_id: string };
}) {
  const [nodes, setNodes, onNodesChange] = useNodesState(initialNodes);
  const [edges, setEdges, onEdgesChange] = useEdgesState(initialEdges);
  const [campaign, setCampaign] = useCampaign();
  const [campaignSupport, setCampaignSupport] = useCampaignSupport();
  const statusCampaign = useRef(campaign.status);
  const [isTourOpen, setIsTourOpen] = useState(false);
  const { user, invalidate } = useUser();

  const handleExplore = async () => {
    setIsTourOpen(false);
    await UpdateOnboarding(
      {
        campaigns: {
          onboarding: false,
          step: 0,
          status: 'completed',
        },
      },
      user
    );
    await invalidate?.();
  };

  useEffect(() => {
    if (user?.ui_metadata?.campaigns?.onboarding) {
      setIsTourOpen(true);
    }
  }, []);

  useEffect(() => {
    statusCampaign.current = campaign.status;
  }, [campaign.status]);

  const {
    data: getCampaignData,
    isLoading: isGetCampaignLoading,
    isSuccess,
  } = useGetCampaignQuery(params.campaign_id);

  useEffect(() => {
    if (isSuccess) {
      const data = getCampaignData.data;
      setCampaign(data);
      setCampaignSupport((prev) => ({
        ...prev,
        templates: data.templates.reduce((acc: any, template) => {
          acc[template.id] = null;
          return acc;
        }, {}),
        draftTemplates: data.draft_templates.reduce((acc: any, template) => {
          acc[template.id] = null;
          return acc;
        }, {}),
      }));
      if (data.list_contact_id) {
        updateNodeData(CAMPAIGN_NODE_ID.ENTRY_RULES, 'isCompleted', true);
        renderOutreachNode();
        renderTemplateNodes(data.templates);
        addAddTemplateNodeButton(data.templates);
        if (data.templates.length > 0) {
          setNodes((nds) => merge(nds, afterOutreachEmailNodes));
          setEdges((eds) => merge(eds, afterOutreachEmailEdges));
        }
      }
      if (data.cmetadata.config) {
        data.cmetadata.config.forEach((config: any) => {
          const intentNodeId = getIntentNodeIdByType(config.intent_type);
          if (intentNodeId) {
            const actionNodeId = getActionNodeIdByType(config.who);
            if (actionNodeId) {
              handleOnClickAction(intentNodeId, actionNodeId, undefined, data);
              updateNodeData(
                `${intentNodeId}-${actionNodeId}`,
                'isCompleted',
                true
              );
            }
          }
        });
      }
    }
  }, [isSuccess]);

  function updateCampaign(campaignId: string, data: any) {
    const myPromise = async () => {
      return await CampaignApi.update(campaignId, data);
    };

    toast.promise(myPromise, {
      loading: 'Updating...',
      success: () => {
        setCampaign((prev) => ({ ...prev, name: data.name }));
        return 'Update campaign successfully';
      },
      error: 'Failed to update campaign',
    });
  }

  /*  workflow manager */

  const isValid = (isAnotherCase: boolean = false): boolean => {
    if (isAnotherCase) {
      if (statusCampaign.current === 'active') {
        toast.warning('Cannot delete nodes while workflow is activated');
        return false;
      }
      return true;
    } else {
      if (statusCampaign.current !== 'draft') {
        toast.warning('Cannot delete nodes while workflow is activated');
        return false;
      }
      return true;
    }
  };

  function deleteNode(nodeId: string): void {
    setNodes((prev) => prev.filter((node) => node.id !== nodeId));
    setEdges((prev) =>
      prev.filter((edge) => edge.source !== nodeId && edge.target !== nodeId)
    );
  }

  /* node utils */
  const addTemplateNode = (
    data: Partial<TemplateNode['data']>,
    pos: number,
    template_id: string
  ) => {
    const nodeId = `template-${template_id}`;
    const templateNode: AppNode = {
      id: nodeId,
      type: 'template-node',
      position: {
        x: X_OFFSET + 1.5 * NODE_WIDTH + NODE_GAP,
        y: Y_OFFSET + (pos + 1) * 112,
      },
      data: {
        ...data,
        label: `Wait a few days`,
        children: pos > 0 ? <p>Follow-up Email {pos}</p> : <p>First Email</p>,
      },
    };
    const templateEdge = {
      id: `${CAMPAIGN_NODE_ID.OUTREACH_EMAIL}->${nodeId}`,
      source: CAMPAIGN_NODE_ID.OUTREACH_EMAIL,
      target: nodeId,
      type: 'flow',
      sourceHandle: 'custom-node-handle-bottom',
      pathOptions: {
        borderRadius: 12,
      },
    };
    setNodes((nds) => [...nds, templateNode]);
    setEdges((eds) => [...eds, templateEdge]);
  };

  const addAddTemplateNodeButton = (templates: any[]) => {
    const addTemplateNode: AppNode = {
      id: 'add-template-node',
      type: 'default',
      position: {
        x: X_OFFSET + 1.5 * NODE_WIDTH + NODE_GAP,
        y: Y_OFFSET + (templates.length + 1) * 112,
      },
      data: {
        label: '+ Add Email',
      },
      style: {
        background: 'var(--mantine-color-indigo-0)',
        color: 'var(--mantine-color-indigo-9)',
        border: 'none',
        borderRadius: 8,
        width: 96,
        fontSize: 14,
      },
    };
    setNodes((nds) => [...nds, addTemplateNode]);
  };

  const renderOutreachNode = () => {
    setNodes((nds) => [...nds, outreachNode]);
    setEdges((eds) => [...eds, outreachEdge]);
    updateNodeData(CAMPAIGN_NODE_ID.OUTREACH_EMAIL, 'isCompleted', true);
  };

  const renderTemplateNodes = (templates: any[]) => {
    templates.forEach((template, index) =>
      addTemplateNode({ isCompleted: true }, index, template.id)
    );
  };

  const addNewTemplateNode = () => {
    openSheet({
      size: 'xl',
      content: (
        <TemplateComposer
          templateId=""
          templateIndex={campaign.templates.length}
          closeSheet={closeSheet}
          onCreateSuccess={(template_id) => {
            if (campaign.templates.length === 0) {
              updateNodeData(
                CAMPAIGN_NODE_ID.OUTREACH_EMAIL,
                'isCompleted',
                true
              );
              setNodes((nds) => merge(nds, afterOutreachEmailNodes));
              setEdges((eds) => merge(eds, afterOutreachEmailEdges));
            }
            addTemplateNode(
              { isCompleted: true },
              campaign.templates.length,
              template_id
            );
            updateNode('add-template-node', 'position', {
              x: X_OFFSET + 1.5 * NODE_WIDTH + NODE_GAP,
              y: Y_OFFSET + (campaign.templates.length + 2) * 112,
            });
          }}
        />
      ),
    });
  };

  const onConnect: OnConnect = useCallback(
    (_params) => setEdges((eds) => addEdge(_params, eds)),
    [setEdges]
  );

  const {
    open: openSheet,
    close: closeSheet,
    opened: sheetOpened,
    size: sheetSize,
    content: sheetContent,
  } = useSheet();
  const [activeNodeId, setActiveNodeId] = useState<string | null>(null);

  const handleCloseSheet = (nodeId?: string) => {
    const closing = nodeId ?? activeNodeId;
    if (closing) {
      updateNode(closing, 'selected', false);
      markJustEdited(closing);
    }
    setActiveNodeId(null);
    closeSheet();
  };

  // Flag the node the panel was open on, and clear the flag on every other
  // one, so exactly one card is marked at a time. It fades after a few
  // seconds — long enough to catch the eye on a canvas of similar cards,
  // short enough not to become permanent decoration.
  const justEditedTimer = useRef<ReturnType<typeof setTimeout> | null>(null);

  const markJustEdited = (nodeId: string) => {
    if (justEditedTimer.current) clearTimeout(justEditedTimer.current);
    setNodes((nds) =>
      nds.map((nd) =>
        ({ ...nd, data: { ...nd.data, justEdited: nd.id === nodeId } }) as AppNode
      )
    );
    // Clear by id rather than by reading the flag back: node data is a union
    // and not every member of it carries the field.
    justEditedTimer.current = setTimeout(() => {
      setNodes((nds) =>
        nds.map((nd) =>
          nd.id === nodeId
            ? ({ ...nd, data: { ...nd.data, justEdited: false } } as AppNode)
            : nd
        )
      );
    }, 6000);
  };

  useEffect(() => {
    return () => {
      if (justEditedTimer.current) clearTimeout(justEditedTimer.current);
    };
  }, []);

  const updateNode = (id: string, key: string, value: any) => {
    setNodes((nds) =>
      nds.map((nd) => (nd.id === id ? { ...nd, [key]: value } : nd))
    );
  };

  const updateNodeData = (id: string, key: string, value: any) => {
    setNodes((nds) =>
      nds.map((nd) =>
        nd.id === id
          ? ({ ...nd, data: { ...nd.data, [key]: value } } as AppNode)
          : nd
      )
    );
  };

  const removeActionHoldNode = () => {
    setNodes((nds) =>
      nds.filter((node) => node.id !== CAMPAIGN_NODE_ID.ACTION_HOLD)
    );
    setEdges((eds) =>
      eds.filter((edge) => edge.id !== 'edge-with-action-hold')
    );
  };

  const handleOnClickAction = (
    intentNodeId: string,
    actionId: string,
    callback?: (a: string, b: string) => void,
    campaign?: any
  ) => {
    const actionNode: AppNode = {
      id: `${intentNodeId}-${actionId}`,
      type: 'custom-node',
      data: {
        ...actionNodeData[actionId],
        deleteNode: async (nodeId) => {
          if (!isValid()) {
            return;
          }
          const copy = [...(campaign.cmetadata.config || [])];
          const payload = {
            config: copy.filter(
              (c) => c.intent_type !== INTENT_TYPE[intentNodeId]
            ),
          };
          await CampaignApi.assign(campaign.id, payload);
          setCampaign((prev) => ({
            ...prev,
            cmetadata: { ...prev.cmetadata, ...payload },
          }));
          deleteNode(nodeId);
        },
      },
      position: {
        x: INTENT_NODE_POS[intentNodeId].x + NODE_WIDTH + NODE_GAP,
        y: INTENT_NODE_POS[intentNodeId].y - 60,
      },
    };
    const edge: Edge = {
      id: `${intentNodeId}->${actionId}`,
      source: intentNodeId,
      target: `${intentNodeId}-${actionId}`,
    };
    setEdges((eds) => addEdge(edge, eds));
    setNodes((nds) => [...nds, actionNode]);

    callback?.(intentNodeId, actionId);
  };

  const handleAfterCreateActionNode = (
    intentNodeId: string,
    actionId: string
  ) => {
    removeActionHoldNode();

    updateNode(`${intentNodeId}-${actionId}`, 'selected', true);
    setActiveNodeId(`${intentNodeId}-${actionId}`);
    if (actionId === ACTION_NODE_ID.AI_REPLY) {
      openSheet({
        size: 'lg',
        content: (
          <AIReplyV2
            intent={INTENT_TYPE[intentNodeId]}
            closeSheet={() => handleCloseSheet(`${intentNodeId}-${actionId}`)}
            onSuccess={() =>
              updateNodeData(`${intentNodeId}-${actionId}`, 'isCompleted', true)
            }
          />
        ),
      });
    }
    if (actionId === ACTION_NODE_ID.DRAFT_EMAIL) {
      openSheet({
        size: 'xl',
        content: (
          <DraftEmail
            intentType={INTENT_TYPE[intentNodeId]}
            closeSheet={() => handleCloseSheet(`${intentNodeId}-${actionId}`)}
            onSuccess={() =>
              updateNodeData(`${intentNodeId}-${actionId}`, 'isCompleted', true)
            }
          />
        ),
      });
    }
    if (actionId === ACTION_NODE_ID.ASSIGN_PERSON) {
      openSheet({
        content: (
          <AssignPerson
            intentType={INTENT_TYPE[intentNodeId]}
            closeSheet={() => handleCloseSheet(`${intentNodeId}-${actionId}`)}
            onSuccess={() =>
              updateNodeData(`${intentNodeId}-${actionId}`, 'isCompleted', true)
            }
          />
        ),
      });
    }
  };

  const onNodeClick = (node: AppNode) => {
    if (node.id === CAMPAIGN_NODE_ID.ENTRY_RULES) {
      setActiveNodeId(node.id);
      openSheet({
        content: (
          <ContactListSelector
            closeSheet={() => handleCloseSheet(node.id)}
            onSuccess={() => {
              updateNodeData(node.id, 'isCompleted', true);
              if (
                !nodes.find((nd) => nd.id === CAMPAIGN_NODE_ID.OUTREACH_EMAIL)
              ) {
                renderOutreachNode();
                addAddTemplateNodeButton(campaign?.templates || []);
              }
            }}
          />
        ),
      });
    }

    if (node.id.startsWith('template-')) {
      const templateId = node.id.split('template-')[1];
      const templateIndex = campaign.templates.findIndex(
        (template) => template.id === templateId
      );
      setActiveNodeId(node.id);
      openSheet({
        size: 'xl',
        content: (
          <TemplateComposer
            templateId={templateId}
            templateIndex={templateIndex}
            closeSheet={() => handleCloseSheet(node.id)}
          />
        ),
      });
    }

    if (node.id === 'add-template-node') {
      addNewTemplateNode();
    }

    if (!(node.id === CAMPAIGN_NODE_ID.ACTION_HOLD)) {
      removeActionHoldNode();
    }

    if (INTENT_NODE_IDS.includes(node.id)) {
      const outgoers = getOutgoers(node, nodes, edges);
      if (outgoers.length > 0) {
        return;
      }
      const actionHoldNode: AppNode = {
        id: CAMPAIGN_NODE_ID.ACTION_HOLD,
        type: 'action-node',
        data: {
          actions: ACTION_NODE_IDS.map(
            (actionId) => () =>
              handleOnClickAction(
                node.id,
                actionId,
                handleAfterCreateActionNode,
                campaign
              )
          ),
        },
        position: { x: node.position.x + 400, y: node.position.y - 71 },
      };
      const edge: Edge = {
        id: 'edge-with-action-hold',
        source: node.id,
        target: CAMPAIGN_NODE_ID.ACTION_HOLD,
      };
      setEdges((eds) => addEdge(edge, eds));
      setNodes((nds) => [...nds, actionHoldNode]);
    }

    if (node.id.includes(ACTION_NODE_ID.AI_REPLY)) {
      const intentNodeId = node.id.split('-')[0];
      setActiveNodeId(node.id);
      openSheet({
        size: 'lg',
        content: (
          <AIReplyV2
            intent={INTENT_TYPE[intentNodeId]}
            closeSheet={() => handleCloseSheet(node.id)}
            onSuccess={() => updateNodeData(node.id, 'isCompleted', true)}
          />
        ),
      });
    }

    if (node.id.includes(ACTION_NODE_ID.DRAFT_EMAIL)) {
      const intentNodeId = node.id.split('-')[0];
      setActiveNodeId(node.id);
      openSheet({
        size: 'xl',
        content: (
          <DraftEmail
            intentType={INTENT_TYPE[intentNodeId]}
            closeSheet={() => handleCloseSheet(node.id)}
            onSuccess={() => updateNodeData(node.id, 'isCompleted', true)}
          />
        ),
      });
    }

    if (node.id.includes(ACTION_NODE_ID.ASSIGN_PERSON)) {
      const intentNodeId = node.id.split('-')[0];
      setActiveNodeId(node.id);
      openSheet({
        content: (
          <AssignPerson
            intentType={INTENT_TYPE[intentNodeId]}
            closeSheet={() => handleCloseSheet(node.id)}
            onSuccess={() => updateNodeData(node.id, 'isCompleted', true)}
          />
        ),
      });
    }
  };

  const handleOpenScenarioSimulation = () => {
    openSheet({
      size: 'md',
      content: <ScenarioSimulation closeSheet={handleCloseSheet} />,
    });
  };

  const handleOpenAgentSelector = () => {
    openSheet({ content: <AgentSelector closeSheet={handleCloseSheet} /> });
  };

  const [visibleMiniMap, setVisibleMiniMap] = useState(true);
  const [showMiniMapButton, setShowMiniMapButton] = useState(false);
  const hideMiniMapTimeout = useRef<NodeJS.Timeout | null>(null);

  const handleFlowMouseMove = () => {
    setVisibleMiniMap(true);

    if (hideMiniMapTimeout.current) {
      clearTimeout(hideMiniMapTimeout.current);
    }
    hideMiniMapTimeout.current = setTimeout(() => {
      setVisibleMiniMap(false);
      hideMiniMapTimeout.current = setTimeout(() => {
        setShowMiniMapButton(false);
      }, 500);
    }, 1500);
  };

  useEffect(() => {
    return () => {
      if (hideMiniMapTimeout.current) clearTimeout(hideMiniMapTimeout.current);
    };
  }, []);

  const handleOpenSettings = () => {
    openSheet({ content: <Settings closeSheet={handleCloseSheet} /> });
  };

  return (
    <ContentLayout
      variant="editor"
      leftSection={<BackButton href="/campaigns" />}
      title={
        <Editable
          className="text-base font-semibold"
          value={campaign.name}
          onSubmit={(newValue: string) =>
            updateCampaign(campaign.id, { name: newValue })
          }
        />
      }
      rightSection={
        <div className="flex gap-2">
          <ActivateButton campaignId={campaign.id} />
          <Button onClick={handleOpenAgentSelector} variant="outline">
            <Inbox />
            AI Inbox
          </Button>
          <MainButton
            text="Preview"
            variant="outline"
            icon={Eye}
            onClick={handleOpenScenarioSimulation}
            disabled={campaign.templates.length < 1}
          />
          <MainButton
            text="Settings"
            variant="outline"
            icon={Bolt}
            onClick={handleOpenSettings}
          />
        </div>
      }
    >
      {/* Reply notification banner */}
      {(() => {
        const replyCount = Object.values(campaignSupport.preview).flat().length;
        const intentsWithReplies = Object.entries(
          campaignSupport.preview
        ).filter(([, emails]) => emails.length > 0);
        if (replyCount === 0 || campaign.status === 'draft') return null;
        return (
          <div className="mx-4 mb-1 mt-2 rounded-lg border border-green-500/20 bg-green-500/5 px-4 py-3">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-3">
                <div className="relative">
                  <MessageCircle className="h-5 w-5 text-green-400" />
                  <span className="absolute -right-1.5 -top-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-green-500 text-[0.55rem] font-bold text-black">
                    {replyCount}
                  </span>
                </div>
                <div>
                  <span className="text-sm font-medium text-[#EEEEF0]">
                    {replyCount} {replyCount === 1 ? 'reply' : 'replies'}{' '}
                    received
                  </span>
                  <div className="mt-0.5 flex gap-1.5">
                    {intentsWithReplies.map(([intent, emails]) => (
                      <span
                        key={intent}
                        className="text-[0.65rem] text-[#8B8B98]"
                      >
                        {intent}: {emails.length}
                      </span>
                    ))}
                  </div>
                </div>
              </div>
              <Button
                size="sm"
                variant="outline"
                className="border-green-500/30 text-green-400 hover:bg-green-500/10"
                onClick={handleOpenAgentSelector}
              >
                <Sparkles className="mr-1.5 h-3.5 w-3.5" />
                Review & AI Reply
              </Button>
            </div>
          </div>
        );
      })()}
      <div className={classes.flowContainer}>
        {!isGetCampaignLoading && (
          <ReactFlowProvider>
            <ReactFlow
              nodes={nodes}
              edges={edges}
              nodeTypes={nodeTypes}
              edgeTypes={edgeTypes}
              onNodesChange={onNodesChange}
              onEdgesChange={onEdgesChange}
              onConnect={onConnect}
              onNodeClick={(_, node) => onNodeClick(node)}
              onPaneClick={removeActionHoldNode}
              onMoveStart={() => setShowMiniMapButton(true)}
              onMoveEnd={handleFlowMouseMove}
              onNodeDrag={(_, node) => updateNode(node.id, 'selected', false)}
              onNodeDragStart={(_, node) =>
                updateNode(node.id, 'selected', false)
              }
              // nodesConnectable
              // fitView
              // nodesDraggable={false}
              deleteKeyCode={null}
              defaultEdgeOptions={{
                // Every edge gets the animated wire unless it opts out.
                type: 'flow',
                style: { stroke: '#D3D3D3', strokeWidth: 2 },
                markerEnd: {
                  type: MarkerType.Arrow,
                  // Matches the violet end of the wire gradient.
                  color: '#8b5cf6',
                },
              }}
            >
              <Controls orientation="horizontal" />
              <Background
                variant={BackgroundVariant.Lines}
                gap={192}
                lineWidth={2}
                color="var(--mantine-color-gray-0)"
              />
              {showMiniMapButton && (
                <MiniMap
                  position="bottom-left"
                  className="-translate-y-10"
                  style={{
                    transition: 'opacity 0.5s',
                    opacity: visibleMiniMap ? 1 : 0,
                    boxShadow: '0 2px 8px rgba(0,0,0,0.10)',
                    borderRadius: 8,
                  }}
                />
              )}
            </ReactFlow>
          </ReactFlowProvider>
        )}
        <Sheet
          open={sheetOpened}
          onOpenChange={(open) => !open && handleCloseSheet()}
        >
          <SheetContent
            withCloseButton={false}
            className={cn('p-0', sheetSize)}
          >
            <VisuallyHidden>
              <SheetHeader>
                <SheetTitle />
                <SheetDescription />
              </SheetHeader>
            </VisuallyHidden>
            {sheetContent}
          </SheetContent>
        </Sheet>
      </div>
      <WelcomeDialog
        open={isTourOpen}
        onExplore={handleExplore}
        title="Start setting up campaign nodes"
        description={
          <DescriptionText
            title="Design Campaign Flow"
            description="Add message steps, delays, and triggers to build your AI campaign logic.."
          />
        }
        imageSrc=""
        textButton="Get Started"
      />
    </ContentLayout>
  );
}

const useSheet = () => {
  const breakpoint = {
    xs: 'w-[400px] sm:max-w-[400px]',
    sm: 'w-[640px] sm:max-w-[640px]',
    md: 'w-[768px] sm:max-w-[768px]',
    lg: 'w-[1024px] sm:max-w-[1024px]',
    xl: 'w-[1280px] sm:max-w-[1280px]',
    '2xl': 'w-[1536px] sm:max-w-[1536px]',
  };

  const [size, setSize] = useState<'xs' | 'sm' | 'md' | 'lg' | 'xl' | '2xl'>(
    'xs'
  );
  const [content, setContent] = useState<React.ReactNode | null>(null);
  const [opened, { open, close }] = useDisclosure();

  const handleClose = useCallback(() => {
    setContent(null);
    close();
  }, [close]);

  const handleOpen = useCallback(
    (params: {
      size?: typeof size;
      className?: string;
      content: React.ReactNode;
    }) => {
      setSize(params.size || 'xs');
      setContent(params.content);
      open();
    },
    [open]
  );

  return {
    opened,
    size: breakpoint[size],
    content,
    close: handleClose,
    open: handleOpen,
  };
};
