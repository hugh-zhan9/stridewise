import { apiClient } from './client';
import type { ProfileResponse, Envelope, UpdateGoalRequest } from './types';

export const profileApi = {
  /**
   * 获取当前用户档案
   */
  getProfile: async (): Promise<ProfileResponse> => {
    const response = await apiClient.get<Envelope<ProfileResponse>>('/internal/v1/user/profile');
    if (response.data.error) {
      throw new Error(response.data.error.message);
    }
    return response.data.data;
  },

  /**
   * 初始化/全量更新用户档案
   */
  updateProfile: async (data: any): Promise<ProfileResponse> => {
    // 根据后端 userProfileRequest 结构，前端可以直接发 POST 更新整个 Profile
    data.user_id = 'user_001'; // 强行注入
    const response = await apiClient.post<Envelope<ProfileResponse>>('/internal/v1/user/profile', data);
    if (response.data.error) {
      throw new Error(response.data.error.message);
    }
    return response.data.data;
  }
};
