import { useState, useEffect } from 'react';
import { Activity, Loader2, CloudRain, Wind } from 'lucide-react';
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
      // 真实业务中这里应该跳转到准备跑步页面，或者直接更新状态
      await recommendationApi.consume(recommendation.id);
      alert('已确认执行！');
    } catch (err: any) {
      alert('操作失败：' + err.message);
    }
  };

  // 根据 risk_level 返回对应的颜色和文本
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
      <div className="flex flex-col items-center justify-center h-full pt-32 text-text-secondary">
        <Loader2 className="animate-spin mb-4" size={32} />
        <p>AI 正在分析您的身体基线与天气...</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="p-6 pt-32 text-center">
        <div className="bg-status-stop/10 text-status-stop p-4 rounded-xl border border-status-stop/20">
          <p>{error}</p>
          <button 
            onClick={() => window.location.reload()} 
            className="mt-4 px-4 py-2 bg-status-stop/20 rounded-lg text-sm"
          >
            重试
          </button>
        </div>
      </div>
    );
  }

  const rec = recommendation!;
  const riskUI = getRiskUI(rec.risk_level);

  // 映射运动类型为中文
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
    <div className="p-6 space-y-6">
      {/* 顶部 - 环视视窗 (Environment Bar) */}
      <section className="flex items-center justify-between bg-carbon rounded-2xl p-4 shadow-lg border border-white/5 relative overflow-hidden">
        {/* 状态呼吸灯 */}
        <div className={clsx("absolute top-0 right-0 w-32 h-32 blur-3xl rounded-full -translate-y-1/2 translate-x-1/2 pointer-events-none", riskUI.glow)}></div>
        
        <div>
          {/* 这里温度应该是从独立的天气API拿，暂时写死或者等天气接口接好 */}
          <h2 className="text-4xl font-mono font-bold">18°C</h2> 
          <p className="text-text-secondary/80 text-sm mt-1 flex items-center gap-1">
            <span className={clsx("w-2 h-2 rounded-full", riskUI.color)}></span>
            {riskUI.text}
          </p>
        </div>
        <div className="text-right text-xs text-text-secondary/60 space-y-1 flex flex-col items-end">
          <p className="flex items-center gap-1"><CloudRain size={12}/> 降水 0%</p>
          <p className="flex items-center gap-1"><Wind size={12}/> 风速 2级</p>
          <p>AQI 42 优</p>
        </div>
      </section>

      {/* 视觉中心 - AI 决策卡片 (The Wise Card) */}
      <section className="bg-gradient-to-br from-space-gray to-carbon rounded-3xl p-6 shadow-2xl border border-electric-blue/20">
        <div className="flex items-center gap-2 mb-4">
          <div className={clsx("w-2 h-2 rounded-full animate-pulse", riskUI.color)}></div>
          <span className={clsx("text-sm font-bold tracking-wider uppercase", 
            rec.risk_level === 'red' ? 'text-status-stop' : 
            rec.risk_level === 'yellow' ? 'text-status-caution' : 'text-status-go'
          )}>
            RECOMMENDATION
          </span>
        </div>
        
        <h1 className="text-3xl font-bold mb-6">{titleStr}</h1>
        
        {rec.should_run && (
          <div className="grid grid-cols-2 gap-4 mb-6">
            <div className="bg-space-obsidian/50 rounded-xl p-4">
              <p className="text-text-secondary/60 text-xs mb-1">目标</p>
              <p className="font-mono text-xl font-bold">{rec.target_volume}</p>
            </div>
            <div className="bg-space-obsidian/50 rounded-xl p-4">
              <p className="text-text-secondary/60 text-xs mb-1">建议配速</p>
              <p className="font-mono text-lg font-bold">{rec.intensity_range}</p>
            </div>
          </div>
        )}

        {/* AI 解释区 */}
        <div className="bg-electric-blue/10 rounded-xl p-4 border border-electric-blue/20">
          <div className="text-sm text-text-secondary leading-relaxed space-y-2">
            <p className="text-electric-blue font-bold flex items-center justify-between">
              <span>AI 洞察 ({rec.ai_model}):</span>
              {rec.is_fallback && <span className="text-xs text-status-caution font-normal">基于安全规则兜底</span>}
            </p>
            <ul className="list-disc pl-4 space-y-1">
              {rec.explanation.map((exp, idx) => (
                <li key={idx}>{exp}</li>
              ))}
            </ul>
          </div>
        </div>

        {/* 贴士区 */}
        {(rec.hydration_tip || rec.clothing_tip) && (
          <div className="mt-4 pt-4 border-t border-white/5 text-xs text-text-secondary/60 space-y-1">
            {rec.hydration_tip && <p>💧 {rec.hydration_tip}</p>}
            {rec.clothing_tip && <p>👕 {rec.clothing_tip}</p>}
          </div>
        )}
      </section>

      {/* 底部操作区 */}
      <section className="flex gap-4 pt-4">
        <button 
          onClick={handleConsume}
          className="flex-1 bg-text-primary text-space-obsidian font-bold py-4 rounded-full text-lg active:scale-95 transition-transform flex items-center justify-center gap-2"
        >
          <Activity size={20} />
          {rec.should_run ? '去执行' : '确认收到'}
        </button>
        <button className="px-6 py-4 rounded-full border border-text-secondary/30 text-text-secondary font-medium active:scale-95 transition-transform">
          {rec.alternative_workouts && rec.alternative_workouts.length > 0 ? '查看替代方案' : '换个方案'}
        </button>
      </section>
    </div>
  );
}