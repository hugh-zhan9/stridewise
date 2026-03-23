import { useState, useEffect } from 'react';
import { Activity, Loader2, CloudRain, Wind, ThermometerSun } from 'lucide-react';
import { recommendationApi } from '../../api/recommendation';
import type { Recommendation } from '../../api/types';
import { clsx } from 'clsx';

export default function TodayPage() {
  const [loading, setLoading] = useState(true);
  const [recommendation, setRecommendation] = useState<Recommendation | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchData() {
      try {
        setLoading(true);
        const data = await recommendationApi.getToday();
        setRecommendation(data);
      } catch (err: any) {
        setError(err.message || '获取今日建议失败');
      } finally {
        setLoading(false);
      }
    }
    fetchData();
  }, []);

  const handleConsume = async () => {
    if (!recommendation) return;
    try {
      await recommendationApi.consume(recommendation.id);
      alert('已确认执行！');
    } catch (err: any) {
      alert('操作失败：' + err.message);
    }
  };

  const getRiskUI = (level: string) => {
    switch (level) {
      case 'green': return { color: 'bg-status-go', text: '适宜户外', glow: 'bg-status-go/20' };
      case 'yellow': return { color: 'bg-status-caution', text: '需降强度', glow: 'bg-status-caution/20' };
      case 'red': return { color: 'bg-status-stop', text: '不宜户外', glow: 'bg-status-stop/20' };
      default: return { color: 'bg-status-go', text: '适宜户外', glow: 'bg-status-go/20' };
    }
  };

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center h-[70vh] text-text-secondary">
        <Loader2 className="animate-spin mb-4" size={40} />
        <p className="text-lg">AI 正在分析您的身体基线与天气...</p>
      </div>
    );
  }

  if (error || !recommendation) {
    return (
      <div className="p-6 pt-32 text-center">
        <div className="bg-status-stop/10 text-status-stop p-6 rounded-xl border border-status-stop/20 inline-block">
          <p className="text-lg">{error || '暂无数据'}</p>
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

  const rec = recommendation;
  const riskUI = getRiskUI(rec.risk_level);

  const workoutTypeMap: Record<string, string> = {
    'easy_run': '轻松跑',
    'long_run': '长距离耐力跑',
    'interval': '间歇跑',
    'strength': '力量训练',
    'stretch': '拉伸恢复',
    'rest': '强制休息'
  };

  const titlePrefix = rec.should_run ? '🏃‍♂️ 户外' : (rec.workout_type === 'rest' ? '🛑 ' : '🏠 室内');
  const titleStr = `${titlePrefix}${workoutTypeMap[rec.workout_type] || rec.workout_type}`;

  return (
    <div className="space-y-8 animate-in fade-in duration-500">
      <header>
        <h1 className="text-3xl font-bold">今日概览</h1>
        <p className="text-text-secondary mt-2">基于您的恢复状态和外部环境，AI 为您生成的专属训练计划。</p>
      </header>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-8">
        
        {/* 左侧主要内容：AI 建议卡片 */}
        <div className="lg:col-span-2 space-y-8">
          <section className="bg-gradient-to-br from-space-gray to-carbon rounded-3xl p-8 shadow-2xl border border-electric-blue/20 relative overflow-hidden">
            {/* 背景修饰 */}
            <div className={clsx("absolute top-0 right-0 w-64 h-64 blur-3xl rounded-full -translate-y-1/2 translate-x-1/3 pointer-events-none opacity-50", riskUI.glow)}></div>
            
            <div className="flex items-center gap-2 mb-6 relative z-10">
              <div className={clsx("w-3 h-3 rounded-full animate-pulse", riskUI.color)}></div>
              <span className={clsx("text-sm font-bold tracking-wider uppercase", 
                rec.risk_level === 'red' ? 'text-status-stop' : 
                rec.risk_level === 'yellow' ? 'text-status-caution' : 'text-status-go'
              )}>
                RECOMMENDATION
              </span>
            </div>
            
            <h2 className="text-4xl md:text-5xl font-bold mb-10 relative z-10">{titleStr}</h2>
            
            {rec.should_run && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-6 mb-8 relative z-10">
                <div className="bg-space-obsidian/40 backdrop-blur-sm rounded-2xl p-6 border border-white/5">
                  <p className="text-text-secondary/70 text-sm mb-2">目标距离/时长</p>
                  <p className="font-mono text-3xl font-bold text-white">{rec.target_volume}</p>
                </div>
                <div className="bg-space-obsidian/40 backdrop-blur-sm rounded-2xl p-6 border border-white/5">
                  <p className="text-text-secondary/70 text-sm mb-2">建议配速</p>
                  <p className="font-mono text-2xl font-bold text-electric-blue">{rec.intensity_range}</p>
                </div>
              </div>
            )}

            {/* AI 解释区 */}
            <div className="bg-electric-blue/5 rounded-2xl p-6 border border-electric-blue/20 relative z-10">
              <div className="text-text-secondary leading-relaxed space-y-4">
                <p className="text-electric-blue font-bold flex items-center justify-between text-lg">
                  <span>AI 洞察 ({rec.ai_model}):</span>
                  {rec.is_fallback && <span className="text-sm text-status-caution font-normal px-3 py-1 bg-status-caution/10 rounded-full">基于安全规则兜底</span>}
                </p>
                <ul className="space-y-3">
                  {rec.explanation.map((exp, idx) => (
                    <li key={idx} className="flex gap-3">
                      <span className="text-electric-blue mt-1">•</span>
                      <span className="text-[15px] text-white/90">{exp}</span>
                    </li>
                  ))}
                </ul>
              </div>
            </div>
            
            <div className="mt-8 flex flex-col md:flex-row gap-4 relative z-10">
              <button 
                onClick={handleConsume}
                className="px-8 py-4 bg-text-primary text-space-obsidian font-bold rounded-full text-lg hover:bg-gray-200 active:scale-95 transition-all flex items-center justify-center gap-2"
              >
                <Activity size={20} />
                {rec.should_run ? '去执行' : '确认收到'}
              </button>
              <button 
                onClick={() => alert('正在呼叫 AI 重新生成替代方案...')}
                className="px-8 py-4 rounded-full border border-text-secondary/30 text-text-secondary font-medium hover:bg-white/5 active:scale-95 transition-all"
              >
                {rec.alternative_workouts && rec.alternative_workouts.length > 0 ? '查看替代方案' : '换个方案'}
              </button>
            </div>
          </section>
        </div>

        {/* 右侧边栏：天气与贴士 */}
        <div className="space-y-6">
          {/* 天气视窗 */}
          <section className="bg-carbon rounded-3xl p-6 md:p-8 border border-white/5 shadow-xl">
            <h3 className="text-text-secondary font-medium mb-4 flex items-center gap-2">
              <ThermometerSun size={18} />
              实时环境
            </h3>
            <div className="flex items-center justify-between">
              <div>
                <h2 className="text-5xl font-mono font-bold">18°C</h2> 
                <p className="text-text-secondary/80 mt-2 flex items-center gap-2">
                  <span className={clsx("w-2.5 h-2.5 rounded-full shadow-[0_0_8px]", riskUI.color, `shadow-${riskUI.color}`)}></span>
                  {riskUI.text}
                </p>
              </div>
            </div>
            <div className="grid grid-cols-2 gap-4 mt-8">
              <div className="bg-space-obsidian/50 p-4 rounded-xl">
                <p className="text-text-secondary/60 text-xs mb-1 flex items-center gap-1"><CloudRain size={12}/> 降水概率</p>
                <p className="font-mono font-bold text-lg">0%</p>
              </div>
              <div className="bg-space-obsidian/50 p-4 rounded-xl">
                <p className="text-text-secondary/60 text-xs mb-1 flex items-center gap-1"><Wind size={12}/> 风速</p>
                <p className="font-mono font-bold text-lg">2级</p>
              </div>
              <div className="bg-space-obsidian/50 p-4 rounded-xl col-span-2 flex justify-between items-center">
                <p className="text-text-secondary/60 text-sm">空气质量 (AQI)</p>
                <p className="font-mono font-bold text-status-go text-lg">42 · 优</p>
              </div>
            </div>
          </section>

          {/* 贴士区 */}
          {(rec.hydration_tip || rec.clothing_tip) && (
            <section className="bg-carbon rounded-3xl p-6 md:p-8 border border-white/5 shadow-xl">
              <h3 className="text-text-secondary font-medium mb-6">出行贴士</h3>
              <div className="space-y-6">
                {rec.hydration_tip && (
                  <div className="flex gap-4 items-start">
                    <div className="text-2xl bg-space-gray p-2 rounded-xl border border-white/5">💧</div>
                    <p className="text-sm text-text-secondary/90 leading-relaxed mt-1">{rec.hydration_tip}</p>
                  </div>
                )}
                {rec.clothing_tip && (
                  <div className="flex gap-4 items-start">
                    <div className="text-2xl bg-space-gray p-2 rounded-xl border border-white/5">👕</div>
                    <p className="text-sm text-text-secondary/90 leading-relaxed mt-1">{rec.clothing_tip}</p>
                  </div>
                )}
              </div>
            </section>
          )}
        </div>
      </div>
    </div>
  );
}