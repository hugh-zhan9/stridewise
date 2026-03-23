import { useState, useEffect } from 'react';
import { Loader2 } from 'lucide-react';
import { profileApi } from '../../api/profile';
import type { ProfileResponse } from '../../api/types';

export default function ProfilePage() {
  const [loading, setLoading] = useState(true);
  const [profile, setProfile] = useState<ProfileResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchProfile() {
      try {
        setLoading(true);
        const data = await profileApi.getProfile();
        setProfile(data);
      } catch (err: any) {
        setError(err.message || '获取用户资料失败');
      } finally {
        setLoading(false);
      }
    }
    fetchProfile();
  }, []);

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center h-full pt-32 text-text-secondary">
        <Loader2 className="animate-spin mb-4" size={32} />
        <p>加载个人档案中...</p>
      </div>
    );
  }

  if (error || !profile) {
    return (
      <div className="p-6 pt-32 text-center">
        <div className="bg-status-stop/10 text-status-stop p-4 rounded-xl border border-status-stop/20">
          <p>{error || '数据异常'}</p>
        </div>
      </div>
    );
  }

  // 映射能力层级
  const abilityMap: Record<string, string> = {
    'beginner': '入门跑者',
    'intermediate': '进阶跑者',
    'advanced': '精英跑者'
  };

  // 映射目标类型
  const goalMap: Record<string, string> = {
    'fat_loss': '健康减脂',
    'health_maintain': '保持健康',
    'improve_5k': '成绩提升'
  };

  return (
    <div className="p-6 space-y-8">
      {/* 个人名片 */}
      <section className="flex items-center gap-4">
        <div className="w-16 h-16 rounded-full bg-gradient-to-tr from-electric-blue to-purple-500 p-0.5">
          <div className="w-full h-full bg-carbon rounded-full flex items-center justify-center">
            <span className="text-xl font-bold uppercase">{profile.user_id.slice(0, 1) || 'U'}</span>
          </div>
        </div>
        <div>
          <h1 className="text-2xl font-bold">跑者_{profile.user_id.slice(-4)}</h1>
          <p className="text-sm text-electric-blue font-medium mt-1">
            🏅 {abilityMap[profile.ability_level] || '跑者'}
          </p>
        </div>
      </section>

      {/* 身体基准信息 */}
      <section className="grid grid-cols-2 gap-4">
        <div className="bg-carbon rounded-xl p-4 border border-white/5">
          <p className="text-text-secondary/60 text-xs mb-1">静息心率</p>
          <p className="font-mono text-xl font-bold">{profile.resting_hr || '--'} <span className="text-sm font-sans font-normal text-text-secondary">bpm</span></p>
        </div>
        <div className="bg-carbon rounded-xl p-4 border border-white/5">
          <p className="text-text-secondary/60 text-xs mb-1">训练偏好</p>
          <p className="text-sm font-bold mt-1.5">{profile.weekly_sessions}次/周</p>
        </div>
      </section>

      {/* 目标设定舱 */}
      <section className="bg-carbon rounded-2xl p-5 border border-white/5 relative overflow-hidden">
        <h2 className="text-sm text-text-secondary/60 mb-2">当前长期目标 · <span className="text-text-secondary">{goalMap[profile.goal_type]}</span></h2>
        <h3 className="text-xl font-bold mb-1">{profile.goal_target || '未设置具体目标'}</h3>
        {profile.ability_level_reason && (
          <p className="text-xs text-text-secondary/60 mt-3 p-2 bg-space-obsidian/40 rounded-lg">
            引擎评级依据: {profile.ability_level_reason}
          </p>
        )}
        
        <button className="mt-4 text-electric-blue text-sm font-medium active:scale-95 transition-transform">调整目标与参数 →</button>
      </section>

      {/* 数据源管理 */}
      <section>
        <h2 className="text-sm text-text-secondary/80 font-medium mb-4">数据源连接</h2>
        <div className="space-y-3">
          <div className="flex items-center justify-between bg-space-gray p-4 rounded-xl border border-white/5">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 bg-white text-black rounded-lg flex items-center justify-center font-bold text-xs">Ap</div>
              <span className="font-medium">Apple Health</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-status-go text-xs font-medium">已连接</span>
              <div className="w-10 h-6 bg-status-go rounded-full relative shadow-inner">
                <div className="w-4 h-4 bg-white rounded-full absolute right-1 top-1 shadow-sm"></div>
              </div>
            </div>
          </div>

          <div className="flex items-center justify-between bg-space-gray p-4 rounded-xl border border-white/5 opacity-60">
            <div className="flex items-center gap-3">
              <div className="w-8 h-8 bg-[#FC4C02] text-white rounded-lg flex items-center justify-center font-bold text-xs">St</div>
              <span className="font-medium">Strava</span>
            </div>
            <div className="flex items-center gap-2">
              <span className="text-text-secondary/60 text-xs font-medium">未连接</span>
              <div className="w-10 h-6 bg-space-obsidian border border-text-secondary/30 rounded-full relative">
                <div className="w-4 h-4 bg-text-secondary/50 rounded-full absolute left-1 top-1"></div>
              </div>
            </div>
          </div>
        </div>
      </section>
      
      <div className="pt-8 pb-4 text-center">
        <a href="#" className="text-xs text-text-secondary/40 underline underline-offset-4 hover:text-text-secondary transition-colors">健康免责声明与隐私协议</a>
      </div>
    </div>
  );
}