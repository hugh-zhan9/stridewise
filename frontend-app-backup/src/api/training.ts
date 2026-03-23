import { apiClient } from './client';
import type { TrainingLogListResponse, Envelope, BaselineResponse } from './types';

export const trainingApi = {
  /**
   * 获取训练记录流水
   * GET /api/v1/training/logs
   */
  getLogs: async (cursor?: string, pageSize: number = 20): Promise<TrainingLogListResponse> => {
    const params = new URLSearchParams();
    if (cursor) params.append('cursor', cursor);
    params.append('page_size', pageSize.toString());

    const response = await apiClient.get<Envelope<TrainingLogListResponse>>(`/api/v1/training/logs?${params.toString()}`);
    if (response.data.error) {
      throw new Error(response.data.error.message);
    }
    return response.data.data;
  },

  /**
   * 获取当前能力基线
   * GET /api/v1/profile/baseline
   */
  getBaseline: async (): Promise<BaselineResponse> => {
    const response = await apiClient.get<Envelope<BaselineResponse>>('/api/v1/profile/baseline');
    if (response.data.error) {
      throw new Error(response.data.error.message);
    }
    return response.data.data;
  }
};
