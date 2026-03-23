import { apiClient } from './client';
import type { Recommendation, Envelope } from './types';

export const recommendationApi = {
  /**
   * 获取今日训练建议
   * GET /api/v1/recommendations/today
   */
  getToday: async (): Promise<Recommendation> => {
    // 注意：当前 API 返回的是 Envelope<Recommendation>
    const response = await apiClient.get<Envelope<Recommendation>>('/api/v1/recommendations/today');
    
    // 如果业务报错，直接抛出
    if (response.data.error) {
      throw new Error(response.data.error.message);
    }
    
    return response.data.data;
  },
  
  /**
   * 标记建议已消费
   * POST /api/v1/recommendations/{id}/consume
   */
  consume: async (id: string): Promise<Recommendation> => {
    const response = await apiClient.post<Envelope<Recommendation>>(`/api/v1/recommendations/${id}/consume`);
    if (response.data.error) {
      throw new Error(response.data.error.message);
    }
    return response.data.data;
  }
};
