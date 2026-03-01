'use client';

import { Badge } from '@/components/ui/badge';
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card';
import { Button } from '@/components/ui/button';
import { Loader2, CheckCircle2, Clock, Pause, AlertCircle } from 'lucide-react';
import { format } from 'date-fns';
import type { EnrollmentRead } from '@/network/client/enrollment';

interface ContactEnrollmentStatusProps {
  enrollments: EnrollmentRead[];
  isLoading?: boolean;
  onExecuteStage?: (enrollmentId: string) => void;
}

const STATUS_CONFIG = {
  pending_approval: {
    label: 'Pending Approval',
    variant: 'secondary' as const,
    icon: Clock,
    color: 'text-yellow-600',
  },
  active: {
    label: 'Active',
    variant: 'default' as const,
    icon: CheckCircle2,
    color: 'text-green-600',
  },
  paused: {
    label: 'Paused',
    variant: 'outline' as const,
    icon: Pause,
    color: 'text-gray-600',
  },
  completed: {
    label: 'Completed',
    variant: 'secondary' as const,
    icon: CheckCircle2,
    color: 'text-blue-600',
  },
};

export function ContactEnrollmentStatus({
  enrollments,
  isLoading,
  onExecuteStage,
}: ContactEnrollmentStatusProps) {
  if (isLoading) {
    return (
      <Card>
        <CardContent className="flex items-center justify-center py-6">
          <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
        </CardContent>
      </Card>
    );
  }

  if (!enrollments || enrollments.length === 0) {
    return (
      <Card>
        <CardHeader>
          <CardTitle className="text-base">Playbook Enrollments</CardTitle>
          <CardDescription>This contact is not enrolled in any playbooks</CardDescription>
        </CardHeader>
      </Card>
    );
  }

  return (
    <div className="space-y-4">
      <h3 className="font-semibold text-gray-900">Active Playbook Enrollments</h3>
      {enrollments.map((enrollment) => {
        const statusConfig = STATUS_CONFIG[enrollment.enrollment_status];
        const StatusIcon = statusConfig.icon;

        return (
          <Card key={enrollment.enrollment_id}>
            <CardHeader>
              <div className="flex items-start justify-between">
                <div className="space-y-1">
                  <CardTitle className="text-base">Playbook Enrollment</CardTitle>
                  <CardDescription>
                    Current Stage: Stage {enrollment.current_stage_order + 1}
                  </CardDescription>
                </div>
                <Badge variant={statusConfig.variant}>
                  <StatusIcon className="mr-1 h-3 w-3" />
                  {statusConfig.label}
                </Badge>
              </div>
            </CardHeader>
            <CardContent className="space-y-4">
              {/* Current Stage Info */}
              <div className="rounded-lg border bg-muted/50 p-3">
                <div className="mb-2 flex items-center justify-between">
                  <span className="text-sm font-medium">Current Stage</span>
                  <Badge variant="outline">Stage {enrollment.current_stage_order + 1}</Badge>
                </div>
                <p className="text-sm text-muted-foreground">
                  Stage ID: <span className="font-mono text-xs">{enrollment.current_stage_id}</span>
                </p>
              </div>

              {/* Execution Log */}
              {enrollment.execution_log && enrollment.execution_log.length > 0 && (
                <div>
                  <h4 className="mb-2 text-sm font-medium">Execution History</h4>
                  <div className="space-y-2">
                    {enrollment.execution_log.map((log, idx) => (
                      <div
                        key={idx}
                        className="flex items-start justify-between rounded-md border bg-background p-2 text-sm"
                      >
                        <div>
                          <p className="font-medium">{log.stage_name}</p>
                          <p className="text-xs text-muted-foreground">
                            {format(new Date(log.executed_at), 'MMM d, yyyy h:mm a')}
                          </p>
                          {log.notes && (
                            <p className="mt-1 text-xs text-muted-foreground">{log.notes}</p>
                          )}
                        </div>
                        <Badge variant={log.status === 'completed' ? 'default' : 'secondary'}>
                          {log.status}
                        </Badge>
                      </div>
                    ))}
                  </div>
                </div>
              )}

              {/* Metadata */}
              <div className="grid grid-cols-2 gap-4 border-t pt-3 text-xs text-muted-foreground">
                {enrollment.enrolled_at && (
                  <div>
                    <span className="font-medium">Enrolled:</span>{' '}
                    {format(new Date(enrollment.enrolled_at), 'MMM d, yyyy')}
                  </div>
                )}
                {enrollment.completed_at && (
                  <div>
                    <span className="font-medium">Completed:</span>{' '}
                    {format(new Date(enrollment.completed_at), 'MMM d, yyyy')}
                  </div>
                )}
              </div>

              {/* Actions */}
              {enrollment.enrollment_status === 'active' && onExecuteStage && (
                <div className="flex justify-end border-t pt-3">
                  <Button
                    size="sm"
                    onClick={() => onExecuteStage(enrollment.enrollment_id)}
                    variant="outline"
                  >
                    Execute Next Stage
                  </Button>
                </div>
              )}
            </CardContent>
          </Card>
        );
      })}
    </div>
  );
}
