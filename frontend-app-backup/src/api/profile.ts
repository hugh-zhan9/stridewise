import { apiClient } from './client';
import type { ProfileResponse, Envelope } from './types';

export const profileApi = {
  /**
   * 获取当前用户档案
   * 在现阶段，由于是单用户，我们可以假设有一个获取 Profile 的接口
   * 实际的 OpenAPI 设计里有 /api/v1/profile/init 和 update
   * 为了前端展示，我们需要一个 GET 接口，这里先定义为 /api/v1/profile/me
   */
  getProfile: async (): Promise<ProfileResponse> => {
    const response = await apiClient.get<Envelope<ProfileResponse>>('/api/v1/profile/me');
    if (response.data.error) {
      throw new Error(response.data.error.message);
    }
    return response.data.data;
  }
};
