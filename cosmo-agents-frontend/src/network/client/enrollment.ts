import { kyClient } from '@/lib/ky';
import type { ApiResponse } from '@/models/response';

export type EnrollmentStatus =
  | 'pending_approval'
  | 'active'
  | 'paused'
  | 'completed';

export interface EnrollContactRequest {
  playbook_id: string;
}

export interface EnrollmentRead {
  enrollment_id: string;
  contact_id: string;
  playbook_id: string;
  automation_rule_id?: string;
  enrollment_status: EnrollmentStatus;
  current_stage_order: number;
  current_stage_id: string;
  enrolled_at?: string;
  completed_at?: string;
  execution_log?: Array<{
    stage_id: string;
    stage_name: string;
    executed_at: string;
    status: string;
    notes?: string;
  }>;
  created_at: string;
  updated_at: string;
}

export interface PendingApprovalRequest {
  request_id: string;
  contact_id: string;
  playbook_id: string;
  automation_rule_id: string;
  reason: string;
  fit_score: number;
  engagement_score: number;
  status: 'pending' | 'approved' | 'rejected';
  reviewed_by?: string;
  reviewed_at?: string;
  created_at: string;
}

export interface UpdateEnrollmentStatusRequest {
  status: 'active' | 'paused' | 'completed';
}

const EnrollmentApi = {
  // Enroll a contact into a playbook
  enrollContact: async (contactId: string, playbookId: string) => {
    const data = await kyClient.post(`v1/contacts/${contactId}/enroll`, {
      json: { playbook_id: playbookId },
    });
    return data.json<ApiResponse<{ message: string }>>();
  },

  // Get enrollment details
  getEnrollment: async (enrollmentId: string) => {
    const data = await kyClient.get(`v1/enrollments/${enrollmentId}`);
    return data.json<ApiResponse<EnrollmentRead>>();
  },

  // Update enrollment status
  updateEnrollmentStatus: async (
    enrollmentId: string,
    status: EnrollmentStatus
  ) => {
    const data = await kyClient.patch(`v1/enrollments/${enrollmentId}/status`, {
      json: { status },
    });
    return data.json<ApiResponse<{ message: string }>>();
  },

  // List pending approval requests
  listPendingApprovals: async () => {
    const data = await kyClient.get('v1/enrollment-approvals/pending');
    return data.json<ApiResponse<PendingApprovalRequest[]>>();
  },

  // Approve an enrollment request
  approveEnrollment: async (requestId: string) => {
    const data = await kyClient.post(
      `v1/enrollment-approvals/${requestId}/approve`
    );
    return data.json<ApiResponse<{ message: string }>>();
  },

  // Reject an enrollment request
  rejectEnrollment: async (requestId: string) => {
    const data = await kyClient.post(
      `v1/enrollment-approvals/${requestId}/reject`
    );
    return data.json<ApiResponse<{ message: string }>>();
  },
};

export default EnrollmentApi;
