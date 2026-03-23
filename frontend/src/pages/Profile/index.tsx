import { useState, useEffect } from 'react';
import { Loader2, Smartphone, Activity, X } from 'lucide-react';
import { profileApi } from '../../api/profile';
import type { ProfileResponse } from '../../api/types';

export default function ProfilePage() {
  const [loading, setLoading] = useState(true);
  const [profile, setProfile] = useState<ProfileResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  // Modal states
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [formData, setFormData] = useState({
    goal_type: 'improve_5k' as 'fat_loss' | 'health_maintain' | 'improve_5k',
    goal_target: ''
  });

  useEffect(() => {
    async function fetchProfile() {
      try {
        setLoading(true);
        const data = await profileApi.getProfile();
        setProfile(data);
        setFormData({
          goal_type: data.goal_type as any,
          goal_target: data.goal_target || ''
        });
      } catch (err: any) {
        setError(err.message || '获取用户资料失败');
      } finally {
        setLoading(false);
      }
    }
    fetchProfile();
  }, []);

  const handleUpdateGoal = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setSubmitting(true);
      const updatedProfile = await profileApi.updateGoal(formData);
      setProfile(updatedProfile);
      setIsModalOpen(false);
    } catch (err: any) {
      alert('更新失败: ' + err.message);
    } finally {
      setSubmitting(false);
    }
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center h-[70vh] text-text-secondary">
        <Loader2 className="animate-spin mb-4" size={40} />
        <p className="text-lg">加载档案数据中...</p>
      </div>
    );
  }

  if (error || !profile) {
    return (
      <div className="p-6 pt-32 text-center">
        <div className="bg-status-stop/10 text-status-stop p-6 rounded-xl border border-status-stop/20 inline-block">
          <p className="text-lg">{error || '数据异常'}</p>
        </div>
      </div>
    );
  }

  const abilityMap: Record<string, string> = {
    'beginner': '入门跑者',
    'intermediate': '进阶跑者',
    'advanced': '精英跑者'
  };

  const goalMap: Record<string, string> = {
    'fat_loss': '健康减脂',
    'health_maintain': '保持健康',
    'improve_5k': '成绩提升'
  };

  return (
    <div className="space-y-8 animate-in fade-in duration-500 relative">
      <header>
        <h1 className="text-3xl font-bold">我的档案</h1>
      </header>

      {/* Goal Update Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="bg-carbon border border-white/10 rounded-2xl w-full max-w-md shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200">
            <div className="flex justify-between items-center p-5 border-b border-white/5">
              <h3 className="font-bold text-lg">调整长期目标</h3>
              <button onClick={() => setIsModalOpen(false)} className="text-text-secondary hover:text-white transition-colors">
                <X size={20} />
              </button>
            </div>
            <form onSubmit={handleUpdateGoal} className="p-5 space-y-5">
              <div>
                <label className="block text-sm text-text-secondary mb-2">目标类型</label>
                <div className="grid grid-cols-1 gap-2">
                  {Object.entries(goalMap).map(([k, v]) => (
                    <label 
                      key={k} 
                      className={`flex items-center gap-3 p-3 rounded-xl border cursor-pointer transition-all ${
                        formData.goal_type === k ? 'border-electric-blue bg-electric-blue/10' : 'border-white/10 bg-space-obsidian/50 hover:border-white/30'
                      }`}
                    >
                      <input 
                        type="radio" 
                        name="goal_type" 
                        value={k} 
                        checked={formData.goal_type === k}
                        onChange={(e) => setFormData({...formData, goal_type: e.target.value as any})}
                        className="hidden" 
                      />
                      <div className={`w-4 h-4 rounded-full border-2 flex items-center justify-center ${
                        formData.goal_type === k ? 'border-electric-blue' : 'border-text-secondary/50'
                      }`}>
                        {formData.goal_type === k && <div className="w-2 h-2 bg-electric-blue rounded-full"></div>}
                      </div>
                      <span className="font-medium">{v}</span>
                    </label>
                  ))}
                </div>
              </div>

              <div>
                <label className="block text-sm text-text-secondary mb-2">具体目标描述 (可选)</label>
                <input 
                  type="text" 
                  value={formData.goal_target} 
                  onChange={e => setFormData({...formData, goal_target: e.target.value})} 
                  placeholder="例如: 5公里跑进25分钟，或半马完赛"
                  className="w-full bg-space-obsidian border border-white/10 rounded-xl px-4 py-3 text-sm focus:outline-none focus:border-electric-blue text-white" 
                />
              </div>

              <div className="pt-2 flex justify-end gap-3">
                <button type="button" onClick={() => setIsModalOpen(false)} className="px-5 py-2.5 rounded-xl text-sm text-text-secondary hover:text-white hover:bg-white/5 transition-colors">
                  取消
                </button>
                <button type="submit" disabled={submitting} className="px-6 py-2.5 rounded-xl text-sm font-bold bg-electric-blue text-white hover:bg-[#0066CC] disabled:opacity-50 transition-colors flex items-center gap-2">
                  {submitting && <Loader2 size={16} className="animate-spin" />}
                  保存更改
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        {/* 左侧：个人名片 */}
        <div className="lg:col-span-1 space-y-6">
          <section className="bg-carbon rounded-3xl p-8 border border-white/5 shadow-xl text-center">
            <div className="w-24 h-24 mx-auto rounded-full bg-gradient-to-tr from-electric-blue to-purple-500 p-1 mb-6">
              <div className="w-full h-full bg-carbon rounded-full flex items-center justify-center">
                <span className="text-4xl font-bold uppercase">{profile.user_id.slice(0, 1) || 'U'}</span>
              </div>
            </div>
            <h2 className="text-2xl font-bold">跑者_{profile.user_id.slice(-4)}</h2>
            <div className="inline-flex items-center gap-2 mt-3 px-4 py-1.5 bg-electric-blue/10 text-electric-blue rounded-full text-sm font-medium">
              <span>🏅</span>
              {abilityMap[profile.ability_level] || '跑者'}
            </div>
          </section>

          <section className="bg-carbon rounded-3xl p-8 border border-white/5 shadow-xl">
             <h3 className="text-lg font-bold mb-6">基本体征与偏好</h3>
             <div className="space-y-4">
                <div className="flex justify-between items-center py-3 border-b border-white/5">
                  <span className="text-text-secondary">静息心率</span>
                  <span className="font-mono font-bold text-xl">{profile.resting_hr || '--'} <span className="text-sm font-sans text-text-secondary/60">bpm</span></span>
                </div>
                <div className="flex justify-between items-center py-3 border-b border-white/5">
                  <span className="text-text-secondary">训练频率</span>
                  <span className="font-bold text-lg">{profile.weekly_sessions}次/周</span>
                </div>
                <div className="flex justify-between items-center py-3">
                  <span className="text-text-secondary">周跑量区间</span>
                  <span className="font-bold text-lg">{profile.weekly_distance_km} km</span>
                </div>
             </div>
          </section>
        </div>

        {/* 右侧：目标与设备 */}
        <div className="lg:col-span-2 space-y-6">
          <section className="bg-carbon rounded-3xl p-8 border border-white/5 shadow-xl">
            <div className="flex justify-between items-start mb-8">
              <div>
                <h3 className="text-xl font-bold flex items-center gap-2"><Activity size={20} className="text-electric-blue"/> 当前长期目标</h3>
                <p className="text-text-secondary mt-1">{goalMap[profile.goal_type]}</p>
              </div>
              <button 
                onClick={() => setIsModalOpen(true)}
                className="text-electric-blue bg-electric-blue/10 hover:bg-electric-blue/20 px-5 py-2.5 rounded-xl transition-colors text-sm font-medium border border-electric-blue/20"
              >
                调整参数
              </button>
            </div>
            
            <div className="bg-space-obsidian/40 rounded-2xl p-8 border border-white/5">
              <h4 className="text-3xl font-bold mb-6">{profile.goal_target || '未设置具体目标'}</h4>
              {profile.ability_level_reason && (
                <div className="bg-space-gray p-5 rounded-xl text-sm text-text-secondary/80 leading-relaxed border-l-4 border-electric-blue">
                  <span className="font-bold text-white mr-2 block mb-1">AI 评级依据</span>
                  {profile.ability_level_reason}
                </div>
              )}
            </div>
          </section>

          <section className="bg-carbon rounded-3xl p-8 border border-white/5 shadow-xl">
            <h3 className="text-xl font-bold mb-8 flex items-center gap-2"><Smartphone size={20} /> 数据源与设备连接</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
              <div className="flex items-center justify-between bg-space-gray p-6 rounded-2xl border border-status-go/30 transition-all hover:bg-white/5 cursor-pointer shadow-lg shadow-status-go/5">
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 bg-white text-black rounded-xl flex items-center justify-center font-bold text-lg">Ap</div>
                  <div>
                    <span className="font-bold text-lg block mb-1">Apple Health</span>
                    <span className="text-xs text-status-go flex items-center gap-1"><span className="w-1.5 h-1.5 rounded-full bg-status-go"></span>已连接并正在同步</span>
                  </div>
                </div>
                <div className="w-12 h-7 bg-status-go rounded-full relative shadow-inner">
                  <div className="w-5 h-5 bg-white rounded-full absolute right-1 top-1 shadow-sm"></div>
                </div>
              </div>

              <div className="flex items-center justify-between bg-space-gray p-6 rounded-2xl border border-white/5 opacity-60 transition-all hover:opacity-100 cursor-pointer">
                <div className="flex items-center gap-4">
                  <div className="w-12 h-12 bg-[#FC4C02] text-white rounded-xl flex items-center justify-center font-bold text-lg">St</div>
                  <div>
                    <span className="font-bold text-lg block mb-1">Strava</span>
                    <span className="text-xs text-text-secondary/60">未连接</span>
                  </div>
                </div>
                <div className="w-12 h-7 bg-space-obsidian border border-text-secondary/30 rounded-full relative">
                  <div className="w-5 h-5 bg-text-secondary/50 rounded-full absolute left-1 top-1"></div>
                </div>
              </div>
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}