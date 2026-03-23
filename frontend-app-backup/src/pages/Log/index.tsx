import { useState, useEffect } from 'react';
import { Loader2, Plus } from 'lucide-react';
import { trainingApi } from '../../api/training';
import type { TrainingLogListResponse, BaselineResponse } from '../../api/types';
import { format } from 'date-fns';
import { zhCN } from 'date-fns/locale';

export default function LogPage() {
  const [loading, setLoading] = useState(true);
  const [baseline, setBaseline] = useState<BaselineResponse | null>(null);
  const [logsData, setLogsData] = useState<TrainingLogListResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    async function fetchData() {
      try {
        setLoading(true);
        // 并行请求基线数据和训练日志
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

  if (loading) {
    return (
      <div className="flex flex-col items-center justify-center h-full pt-32 text-text-secondary">
        <Loader2 className="animate-spin mb-4" size={32} />
        <p>正在同步您的成长轨迹...</p>
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

  // 映射运动类型为中文
  const workoutTypeMap: Record<string, string> = {
    'easy_run': '轻松跑',
    'long_run': '长距离耐力跑',
    'interval': '间歇跑',
    'strength': '力量训练',
    'stretch': '拉伸恢复',
    'rest': '休息'
  };

  // 根据恢复等级返回不同颜色
  const getRecoveryColor = (level?: string) => {
    switch(level) {
      case 'high': return 'bg-status-go';
      case 'medium': return 'bg-status-caution';
      case 'low': return 'bg-status-stop';
      default: return 'bg-space-gray';
    }
  };

  return (
    <div className="p-6 space-y-8">
      <header>
        <h1 className="text-2xl font-bold mb-2">成长轨迹</h1>
      </header>

      {/* 基线仪表盘 */}
      {baseline && (
        <section className="bg-carbon rounded-2xl p-5 border border-white/5 shadow-lg">
          <h2 className="text-sm text-text-secondary/80 mb-4 font-medium">当前能力基线</h2>
          
          <div className="space-y-4">
            <div>
              <div className="flex justify-between text-xs mb-2">
                <span className="text-text-secondary/60">轻松跑建议配速</span>
                <span className="font-mono text-electric-blue">{baseline.pace_zone?.easy || '暂无数据'}</span>
              </div>
              {/* 简单的装饰条 */}
              <div className="h-2 w-full bg-space-obsidian rounded-full overflow-hidden">
                <div className="h-full bg-electric-blue w-3/4 rounded-full"></div>
              </div>
            </div>

            <div>
              <div className="flex justify-between text-xs mb-2">
                <span className="text-text-secondary/60">恢复状态 (Recovery Level)</span>
                <span className="font-mono uppercase text-text-secondary">{baseline.recovery_level}</span>
              </div>
              <div className="h-2 w-full bg-space-obsidian rounded-full overflow-hidden">
                <div className={`h-full w-full rounded-full opacity-80 ${getRecoveryColor(baseline.recovery_level)}`}></div>
              </div>
            </div>
            
            <div className="pt-2 flex justify-between text-[10px] text-text-secondary/40">
              <span>建议周跑量: {baseline.weekly_volume_range?.min_km}-{baseline.weekly_volume_range?.max_km} km</span>
              <span>更新于 {format(new Date(baseline.updated_at), 'MM-dd HH:mm')}</span>
            </div>
          </div>
        </section>
      )}

      {/* 训练日志流 */}
      <section>
        <div className="flex items-center justify-between mb-4">
          <h2 className="text-sm text-text-secondary/80 font-medium">最近记录</h2>
          <button className="w-8 h-8 rounded-full bg-electric-blue text-white flex items-center justify-center shadow-lg shadow-electric-blue/30 active:scale-95 transition-transform">
            <Plus size={18} />
          </button>
        </div>

        <div className="space-y-4">
          {logsData?.items.map((log) => {
            const dateStr = format(new Date(log.train_date_local), 'MM月dd日', { locale: zhCN });
            return (
              <div key={log.id} className="bg-space-gray rounded-2xl p-4 border border-white/5 border-l-4 border-l-status-go shadow-md">
                <div className="flex justify-between items-start mb-3">
                  <div>
                    <p className="text-xs text-text-secondary/60 mb-1">{dateStr} · {log.source === 'manual' ? '手动记录' : '第三方同步'}</p>
                    <p className="font-bold text-lg">{workoutTypeMap[log.train_type] || log.train_type}</p>
                  </div>
                  <div className="text-right font-mono">
                    <p className="text-xl font-bold text-electric-blue">{log.distance_km} <span className="text-xs text-text-secondary font-sans font-normal">km</span></p>
                  </div>
                </div>
                
                <div className="flex gap-4 text-sm font-mono text-text-secondary/80 mb-4 bg-space-obsidian/40 rounded-lg p-2">
                  <span title="时长">⏱️ {log.duration_min}m</span>
                  <span title="平均配速">⚡️ {log.avg_pace}</span>
                  <span title="主观疲劳度">RPE: {log.rpe}/10</span>
                </div>

                {/* 反馈闭环 */}
                <div className="mt-4 pt-3 border-t border-white/5 flex items-center justify-between">
                  <span className="text-xs text-text-secondary/80">AI 建议对你有用吗？</span>
                  <div className="flex gap-2">
                    <button className="px-4 py-1.5 bg-carbon rounded-full text-xs hover:bg-white/10 active:scale-95 transition-transform">👍 有用</button>
                    <button className="px-4 py-1.5 bg-carbon rounded-full text-xs hover:bg-white/10 active:scale-95 transition-transform">👎 无用</button>
                  </div>
                </div>
              </div>
            );
          })}
          
          {logsData?.items.length === 0 && (
            <div className="text-center text-text-secondary/60 text-sm py-10">
              暂无训练记录，快去开启你的第一跑吧！
            </div>
          )}
        </div>
      </section>
    </div>
  );
}