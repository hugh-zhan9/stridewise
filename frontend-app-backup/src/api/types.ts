export interface EnvelopeError {
  code: string;
  message: string;
}

export interface EnvelopeMeta {
  request_id: string;
  timestamp: string;
  fallback_reason?: string;
  confidence: number;
}

export interface Envelope<T> {
  data: T;
  error: EnvelopeError | null;
  meta: EnvelopeMeta;
}

export interface AlternativeWorkout {
  type: 'treadmill' | 'strength' | 'mobility' | 'rest';
  title: string;
  duration_min?: number;
  intensity?: 'low' | 'medium';
  tips?: string[];
}

export interface Recommendation {
  id: string;
  user_id: string;
  created_at: string;
  recommendation_date: string;
  rec_date_local: string;
  status: 'draft' | 'active' | 'consumed' | 'expired';
  
  // AI output mapping
  should_run: boolean;
  workout_type: string;
  intensity_range: string;
  target_volume: string;
  suggested_time_window: string;
  risk_level: 'green' | 'yellow' | 'red';
  hydration_tip?: string;
  clothing_tip?: string;
  alternative_workouts?: AlternativeWorkout[];
  explanation: string[];
  
  // Metadata
  is_fallback: boolean;
  ai_provider?: string;
  ai_model?: string;
  engine_version?: string;
  prompt_version?: string;
  model_name?: string;
}

export interface TrainingLog {
  id: string;
  train_date_local: string;
  train_type: string;
  duration_min: number;
  distance_km: number;
  avg_pace: string;
  rpe: number;
  discomfort_flag: boolean;
  source: 'manual' | 'third_party';
}

export interface TrainingLogListResponse {
  items: TrainingLog[];
  next_cursor: string | null;
}

export interface ProfileResponse {
  user_id: string;
  goal_type: string;
  goal_target: string;
  ability_level: 'beginner' | 'intermediate' | 'advanced';
  ability_level_reason?: string;
  ability_level_updated_at?: string;
  resting_hr?: number;
  timezone: string;
  running_years: string;
  weekly_sessions: string;
  weekly_distance_km: string;
  longest_run_km: string;
  recent_discomfort: 'yes' | 'no';
}
