import { kyClient } from '@/lib/ky';
import type {
  Task,
  TaskListResponse,
  TaskListParams,
  CreateTaskRequest,
  UpdateTaskRequest,
} from '@/models/task';
import type { ApiResponse } from '@/models/response';

const TaskApi = {
  list: async (params?: TaskListParams) => {
    const data = await kyClient.get('v1/task', {
      searchParams: params as Record<string, string>,
    });
    return data.json<ApiResponse<TaskListResponse>>();
  },

  getById: async (id: string) => {
    const data = await kyClient.get(`v1/task/${id}`);
    return data.json<ApiResponse<Task>>();
  },

  create: async (payload: CreateTaskRequest) => {
    const data = await kyClient.post('v1/task', { json: payload });
    return data.json<ApiResponse<Task>>();
  },

  update: async (id: string, payload: UpdateTaskRequest) => {
    const data = await kyClient.put(`v1/task/${id}`, { json: payload });
    return data.json<ApiResponse<Task>>();
  },
};

export default TaskApi;
