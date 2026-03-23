import axios from 'axios';
import AxiosMockAdapter from 'axios-mock-adapter';
import { apiClient } from './client';
import type { Envelope, Recommendation, TrainingLogListResponse, BaselineResponse, ProfileResponse } from './types';

// 获取当前的开发环境变量，只有在非生产环境才启用 mock
const isDev = import.meta.env.DEV;

export function setupMock() {
  if (!isDev) return;

  const mock = new AxiosMockAdapter(apiClient, { delayResponse: 800 });

  // ----------------------------------------------------
  // Mock: /api/v1/recommendations/today
  // ----------------------------------------------------
  const mockRecommendation: Envelope<Recommendation> = {
    data: {
      id: 'rec_mock_123',
      user_id: 'user_001',
      created_at: new Date().toISOString(),
      recommendation_date: new Date().toISOString().split('T')[0],
      rec_date_local: new Date().toISOString().split('T')[0],
      status: 'active',
      should_run: true,
      workout_type: 'easy_run',
      intensity_range: '6\'00" - 6\'30"',
      target_volume: '5.0 km',
      suggested_time_window: '下午 4:00 - 6:00',
      risk_level: 'green',
      hydration_tip: '运动前30分钟补充 300ml 水分',
      clothing_tip: '短袖速干衣 + 运动短裤',
      alternative_workouts: [],
      explanation: [
        '昨日完成高强度间歇，当前处于恢复期，需要排酸。',
        '今晚有 60% 降雨概率，建议在下午 4 点前完成轻松跑。',
        '当前的温度适宜（18°C），是进行户外有氧的黄金窗口。'
      ],
      is_fallback: false,
      ai_provider: 'Claude',
      ai_model: 'claude-3-5-sonnet',
      engine_version: 'v1.0'
    },
    error: null,
    meta: {
      request_id: 'req_12345',
      timestamp: new Date().toISOString(),
      confidence: 0.95
    }
  };

  mock.onGet('/api/v1/recommendations/today').reply(200, mockRecommendation);

  mock.onPost(/\/api\/v1\/recommendations\/.*\/consume/).reply(200, {
    ...mockRecommendation,
    data: {
      ...mockRecommendation.data,
      status: 'consumed'
    }
  });

  // ----------------------------------------------------
  // Mock: /api/v1/profile/baseline
  // ----------------------------------------------------
  const mockBaseline: Envelope<BaselineResponse> = {
    data: {
      pace_zone: {
        easy: "6'00\" - 6'30\"",
        tempo: "5'15\" - 5'30\"",
        interval: "4'45\" - 5'00\""
      },
      weekly_volume_range: {
        min_km: 25,
        max_km: 35
      },
      recovery_level: 'medium',
      updated_at: new Date().toISOString()
    },
    error: null,
    meta: { request_id: 'req_2', timestamp: new Date().toISOString(), confidence: 1.0 }
  };
  mock.onGet('/api/v1/profile/baseline').reply(200, mockBaseline);

  // ----------------------------------------------------
  // Mock: /api/v1/training/logs
  // ----------------------------------------------------
  let mockLogsItems = [
    {
      id: 'log_001',
      train_date_local: new Date().toISOString().split('T')[0],
      train_type: 'easy_run',
      duration_min: 32,
      distance_km: 5.02,
      avg_pace: '06:25',
      rpe: 4,
      discomfort_flag: false,
      source: 'third_party' as const
    },
    {
      id: 'log_002',
      train_date_local: new Date(Date.now() - 86400000).toISOString().split('T')[0], // 昨天
      train_type: 'interval',
      duration_min: 45,
      distance_km: 7.5,
      avg_pace: '05:10',
      rpe: 8,
      discomfort_flag: false,
      source: 'manual' as const
    }
  ];

  mock.onGet(/\/api\/v1\/training\/logs/).reply(() => {
    return [200, {
      data: {
        items: [...mockLogsItems], // return current items
        next_cursor: null
      },
      error: null,
      meta: { request_id: 'req_3', timestamp: new Date().toISOString(), confidence: 1.0 }
    }];
  });

  mock.onPost(/\/api\/v1\/training\/logs/).reply((config) => {
    const requestData = JSON.parse(config.data);
    const newLog = {
      id: `log_manual_${Date.now()}`,
      ...requestData,
      source: 'manual' as const,
      avg_pace: requestData.avg_pace || '06:00'
    };
    
    // Add to the front
    mockLogsItems = [newLog, ...mockLogsItems];

    return [201, {
      data: newLog,
      error: null,
      meta: { request_id: `req_post_${Date.now()}`, timestamp: new Date().toISOString(), confidence: 1.0 }
    }];
  });

  // ----------------------------------------------------
  // Mock: /api/v1/profile/me
  // ----------------------------------------------------
  const mockProfile: Envelope<ProfileResponse> = {
    data: {
      user_id: 'user_001',
      goal_type: 'improve_5k',
      goal_target: '5公里跑进25分钟',
      ability_level: 'intermediate',
      ability_level_reason: '基于过去28天配速与心率表现计算',
      ability_level_updated_at: new Date().toISOString(),
      resting_hr: 52,
      timezone: 'Asia/Shanghai',
      running_years: '1-3',
      weekly_sessions: '2-3',
      weekly_distance_km: '15-30',
      longest_run_km: '10',
      recent_discomfort: 'no'
    },
    error: null,
    meta: { request_id: 'req_4', timestamp: new Date().toISOString(), confidence: 1.0 }
  };
  mock.onGet('/api/v1/profile/me').reply(200, mockProfile);

  // ----------------------------------------------------
  // Mock: /api/v1/recommendations/{id}/feedback
  // ----------------------------------------------------
  mock.onPost(/\/api\/v1\/recommendations\/.*\/feedback/).reply(201, {
    data: {
      id: `feedback_${Date.now()}`,
      recommendation_id: 'rec_mock_123',
      usefulness: 'useful',
      created_at: new Date().toISOString()
    },
    error: null,
    meta: { request_id: `req_${Date.now()}`, timestamp: new Date().toISOString(), confidence: 1.0 }
  });

  // ----------------------------------------------------
  // Mock: /api/v1/profile/goal
  // ----------------------------------------------------
  mock.onPut('/api/v1/profile/goal').reply((config) => {
    const requestData = JSON.parse(config.data);
    const updatedProfile = {
      ...mockProfile.data,
      goal_type: requestData.goal_type,
      goal_target: requestData.goal_target || mockProfile.data.goal_target,
    };
    
    // Update the mock profile reference so subsequent GETs return the new data
    mockProfile.data = updatedProfile;

    return [200, {
      data: updatedProfile,
      error: null,
      meta: { request_id: `req_${Date.now()}`, timestamp: new Date().toISOString(), confidence: 1.0 }
    }];
  });

  console.log('[Mock] API Mocking enabled.');
}
