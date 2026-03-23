import { apiClient } from './client';
import type { TrainingLogListResponse, Envelope, BaselineResponse, CreateTrainingLogRequest, TrainingLog } from './types';

export const trainingApi = {
  /**
   * 新增手动训练记录
   * POST /internal/v1/training/logs
   */
  createLog: async (data: CreateTrainingLogRequest): Promise<TrainingLog> => {
    const payload = {
      ...data,
      user_id: 'user_001',
      training_type: data.train_type,
      start_time: data.train_date_local + ' 00:00:00', // 简单补齐时间
      duration: `${data.duration_min}m`,
      distance_km: data.distance_km,
      pace: data.avg_pace || '06:00',
      rpe: data.rpe,
      discomfort: data.discomfort_flag
    };
    const response = await apiClient.post<Envelope<any>>('/internal/v1/training/logs', payload);
    if (response.data.error) {
      throw new Error(response.data.error.message);
    }
    // 后端接口返回的是 {"log_id": "xxx", "job_id": "xxx"}，前端为了立刻渲染，需要拼凑假数据或重新拉取
    // 这里为了演示方便，先返回请求体拼装成前端所需的 TrainingLog 格式
    return {
      id: response.data.data.log_id,
      train_date_local: data.train_date_local,
      train_type: data.train_type,
      duration_min: data.duration_min,
      distance_km: data.distance_km,
      avg_pace: data.avg_pace || '06:00',
      rpe: data.rpe,
      discomfort_flag: data.discomfort_flag,
      source: 'manual'
    };
  },

  /**
   * 获取训练记录流水
   * GET /internal/v1/training/logs
   */
  getLogs: async (cursor?: string, pageSize: number = 20): Promise<TrainingLogListResponse> => {
    const params = new URLSearchParams();
    if (cursor) params.append('cursor', cursor);
    params.append('page_size', pageSize.toString());

    // 后端返回的是一个数组 []，前端为了兼容目前的列表分页，包一层
    const response = await apiClient.get<Envelope<any>>(`/internal/v1/training/logs?${params.toString()}`);
    if (response.data.error) {
      throw new Error(response.data.error.message);
    }
    
    // 如果返回的本来就是数组，包成 {items: ...}
    let items = response.data.data;
    if (Array.isArray(items)) {
      // 字段名映射
      items = items.map((log: any) => ({
        id: log.log_id,
        train_date_local: log.start_time,
        train_type: log.training_type,
        duration_min: Math.round(log.duration_sec / 60),
        distance_km: log.distance_km,
        avg_pace: log.pace_str,
        rpe: log.rpe,
        discomfort_flag: log.discomfort,
        source: log.source
      }));
      return { items, next_cursor: null };
    }
    return response.data.data;
  },

  /**
   * 获取当前能力基线
   * GET /internal/v1/baseline/current
   */
  getBaseline: async (): Promise<BaselineResponse> => {
    const response = await apiClient.get<Envelope<any>>('/internal/v1/baseline/current');
    if (response.data.error) {
      throw new Error(response.data.error.message);
    }
    // 根据后端 struct 返回调整
    const data = response.data.data;
    return {
      pace_zone: {
        easy: data.easy_pace,
        tempo: data.tempo_pace,
        interval: data.interval_pace
      },
      weekly_volume_range: {
        min_km: data.suggested_weekly_km - 5,
        max_km: data.suggested_weekly_km + 5
      },
      recovery_level: data.recovery_score > 80 ? 'high' : (data.recovery_score > 50 ? 'medium' : 'low'),
      updated_at: data.calculated_at
    };
  }
};
