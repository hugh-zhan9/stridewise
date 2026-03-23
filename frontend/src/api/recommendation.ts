import { apiClient } from './client';
import type { Recommendation, Envelope, FeedbackRequest } from './types';

export const recommendationApi = {
  /**
   * 获取今日训练建议
   */
  getToday: async (): Promise<Recommendation> => {
    const response = await apiClient.get<Envelope<Recommendation>>('/internal/v1/recommendations/latest');
    if (response.data.error) {
      throw new Error(response.data.error.message);
    }
    return response.data.data;
  },

  /**
   * 提交建议反馈
   */
  feedback: async (id: string, data: FeedbackRequest): Promise<void> => {
    const payload = {
      ...data,
      user_id: 'user_001', // 依据后端的 recommendationFeedbackRequest
      useful: data.usefulness // 映射字段名
    };
    const response = await apiClient.post<Envelope<any>>(`/internal/v1/recommendations/${id}/feedback`, payload);
    if (response.data.error) {
      throw new Error(response.data.error.message);
    }
  }
};
