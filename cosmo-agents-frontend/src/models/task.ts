export type TaskStatus =
  | 'pending'
  | 'running'
  | 'cancelled'
  | 'failed'
  | 'done';
export type TaskPriority = 'low' | 'medium' | 'high' | 'critical';

export interface Task {
  id: string;
  created_at: string;
  updated_at: string;
  contact_id: string;
  campaign_id: string;
  template_id: string;
  schedule_at: string | null;
  responded_at: string | null;
  triggered_at: string | null;
  done_at: string | null;
  priority: TaskPriority;
  status: TaskStatus;
  payload: Record<string, any> | null;
  error: string | null;
}

export interface TaskListResponse {
  items: Task[];
  total: number;
  page: number;
  page_size: number;
  total_pages: number;
}

export interface CreateTaskRequest {
  contact_id: string;
  campaign_id: string;
  template_id: string;
  schedule_at?: string;
  priority?: TaskPriority;
  payload?: Record<string, any>;
}

export interface UpdateTaskRequest {
  schedule_at?: string;
  priority?: TaskPriority;
  status?: TaskStatus;
  payload?: Record<string, any>;
  error?: string;
}

export interface TaskListParams {
  offset?: number;
  limit?: number;
  campaign_id?: string;
  contact_id?: string;
  status?: TaskStatus;
  /** Only tasks of campaigns the caller created. */
  mine?: boolean;
  /** Only tasks whose scheduled time has passed. */
  overdue?: boolean;
}
