'use client';

import {
  Card,
  CardContent,
  CardHeader,
  CardTitle,
  CardDescription,
} from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Textarea } from '@/components/ui/textarea';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Mail,
  MessageSquare,
  Phone,
  Clock,
  GitBranch,
  Plus,
  Trash2,
  MoveUp,
  MoveDown,
  Sparkles,
} from 'lucide-react';
import { useState } from 'react';
import { toast } from 'sonner';
import PlaybookApi, {
  type CreatePlaybookRequest,
} from '@/network/client/playbook';

interface PlaybookStage {
  id: string;
  order: number;
  name: string;
  type: 'email' | 'linkedin' | 'call' | 'wait' | 'conditional';
  trigger_conditions: {
    wait_duration: number; // days
    wait_for_event?: string;
  };
  content_config?: {
    template?: string;
    ai_generation_prompt?: string;
  };
  success_criteria: {
    on_reply: 'advance' | 'complete';
    on_timeout: 'next_stage' | 'pause';
  };
}

interface PlaybookConfig {
  name: string;
  description: string;
  playbook_type: 'nurture' | 'outreach' | 're_engagement' | 'upsell';
  stages: PlaybookStage[];
}

const STAGE_ICONS = {
  email: Mail,
  linkedin: MessageSquare,
  call: Phone,
  wait: Clock,
  conditional: GitBranch,
};

const STAGE_COLORS = {
  email: 'border-blue-200 bg-blue-50 dark:border-blue-800 dark:bg-blue-950',
  linkedin:
    'border-purple-200 bg-purple-50 dark:border-purple-800 dark:bg-purple-950',
  call: 'border-green-200 bg-green-50 dark:border-green-800 dark:bg-green-950',
  wait: 'border-gray-200 bg-gray-50 dark:border-gray-800 dark:bg-gray-950',
  conditional:
    'border-yellow-200 bg-yellow-50 dark:border-yellow-800 dark:bg-yellow-950',
};

export function PlaybookBuilder() {
  const [playbook, setPlaybook] = useState<PlaybookConfig>({
    name: '',
    description: '',
    playbook_type: 'outreach',
    stages: [],
  });

  const [editingStage, setEditingStage] = useState<PlaybookStage | null>(null);
  const [isSaving, setIsSaving] = useState(false);

  const addStage = (type: PlaybookStage['type']) => {
    const newStage: PlaybookStage = {
      id: `stage_${Date.now()}`,
      order: playbook.stages.length + 1,
      name: `${type.charAt(0).toUpperCase() + type.slice(1)} Stage ${playbook.stages.length + 1}`,
      type,
      trigger_conditions: {
        wait_duration: type === 'wait' ? 2 : 0,
      },
      content_config:
        type === 'email' || type === 'linkedin'
          ? {
              template: '',
              ai_generation_prompt: '',
            }
          : undefined,
      success_criteria: {
        on_reply: 'advance',
        on_timeout: 'next_stage',
      },
    };

    setPlaybook({
      ...playbook,
      stages: [...playbook.stages, newStage],
    });

    setEditingStage(newStage);
  };

  const removeStage = (stageId: string) => {
    setPlaybook({
      ...playbook,
      stages: playbook.stages
        .filter((s) => s.id !== stageId)
        .map((s, idx) => ({ ...s, order: idx + 1 })),
    });
    if (editingStage?.id === stageId) {
      setEditingStage(null);
    }
  };

  const moveStage = (stageId: string, direction: 'up' | 'down') => {
    const index = playbook.stages.findIndex((s) => s.id === stageId);
    if (
      (direction === 'up' && index === 0) ||
      (direction === 'down' && index === playbook.stages.length - 1)
    ) {
      return;
    }

    const newStages = [...playbook.stages];
    const swapIndex = direction === 'up' ? index - 1 : index + 1;
    [newStages[index], newStages[swapIndex]] = [
      newStages[swapIndex],
      newStages[index],
    ];

    setPlaybook({
      ...playbook,
      stages: newStages.map((s, idx) => ({ ...s, order: idx + 1 })),
    });
  };

  const updateStage = (stageId: string, updates: Partial<PlaybookStage>) => {
    setPlaybook({
      ...playbook,
      stages: playbook.stages.map((s) =>
        s.id === stageId ? { ...s, ...updates } : s
      ),
    });

    if (editingStage?.id === stageId) {
      setEditingStage({ ...editingStage, ...updates });
    }
  };

  const savePlaybook = async () => {
    if (!playbook.name) {
      toast.error('Please enter a playbook name');
      return;
    }
    if (playbook.stages.length === 0) {
      toast.error('Please add at least one stage');
      return;
    }

    setIsSaving(true);
    try {
      const payload: CreatePlaybookRequest = {
        name: playbook.name.trim(),
        description: playbook.description.trim(),
        playbook_type: playbook.playbook_type,
        stages: playbook.stages,
      };
      const response = await PlaybookApi.create(payload);
      toast.success(`Playbook saved: ${response.data.name}`);
    } catch (error: any) {
      toast.error(error?.message || 'Failed to save playbook');
    } finally {
      setIsSaving(false);
    }
  };

  const generateAIContent = (stageId: string) => {
    // TODO: Call AI API to generate content
    toast.info('AI content generation coming soon...');
  };

  return (
    <div className="grid gap-6 lg:grid-cols-3">
      {/* Left Panel - Playbook Config */}
      <Card className="lg:col-span-1">
        <CardHeader>
          <CardTitle>Playbook Settings</CardTitle>
          <CardDescription>Configure your automated campaign</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="playbook-name">Playbook Name</Label>
            <Input
              id="playbook-name"
              placeholder="Enterprise Outreach Q1"
              value={playbook.name}
              onChange={(e) =>
                setPlaybook({ ...playbook, name: e.target.value })
              }
            />
          </div>

          <div className="space-y-2">
            <Label htmlFor="playbook-type">Type</Label>
            <Select
              value={playbook.playbook_type}
              onValueChange={(value: any) =>
                setPlaybook({ ...playbook, playbook_type: value })
              }
            >
              <SelectTrigger id="playbook-type">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="outreach">Cold Outreach</SelectItem>
                <SelectItem value="nurture">Nurture Campaign</SelectItem>
                <SelectItem value="re_engagement">Re-engagement</SelectItem>
                <SelectItem value="upsell">Upsell/Cross-sell</SelectItem>
              </SelectContent>
            </Select>
          </div>

          <div className="space-y-2">
            <Label htmlFor="playbook-description">Description</Label>
            <Textarea
              id="playbook-description"
              placeholder="Target high-fit enterprise prospects with personalized outreach..."
              value={playbook.description}
              onChange={(e) =>
                setPlaybook({ ...playbook, description: e.target.value })
              }
              rows={3}
            />
          </div>

          <Separator />

          <div className="space-y-2">
            <Label>Add Stage</Label>
            <div className="grid grid-cols-2 gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => addStage('email')}
                className="gap-2"
              >
                <Mail className="h-4 w-4" />
                Email
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => addStage('linkedin')}
                className="gap-2"
              >
                <MessageSquare className="h-4 w-4" />
                LinkedIn
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => addStage('call')}
                className="gap-2"
              >
                <Phone className="h-4 w-4" />
                Call
              </Button>
              <Button
                variant="outline"
                size="sm"
                onClick={() => addStage('wait')}
                className="gap-2"
              >
                <Clock className="h-4 w-4" />
                Wait
              </Button>
            </div>
          </div>

          <Separator />

          <Button onClick={savePlaybook} className="w-full" disabled={isSaving}>
            Save Playbook
          </Button>
        </CardContent>
      </Card>

      {/* Middle Panel - Stage Timeline */}
      <Card className="lg:col-span-1">
        <CardHeader>
          <CardTitle>Campaign Stages ({playbook.stages.length})</CardTitle>
          <CardDescription>Click a stage to edit details</CardDescription>
        </CardHeader>
        <CardContent>
          {playbook.stages.length === 0 ? (
            <div className="flex flex-col items-center justify-center py-8 text-center">
              <Plus className="mb-2 h-12 w-12 text-muted-foreground" />
              <p className="text-sm text-muted-foreground">
                No stages yet. Add your first stage from the left panel.
              </p>
            </div>
          ) : (
            <div className="space-y-3">
              {playbook.stages.map((stage, idx) => {
                const Icon = STAGE_ICONS[stage.type];
                const colorClass = STAGE_COLORS[stage.type];

                return (
                  <div key={stage.id} className="space-y-2">
                    <div
                      className={`cursor-pointer rounded-lg border p-3 transition-all hover:shadow-md ${colorClass} ${
                        editingStage?.id === stage.id
                          ? 'ring-2 ring-primary'
                          : ''
                      }`}
                      onClick={() => setEditingStage(stage)}
                    >
                      <div className="mb-2 flex items-start justify-between">
                        <div className="flex items-center gap-2">
                          <Badge variant="secondary" className="h-6 w-6 p-0">
                            {idx + 1}
                          </Badge>
                          <Icon className="h-4 w-4" />
                          <span className="font-medium">{stage.name}</span>
                        </div>
                        <div className="flex gap-1">
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6"
                            onClick={(e) => {
                              e.stopPropagation();
                              moveStage(stage.id, 'up');
                            }}
                            disabled={idx === 0}
                          >
                            <MoveUp className="h-3 w-3" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6"
                            onClick={(e) => {
                              e.stopPropagation();
                              moveStage(stage.id, 'down');
                            }}
                            disabled={idx === playbook.stages.length - 1}
                          >
                            <MoveDown className="h-3 w-3" />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            className="h-6 w-6"
                            onClick={(e) => {
                              e.stopPropagation();
                              removeStage(stage.id);
                            }}
                          >
                            <Trash2 className="h-3 w-3" />
                          </Button>
                        </div>
                      </div>

                      <p className="text-xs text-muted-foreground">
                        {stage.type === 'wait'
                          ? `Wait ${stage.trigger_conditions.wait_duration} days`
                          : stage.type === 'email'
                            ? 'Send personalized email'
                            : stage.type === 'linkedin'
                              ? 'Send LinkedIn message'
                              : stage.type === 'call'
                                ? 'Schedule call'
                                : 'Conditional logic'}
                      </p>
                    </div>

                    {idx < playbook.stages.length - 1 && (
                      <div className="flex justify-center">
                        <div className="h-4 w-px bg-border" />
                      </div>
                    )}
                  </div>
                );
              })}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Right Panel - Stage Editor */}
      <Card className="lg:col-span-1">
        <CardHeader>
          <CardTitle>Stage Details</CardTitle>
          <CardDescription>
            {editingStage
              ? `Editing: ${editingStage.name}`
              : 'Select a stage to edit'}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {!editingStage ? (
            <div className="flex flex-col items-center justify-center py-8 text-center">
              <GitBranch className="mb-2 h-12 w-12 text-muted-foreground" />
              <p className="text-sm text-muted-foreground">
                Click a stage from the timeline to edit its details
              </p>
            </div>
          ) : (
            <div className="space-y-4">
              <div className="space-y-2">
                <Label>Stage Name</Label>
                <Input
                  value={editingStage.name}
                  onChange={(e) =>
                    updateStage(editingStage.id, { name: e.target.value })
                  }
                />
              </div>

              <div className="space-y-2">
                <Label>Wait Duration (days)</Label>
                <Input
                  type="number"
                  min="0"
                  value={editingStage.trigger_conditions.wait_duration}
                  onChange={(e) =>
                    updateStage(editingStage.id, {
                      trigger_conditions: {
                        ...editingStage.trigger_conditions,
                        wait_duration: parseInt(e.target.value) || 0,
                      },
                    })
                  }
                />
                <p className="text-xs text-muted-foreground">
                  Delay before executing this stage
                </p>
              </div>

              {(editingStage.type === 'email' ||
                editingStage.type === 'linkedin') && (
                <>
                  <Separator />
                  <div className="space-y-2">
                    <div className="flex items-center justify-between">
                      <Label>Message Template</Label>
                      <Button
                        variant="outline"
                        size="sm"
                        onClick={() => generateAIContent(editingStage.id)}
                        className="gap-2"
                      >
                        <Sparkles className="h-3 w-3" />
                        AI Generate
                      </Button>
                    </div>
                    <Textarea
                      placeholder="Hi {{first_name}}, I noticed {{company}} is {{recent_signal}}..."
                      value={editingStage.content_config?.template || ''}
                      onChange={(e) =>
                        updateStage(editingStage.id, {
                          content_config: {
                            ...editingStage.content_config,
                            template: e.target.value,
                          },
                        })
                      }
                      rows={6}
                    />
                    <p className="text-xs text-muted-foreground">
                      Use variables: {'{'}
                      {'{'}first_name{'}'}
                      {'}'}, {'{'}
                      {'{'}company{'}'}
                      {'}'}, {'{'}
                      {'{'}pain_point{'}'}
                      {'}'}
                    </p>
                  </div>

                  <div className="space-y-2">
                    <Label>AI Generation Prompt</Label>
                    <Textarea
                      placeholder="Generate cold outreach email emphasizing time savings..."
                      value={
                        editingStage.content_config?.ai_generation_prompt || ''
                      }
                      onChange={(e) =>
                        updateStage(editingStage.id, {
                          content_config: {
                            ...editingStage.content_config,
                            ai_generation_prompt: e.target.value,
                          },
                        })
                      }
                      rows={3}
                    />
                  </div>
                </>
              )}

              <Separator />

              <div className="space-y-2">
                <Label>On Reply</Label>
                <Select
                  value={editingStage.success_criteria.on_reply}
                  onValueChange={(value: any) =>
                    updateStage(editingStage.id, {
                      success_criteria: {
                        ...editingStage.success_criteria,
                        on_reply: value,
                      },
                    })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="advance">
                      Advance to next stage
                    </SelectItem>
                    <SelectItem value="complete">Complete playbook</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label>On Timeout</Label>
                <Select
                  value={editingStage.success_criteria.on_timeout}
                  onValueChange={(value: any) =>
                    updateStage(editingStage.id, {
                      success_criteria: {
                        ...editingStage.success_criteria,
                        on_timeout: value,
                      },
                    })
                  }
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="next_stage">
                      Continue to next stage
                    </SelectItem>
                    <SelectItem value="pause">Pause playbook</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </div>
          )}
        </CardContent>
      </Card>
    </div>
  );
}
