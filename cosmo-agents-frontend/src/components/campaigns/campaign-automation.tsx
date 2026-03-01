'use client';

import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Badge } from '@/components/ui/badge';
import { Separator } from '@/components/ui/separator';
import { Switch } from '@/components/ui/switch';
import { Label } from '@/components/ui/label';
import { Input } from '@/components/ui/input';
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select';
import {
  Play,
  Pause,
  Target,
  AlertCircle,
  CheckCircle2,
  Trash2,
  Users,
} from 'lucide-react';
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from '@/components/ui/alert-dialog';
import { useMemo, useState } from 'react';
import { toast } from 'sonner';
import { kyClient } from '@/lib/ky';
import {
  useAutomationRulesQuery,
  useCreateAutomationRuleMutation,
  useToggleAutomationRuleMutation,
  type AutomationRuleRead,
} from '@/network/client/automation-rule';
import SegmentationApi, { useSegmentationsQuery } from '@/network/client/segmentation';
import PlaybookApi from '@/network/client/playbook';
import { useQuery, useQueryClient, useMutation } from '@tanstack/react-query';

interface EnrollmentRequest {
  contact_id: string;
  contact_name: string;
  fit_score: number;
  engagement_score: number;
  reason: string;
  playbook_name: string;
}

export function CampaignAutomation() {
  const queryClient = useQueryClient();
  const { data: rulesResponse, isLoading: rulesLoading } =
    useAutomationRulesQuery();
  const rules = rulesResponse?.data || [];

  const { data: segmentResponse } = useSegmentationsQuery();
  const segmentOptions = segmentResponse?.data || [];

  const { data: playbookResponse } = useQuery({
    queryKey: ['playbooks'],
    queryFn: () => PlaybookApi.list(),
  });
  const playbookOptions = playbookResponse?.data || [];

  const createRuleMutation = useCreateAutomationRuleMutation();
  const toggleRuleMutation = useToggleAutomationRuleMutation();

  const deleteSegmentMutation = useMutation({
    mutationFn: (id: string) => SegmentationApi.delete(id),
    onSuccess: () => {
      toast.success('Segment deleted successfully');
      queryClient.invalidateQueries({ queryKey: ['segmentations'] });
    },
    onError: (error: any) => {
      toast.error(error?.message || 'Failed to delete segment');
    },
  });
  const [showCreate, setShowCreate] = useState(false);
  const [showCreateSegment, setShowCreateSegment] = useState(false);
  const [form, setForm] = useState({
    name: '',
    segment_id: '',
    playbook_id: '',
    fit_score_threshold: 70,
    engagement_score_threshold: '',
    require_human_approval: true,
    is_active: true,
  });
  const [segmentForm, setSegmentForm] = useState({
    name: '',
    description: '',
    priority: 5,
    is_active: true,
  });

  const pendingRequests: EnrollmentRequest[] = [];

  const toggleRule = (rule: AutomationRuleRead) => {
    toggleRuleMutation.mutate(
      { id: rule.automation_rule_id, is_active: !rule.is_active },
      {
        onSuccess: () => {
          toast.success(
            `Automation ${rule.is_active ? 'paused' : 'activated'}: ${rule.name}`
          );
          queryClient.invalidateQueries({ queryKey: ['automation-rules'] });
        },
        onError: (error: any) => {
          toast.error(error?.message || 'Failed to toggle automation rule');
        },
      }
    );
  };

  const approveEnrollment = (contactId: string) => {
    toast.success('Contact approved for enrollment');
    // TODO: Call API to approve enrollment
  };

  const rejectEnrollment = (contactId: string) => {
    toast.info('Contact enrollment rejected');
    // TODO: Call API to reject enrollment
  };

  const canSubmit = useMemo(() => {
    return form.name && form.segment_id && form.playbook_id;
  }, [form.name, form.segment_id, form.playbook_id]);

  const submitRule = () => {
    if (!canSubmit) {
      toast.error('Please fill all required fields');
      return;
    }
    const engagement =
      form.engagement_score_threshold === ''
        ? undefined
        : Number(form.engagement_score_threshold);
    createRuleMutation.mutate(
      {
        name: form.name.trim(),
        segment_id: form.segment_id,
        playbook_id: form.playbook_id,
        is_active: form.is_active,
        enrollment_criteria: {
          fit_score_threshold: Number(form.fit_score_threshold || 0),
          engagement_score_threshold: engagement,
          require_human_approval: form.require_human_approval,
        },
      },
      {
        onSuccess: () => {
          toast.success('Automation rule created');
          setShowCreate(false);
          setForm({
            name: '',
            segment_id: '',
            playbook_id: '',
            fit_score_threshold: 70,
            engagement_score_threshold: '',
            require_human_approval: true,
            is_active: true,
          });
          queryClient.invalidateQueries({ queryKey: ['automation-rules'] });
        },
        onError: (error: any) => {
          toast.error(error?.message || 'Failed to create automation rule');
        },
      }
    );
  };

  const submitSegment = async () => {
    if (!segmentForm.name.trim()) {
      toast.error('Segment name is required');
      return;
    }
    try {
      const payload = await kyClient
        .post('v1/segmentations', {
          json: {
            name: segmentForm.name.trim(),
            description: segmentForm.description.trim(),
            priority: Number(segmentForm.priority || 5),
            criteria: {},
            icp_definition: {},
            is_active: segmentForm.is_active,
          },
        })
        .json();
      toast.success('Segment created');
      setSegmentForm({ name: '', description: '', priority: 5, is_active: true });
      setShowCreateSegment(false);
      queryClient.invalidateQueries({ queryKey: ['segmentations'] });
    } catch (error: any) {
      toast.error(error?.message || 'Failed to create segment');
    }
  };

  return (
    <div className="space-y-6">
      {/* Segments Management */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Users className="h-5 w-5" />
            Segments
          </CardTitle>
          <CardDescription>
            Manage your contact segments for targeting
          </CardDescription>
        </CardHeader>
        <CardContent>
          {segmentOptions.length === 0 ? (
            <div className="rounded-lg border p-4 text-sm text-muted-foreground text-center">
              No segments created yet
            </div>
          ) : (
            <div className="space-y-2">
              {segmentOptions.map((segment) => (
                <div
                  key={segment.id}
                  className="flex items-center justify-between rounded-lg border p-3"
                >
                  <div className="flex items-center gap-3">
                    <div>
                      <h4 className="font-medium">{segment.name}</h4>
                      {segment.description && (
                        <p className="text-sm text-muted-foreground">
                          {segment.description}
                        </p>
                      )}
                    </div>
                    <Badge variant={segment.is_active ? 'default' : 'secondary'}>
                      {segment.is_active ? 'Active' : 'Inactive'}
                    </Badge>
                  </div>
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button
                        variant="ghost"
                        size="icon"
                        className="text-destructive hover:text-destructive hover:bg-destructive/10"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Delete Segment</AlertDialogTitle>
                        <AlertDialogDescription>
                          Are you sure you want to delete &quot;{segment.name}&quot;? This action cannot be undone. Contacts in this segment will be unassigned.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                          onClick={() => deleteSegmentMutation.mutate(segment.id)}
                          className="bg-destructive text-destructive-foreground hover:bg-destructive/90"
                        >
                          Delete
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </div>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      {/* Automation Rules */}
      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <Target className="h-5 w-5" />
            Automation Rules
          </CardTitle>
          <CardDescription>
            Automatically enroll contacts into playbooks when they meet criteria
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {rulesLoading && (
            <div className="rounded-lg border p-4 text-sm text-muted-foreground">
              Loading automation rules...
            </div>
          )}
          {rules.map((rule) => (
            <div
              key={rule.automation_rule_id}
              className="rounded-lg border bg-card p-4 transition-all hover:shadow-md"
            >
              <div className="mb-3 flex items-start justify-between">
                <div className="flex-1">
                  <div className="mb-1 flex items-center gap-2">
                    <h3 className="font-semibold">{rule.name}</h3>
                    {rule.is_active ? (
                      <Badge variant="default" className="gap-1">
                        <Play className="h-3 w-3" />
                        Active
                      </Badge>
                    ) : (
                      <Badge variant="secondary" className="gap-1">
                        <Pause className="h-3 w-3" />
                        Paused
                      </Badge>
                    )}
                    {rule.enrollment_criteria.require_human_approval && (
                      <Badge variant="outline" className="gap-1">
                        <AlertCircle className="h-3 w-3" />
                        Requires Approval
                      </Badge>
                    )}
                  </div>

                  <div className="text-sm text-muted-foreground">
                    <p>
                      <strong>Segment:</strong> {rule.segment_name} →{' '}
                      <strong>Playbook:</strong> {rule.playbook_name}
                    </p>
                    <p>
                      <strong>Criteria:</strong> Fit Score ≥{' '}
                      {rule.enrollment_criteria.fit_score_threshold}
                      {rule.enrollment_criteria.engagement_score_threshold &&
                        `, Engagement ≥ ${rule.enrollment_criteria.engagement_score_threshold}`}
                    </p>
                  </div>
                </div>

                <Switch
                  checked={rule.is_active}
                  onCheckedChange={() => toggleRule(rule)}
                />
              </div>

              <Separator className="my-3" />

              {/* Stats */}
              <div className="grid grid-cols-4 gap-3">
                <div className="rounded-lg border bg-muted/50 p-2 text-center">
                  <div className="text-xs text-muted-foreground">Enrolled</div>
                  <div className="text-lg font-bold">
                    {rule.stats.contacts_enrolled}
                  </div>
                </div>
                <div className="rounded-lg border bg-yellow-50 p-2 text-center dark:bg-yellow-950">
                  <div className="text-xs text-muted-foreground">Pending</div>
                  <div className="text-lg font-bold">
                    {rule.stats.contacts_pending_approval}
                  </div>
                </div>
                <div className="rounded-lg border bg-blue-50 p-2 text-center dark:bg-blue-950">
                  <div className="text-xs text-muted-foreground">
                    In Progress
                  </div>
                  <div className="text-lg font-bold">
                    {rule.stats.contacts_in_progress}
                  </div>
                </div>
                <div className="rounded-lg border bg-green-50 p-2 text-center dark:bg-green-950">
                  <div className="text-xs text-muted-foreground">
                    Completed
                  </div>
                  <div className="text-lg font-bold">
                    {rule.stats.contacts_completed}
                  </div>
                </div>
              </div>
            </div>
          ))}

          <Button
            variant="outline"
            className="w-full"
            onClick={() => setShowCreate((prev) => !prev)}
          >
            <Target className="mr-2 h-4 w-4" />
            {showCreate ? 'Hide Create Form' : 'Create New Automation Rule'}
          </Button>

          {showCreate && (
            <div className="rounded-lg border bg-muted/40 p-4 space-y-3">
              <div className="grid gap-2">
                <Label>Rule Name</Label>
                <Input
                  value={form.name}
                  onChange={(e) =>
                    setForm({ ...form, name: e.target.value })
                  }
                  placeholder="Auto-enroll High-Fit"
                />
              </div>
              <div className="grid gap-2">
                <Label>Segment</Label>
                <Select
                  value={form.segment_id}
                  onValueChange={(value) =>
                    setForm({ ...form, segment_id: value })
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select segment" />
                  </SelectTrigger>
                  <SelectContent>
                    {segmentOptions.map((seg) => (
                      <SelectItem key={seg.id} value={seg.id}>
                        {seg.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setShowCreateSegment((prev) => !prev)}
                >
                  {showCreateSegment ? 'Hide Segment Form' : 'Create Segment'}
                </Button>
                {showCreateSegment && (
                  <div className="rounded-lg border bg-muted/50 p-3 space-y-2">
                    <div className="grid gap-2">
                      <Label>Segment Name</Label>
                      <Input
                        value={segmentForm.name}
                        onChange={(e) =>
                          setSegmentForm({ ...segmentForm, name: e.target.value })
                        }
                        placeholder="High-Fit Enterprise"
                      />
                    </div>
                    <div className="grid gap-2">
                      <Label>Description</Label>
                      <Input
                        value={segmentForm.description}
                        onChange={(e) =>
                          setSegmentForm({
                            ...segmentForm,
                            description: e.target.value,
                          })
                        }
                        placeholder="Target large accounts"
                      />
                    </div>
                    <div className="grid gap-2">
                      <Label>Priority (1-10)</Label>
                      <Input
                        type="number"
                        min={1}
                        max={10}
                        value={segmentForm.priority}
                        onChange={(e) =>
                          setSegmentForm({
                            ...segmentForm,
                            priority: Number(e.target.value || 5),
                          })
                        }
                      />
                    </div>
                    <div className="flex items-center justify-between">
                      <Label>Active</Label>
                      <Switch
                        checked={segmentForm.is_active}
                        onCheckedChange={(checked) =>
                          setSegmentForm({ ...segmentForm, is_active: checked })
                        }
                      />
                    </div>
                    <Button onClick={submitSegment}>Create Segment</Button>
                  </div>
                )}
              </div>
              <div className="grid gap-2">
                <Label>Playbook</Label>
                <Select
                  value={form.playbook_id}
                  onValueChange={(value) =>
                    setForm({ ...form, playbook_id: value })
                  }
                >
                  <SelectTrigger>
                    <SelectValue placeholder="Select playbook" />
                  </SelectTrigger>
                  <SelectContent>
                    {playbookOptions.map((pb) => (
                      <SelectItem key={pb.playbook_id} value={pb.playbook_id}>
                        {pb.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="grid gap-2">
                <Label>Fit Score Threshold</Label>
                <Input
                  type="number"
                  value={form.fit_score_threshold}
                  onChange={(e) =>
                    setForm({
                      ...form,
                      fit_score_threshold: Number(e.target.value || 0),
                    })
                  }
                />
              </div>
              <div className="grid gap-2">
                <Label>Engagement Score Threshold (optional)</Label>
                <Input
                  type="number"
                  value={form.engagement_score_threshold}
                  onChange={(e) =>
                    setForm({
                      ...form,
                      engagement_score_threshold: e.target.value,
                    })
                  }
                />
              </div>
              <div className="flex items-center justify-between">
                <Label>Require Human Approval</Label>
                <Switch
                  checked={form.require_human_approval}
                  onCheckedChange={(checked) =>
                    setForm({ ...form, require_human_approval: checked })
                  }
                />
              </div>
              <div className="flex items-center justify-between">
                <Label>Active</Label>
                <Switch
                  checked={form.is_active}
                  onCheckedChange={(checked) =>
                    setForm({ ...form, is_active: checked })
                  }
                />
              </div>
              <Button onClick={submitRule} disabled={!canSubmit}>
                Create Rule
              </Button>
            </div>
          )}
        </CardContent>
      </Card>

      {/* Pending Approvals */}
      {pendingRequests.length > 0 && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <AlertCircle className="h-5 w-5" />
              Pending Approvals ({pendingRequests.length})
            </CardTitle>
            <CardDescription>
              Review and approve contacts for automatic enrollment
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-3">
            {pendingRequests.map((request) => (
              <div
                key={request.contact_id}
                className="rounded-lg border border-yellow-200 bg-yellow-50 p-4 dark:border-yellow-800 dark:bg-yellow-950"
              >
                <div className="mb-3 flex items-start justify-between">
                  <div>
                    <h4 className="font-semibold">{request.contact_name}</h4>
                    <p className="text-sm text-muted-foreground">
                      Playbook: {request.playbook_name}
                    </p>
                  </div>
                  <div className="flex gap-2">
                    <Badge variant="default">
                      Fit: {request.fit_score}%
                    </Badge>
                    <Badge variant="secondary">
                      Engagement: {request.engagement_score}%
                    </Badge>
                  </div>
                </div>

                <div className="mb-3 rounded-md bg-background p-2 text-sm">
                  <strong>Reason:</strong> {request.reason}
                </div>

                <div className="flex gap-2">
                  <Button
                    size="sm"
                    onClick={() => approveEnrollment(request.contact_id)}
                    className="flex-1"
                  >
                    <CheckCircle2 className="mr-2 h-4 w-4" />
                    Approve & Enroll
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => rejectEnrollment(request.contact_id)}
                  >
                    Reject
                  </Button>
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      )}

    </div>
  );
}
