import { useState, useEffect } from 'react';
import { Loader2, Plus, TrendingUp, X } from 'lucide-react';
import { trainingApi } from '../../api/training';
import { recommendationApi } from '../../api/recommendation';
import type { TrainingLogListResponse, BaselineResponse } from '../../api/types';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';
import { BarChart, Bar, XAxis, Tooltip, ResponsiveContainer, Cell } from 'recharts';
import { clsx } from 'clsx';

export default function LogPage() {
  const [loading, setLoading] = useState(true);
  const [baseline, setBaseline] = useState<BaselineResponse | null>(null);
  const [logsData, setLogsData] = useState<TrainingLogListResponse | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [feedbackState, setFeedbackState] = useState<Record<string, boolean>>({});

  // Modal states
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [formData, setFormData] = useState({
    train_date_local: format(new Date(), 'yyyy-MM-dd'),
    train_type: 'easy_run',
    duration_min: 30,
    distance_km: 5.0,
    avg_pace: '06:00',
    rpe: 5,
    discomfort_flag: false
  });

  useEffect(() => {
    async function fetchData() {
      try {
        setLoading(true);
        const [baselineRes, logsRes] = await Promise.all([
          trainingApi.getBaseline(),
          trainingApi.getLogs()
        ]);
        setBaseline(baselineRes);
        setLogsData(logsRes);
      } catch (err: any) {
        setError(err.message || '获取数据失败');
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, []);

  const handleAddLog = async (e: React.FormEvent) => {
    e.preventDefault();
    try {
      setSubmitting(true);
      const newLog = await trainingApi.createLog(formData);
      
      if (logsData) {
        setLogsData({
          ...logsData,
          items: [newLog, ...logsData.items]
        });
      }
      setIsModalOpen(false);
    } catch (err: any) {
      alert('添加失败: ' + err.message);
    } finally {
      setSubmitting(false);
    }
  };

  const handleFeedback = async (logId: string, usefulness: 'useful' | 'not_useful') => {
    if (feedbackState[logId]) return; // 已经评价过
    try {
      // 注意：真实情况下建议的反馈和日志可能有一一对应关系，这里用 logId 模拟
      await recommendationApi.feedback(logId, { usefulness });
      setFeedbackState(prev => ({ ...prev, [logId]: true }));
      alert('感谢反馈，AI 引擎已学习！');
    } catch (err: any) {
      alert('反馈提交失败: ' + err.message);
    }
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center h-[70vh] text-text-secondary">
        <Loader2 className="animate-spin mb-4" size={40} />
        <p className="text-lg">正在同步您的成长轨迹...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6 pt-32 text-center">
        <div className="bg-status-stop/10 text-status-stop p-6 rounded-xl border border-status-stop/20 inline-block">
          <p>{error}</p>
          <button 
            onClick={() => window.location.reload()} 
            className="mt-4 px-6 py-2 bg-status-stop/20 hover:bg-status-stop/30 rounded-lg transition-colors"
          >
            重试
          </button>
        </div>
      </div>
    );
  }

  const workoutTypeMap: Record<string, string> = {
    'easy_run': '轻松跑',
    'long_run': '长距离跑',
    'interval': '间歇跑',
    'strength': '力量训练',
    'stretch': '拉伸恢复',
    'rest': '休息'
  };

  const getRecoveryColor = (level?: string) => {
    switch(level) {
      case 'high': return 'bg-status-go';
      case 'medium': return 'bg-status-caution';
      case 'low': return 'bg-status-stop';
      default: return 'bg-space-gray';
    }
  };

  const chartData = [
    { name: '周一', distance: 0 },
    { name: '周二', distance: 5.02 },
    { name: '周三', distance: 0 },
    { name: '周四', distance: 7.5 },
    { name: '周五', distance: 0 },
    { name: '周六', distance: 10.2 },
    { name: '周日', distance: 0 },
  ];

  return (
    <div className="space-y-8 animate-in fade-in duration-500 relative">
      <header className="flex flex-col md:flex-row md:justify-between md:items-end gap-4">
        <div>
          <h1 className="text-3xl font-bold">成长轨迹</h1>
          <p className="text-text-secondary mt-2">您的每一次迈步，引擎都在记录与学习。</p>
        </div>
        <button 
          onClick={() => setIsModalOpen(true)}
          className="px-5 py-2.5 bg-electric-blue text-white font-medium rounded-xl hover:bg-[#0066CC] transition-colors flex items-center justify-center gap-2 shadow-lg shadow-electric-blue/20"
        >
          <Plus size={18} />
          添加记录
        </button>
      </header>

      {/* Modal Overlay */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/60 backdrop-blur-sm p-4">
          <div className="bg-carbon border border-white/10 rounded-2xl w-full max-w-md shadow-2xl overflow-hidden animate-in zoom-in-95 duration-200">
            <div className="flex justify-between items-center p-5 border-b border-white/5">
              <h3 className="font-bold text-lg">手动添加记录</h3>
              <button onClick={() => setIsModalOpen(false)} className="text-text-secondary hover:text-white transition-colors">
                <X size={20} />
              </button>
            </div>
            <form onSubmit={handleAddLog} className="p-5 space-y-4">
              
              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs text-text-secondary mb-1">日期</label>
                  <input type="date" required value={formData.train_date_local} onChange={e => setFormData({...formData, train_date_local: e.target.value})} className="w-full bg-space-obsidian border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-electric-blue text-white" />
                </div>
                <div>
                  <label className="block text-xs text-text-secondary mb-1">运动类型</label>
                  <select value={formData.train_type} onChange={e => setFormData({...formData, train_type: e.target.value})} className="w-full bg-space-obsidian border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-electric-blue text-white">
                    {Object.entries(workoutTypeMap).map(([k, v]) => <option key={k} value={k}>{v}</option>)}
                  </select>
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs text-text-secondary mb-1">距离 (km)</label>
                  <input type="number" step="0.01" required min="0" value={formData.distance_km} onChange={e => setFormData({...formData, distance_km: parseFloat(e.target.value) || 0})} className="w-full bg-space-obsidian border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-electric-blue text-white" placeholder="e.g. 5.0" />
                </div>
                <div>
                  <label className="block text-xs text-text-secondary mb-1">时长 (分钟)</label>
                  <input type="number" required min="1" value={formData.duration_min} onChange={e => setFormData({...formData, duration_min: parseInt(e.target.value) || 0})} className="w-full bg-space-obsidian border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-electric-blue text-white" placeholder="e.g. 30" />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div>
                  <label className="block text-xs text-text-secondary mb-1">配速 (分:秒)</label>
                  <input type="text" placeholder="06:00" value={formData.avg_pace} onChange={e => setFormData({...formData, avg_pace: e.target.value})} className="w-full bg-space-obsidian border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-electric-blue text-white" />
                </div>
                <div>
                  <label className="block text-xs text-text-secondary mb-1">疲劳度 (RPE 1-10)</label>
                  <input type="number" required min="1" max="10" value={formData.rpe} onChange={e => setFormData({...formData, rpe: parseInt(e.target.value) || 1})} className="w-full bg-space-obsidian border border-white/10 rounded-lg px-3 py-2 text-sm focus:outline-none focus:border-electric-blue text-white" />
                </div>
              </div>

              <label className="flex items-center gap-2 text-sm mt-4 p-3 bg-space-gray/50 rounded-lg border border-white/5">
                <input type="checkbox" checked={formData.discomfort_flag} onChange={e => setFormData({...formData, discomfort_flag: e.target.checked})} className="rounded text-electric-blue focus:ring-electric-blue bg-space-obsidian border-white/20" />
                <span>运动中是否感到不适/疼痛？</span>
              </label>

              <div className="pt-4 flex justify-end gap-3">
                <button type="button" onClick={() => setIsModalOpen(false)} className="px-4 py-2 rounded-lg text-sm text-text-secondary hover:text-white hover:bg-white/5 transition-colors">
                  取消
                </button>
                <button type="submit" disabled={submitting} className="px-6 py-2 rounded-lg text-sm font-medium bg-electric-blue text-white hover:bg-[#0066CC] disabled:opacity-50 transition-colors flex items-center gap-2">
                  {submitting && <Loader2 size={14} className="animate-spin" />}
                  保存记录
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      <div className="grid grid-cols-1 xl:grid-cols-3 gap-8">
        {/* 左侧：基线与图表 */}
        <div className="xl:col-span-1 space-y-6">
          {baseline && (
            <section className="bg-carbon rounded-3xl p-6 md:p-8 border border-white/5 shadow-xl">
              <h2 className="text-base text-text-secondary/80 mb-6 font-medium flex items-center gap-2">
                <TrendingUp size={18} className="text-electric-blue"/>
                能力基线评估
              </h2>
              
              <div className="space-y-6">
                <div>
                  <div className="flex justify-between text-sm mb-2">
                    <span className="text-text-secondary">轻松跑配速</span>
                    <span className="font-mono text-white font-bold text-base">{baseline.pace_zone?.easy || '--'}</span>
                  </div>
                  <div className="flex justify-between text-sm mb-2">
                    <span className="text-text-secondary">间歇跑配速</span>
                    <span className="font-mono text-white font-bold text-base">{baseline.pace_zone?.interval || '--'}</span>
                  </div>
                  <div className="h-1.5 w-full bg-space-obsidian rounded-full overflow-hidden mt-4">
                    <div className="h-full bg-electric-blue w-3/4 rounded-full"></div>
                  </div>
                </div>

                <div className="pt-6 border-t border-white/5">
                  <div className="flex justify-between text-sm mb-3">
                    <span className="text-text-secondary">身体恢复状态</span>
                    <span className={clsx("font-bold uppercase", `text-${getRecoveryColor(baseline.recovery_level).split('-')[1]}-go`)}>
                      {baseline.recovery_level === 'high' ? '良好' : baseline.recovery_level === 'medium' ? '一般' : '疲劳'}
                    </span>
                  </div>
                  <div className="h-2 w-full bg-space-obsidian rounded-full overflow-hidden">
                    <div className={clsx("h-full rounded-full w-full opacity-80", getRecoveryColor(baseline.recovery_level))}></div>
                  </div>
                </div>
                
                <div className="pt-2 text-xs text-text-secondary/40">
                  <p>建议周跑量: <span className="font-mono">{baseline.weekly_volume_range?.min_km}-{baseline.weekly_volume_range?.max_km}</span> km</p>
                  <p className="mt-1">更新于 {format(new Date(baseline.updated_at), 'MM-dd HH:mm')}</p>
                </div>
              </div>
            </section>
          )}

          {/* 图表模块 */}
          <section className="bg-carbon rounded-3xl p-6 md:p-8 border border-white/5 shadow-xl">
             <h2 className="text-base text-text-secondary/80 mb-6 font-medium">本周跑量 (km)</h2>
             <div className="h-48 w-full">
               <ResponsiveContainer width="100%" height="100%">
                 <BarChart data={chartData} margin={{ top: 0, right: 0, left: -20, bottom: 0 }}>
                   <XAxis dataKey="name" axisLine={false} tickLine={false} tick={{fill: '#EBEBF5', opacity: 0.5, fontSize: 12}} dy={10} />
                   <Tooltip 
                     cursor={{fill: '#1C1C1E'}} 
                     contentStyle={{backgroundColor: '#2C2C2E', borderColor: '#404040', borderRadius: '12px', border: '1px solid rgba(255,255,255,0.05)'}} 
                     itemStyle={{color: '#0A84FF', fontWeight: 'bold'}}
                     formatter={(value: number) => [`${value} km`, '距离']}
                   />
                   <Bar dataKey="distance" radius={[4, 4, 0, 0]} maxBarSize={30}>
                     {chartData.map((entry, index) => (
                       <Cell key={`cell-${index}`} fill={entry.distance > 0 ? '#0A84FF' : '#1C1C1E'} />
                     ))}
                   </Bar>
                 </BarChart>
               </ResponsiveContainer>
             </div>
          </section>
        </div>

        {/* 右侧：训练流水 */}
        <div className="xl:col-span-2">
          <section className="bg-carbon rounded-3xl p-6 md:p-8 border border-white/5 shadow-xl min-h-full">
            <h2 className="text-xl font-bold mb-6">历史流水</h2>
            <div className="space-y-4">
              {logsData?.items.map((log) => {
                const dateStr = format(new Date(log.train_date_local), 'yyyy年MM月dd日', { locale: zhCN });
                return (
                  <div key={log.id} className="bg-space-gray rounded-2xl p-5 border border-white/5 hover:border-white/20 hover:bg-space-gray/80 transition-all group flex flex-col md:flex-row md:items-center justify-between gap-4">
                    <div className="flex items-center gap-5">
                      <div className="w-12 h-12 rounded-full bg-status-go/20 flex items-center justify-center text-status-go font-bold shrink-0">
                        🏃
                      </div>
                      <div>
                        <p className="font-bold text-lg">{workoutTypeMap[log.train_type] || log.train_type}</p>
                        <p className="text-sm text-text-secondary/60 mt-1">{dateStr} · {log.source === 'manual' ? '手动录入' : '同步数据'}</p>
                      </div>
                    </div>
                    
                    <div className="flex items-center gap-6 md:gap-10 bg-space-obsidian/30 p-3 md:p-0 md:bg-transparent rounded-xl">
                      <div className="text-center md:text-right">
                        <p className="text-xs text-text-secondary/60 mb-1">距离</p>
                        <p className="font-mono text-xl font-bold text-electric-blue">{log.distance_km}<span className="text-sm font-sans text-text-secondary">km</span></p>
                      </div>
                      <div className="text-center md:text-right">
                        <p className="text-xs text-text-secondary/60 mb-1">配速</p>
                        <p className="font-mono text-lg font-bold">{log.avg_pace}</p>
                      </div>
                      <div className="text-center md:text-right">
                        <p className="text-xs text-text-secondary/60 mb-1">时长</p>
                        <p className="font-mono text-lg font-bold">{log.duration_min}m</p>
                      </div>
                      <div className="text-center md:text-right hidden sm:block w-12">
                        <p className="text-xs text-text-secondary/60 mb-1">RPE</p>
                        <p className={clsx("font-mono text-lg font-bold", log.rpe > 7 ? 'text-status-stop' : log.rpe > 4 ? 'text-status-caution' : 'text-status-go')}>{log.rpe}</p>
                      </div>
                    </div>
                    {/* 移动端RPE显示和反馈区域 */}
                    <div className="w-full mt-4 pt-4 border-t border-white/5 flex flex-col sm:flex-row sm:items-center justify-between gap-3">
                      <div className="sm:hidden flex items-center justify-between">
                        <span className="text-sm text-text-secondary/80">疲劳度 (RPE)</span>
                        <span className={clsx("font-mono font-bold", log.rpe > 7 ? 'text-status-stop' : log.rpe > 4 ? 'text-status-caution' : 'text-status-go')}>{log.rpe}/10</span>
                      </div>
                      
                      <div className="flex items-center justify-between w-full sm:w-auto sm:gap-4">
                        <span className="text-xs text-text-secondary/80">
                          {feedbackState[log.id] ? '已收到您的反馈。' : 'AI 建议对你有用吗？'}
                        </span>
                        <div className="flex gap-2">
                          <button 
                            onClick={() => handleFeedback(log.id, 'useful')}
                            disabled={feedbackState[log.id]}
                            className={clsx(
                              "px-3 py-1.5 rounded-full text-xs transition-all active:scale-95",
                              feedbackState[log.id] ? "bg-status-go/20 text-status-go opacity-50 cursor-not-allowed" : "bg-carbon hover:bg-white/10"
                            )}
                          >
                            👍 有用
                          </button>
                          <button 
                            onClick={() => handleFeedback(log.id, 'not_useful')}
                            disabled={feedbackState[log.id]}
                            className={clsx(
                              "px-3 py-1.5 rounded-full text-xs transition-all active:scale-95",
                              feedbackState[log.id] ? "bg-carbon opacity-30 cursor-not-allowed" : "bg-carbon hover:bg-white/10"
                            )}
                          >
                            👎 无用
                          </button>
                        </div>
                      </div>
                    </div>
                  </div>
                );
              })}
              
              {logsData?.items.length === 0 && (
                <div className="text-center text-text-secondary/60 py-20 bg-space-obsidian/20 rounded-2xl border border-dashed border-white/10">
                  暂无记录，今日宜出发。
                </div>
              )}
            </div>
          </section>
        </div>
      </div>
    </div>
  );
}