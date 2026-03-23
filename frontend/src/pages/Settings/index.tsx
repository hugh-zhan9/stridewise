import { Bell, Moon, Shield, HelpCircle, LogOut } from 'lucide-react';
import { clsx } from 'clsx';
import { useState } from 'react';

export default function SettingsPage() {
  const [pushEnabled, setPushEnabled] = useState(true);
  const [darkMode, setDarkMode] = useState(true);

  return (
    <div className="space-y-8 animate-in fade-in duration-500 max-w-3xl">
      <header>
        <h1 className="text-3xl font-bold">系统设置</h1>
        <p className="text-text-secondary mt-2">管理您的偏好设置与账号安全。</p>
      </header>

      <div className="space-y-6">
        {/* 常规设置 */}
        <section>
          <h2 className="text-sm font-bold text-text-secondary/80 mb-4 uppercase tracking-wider pl-2">常规偏好</h2>
          <div className="bg-carbon rounded-3xl border border-white/5 shadow-xl overflow-hidden">
            
            <div className="p-6 flex items-center justify-between border-b border-white/5">
              <div className="flex items-center gap-4">
                <div className="w-10 h-10 rounded-xl bg-space-gray flex items-center justify-center text-electric-blue">
                  <Bell size={20} />
                </div>
                <div>
                  <p className="font-bold text-lg">训练提醒</p>
                  <p className="text-sm text-text-secondary/70">每日推送训练建议与恶劣天气警告</p>
                </div>
              </div>
              <button 
                onClick={() => setPushEnabled(!pushEnabled)}
                className={clsx("w-12 h-7 rounded-full relative transition-colors", pushEnabled ? "bg-status-go" : "bg-space-obsidian border border-white/10")}
              >
                <div className={clsx("w-5 h-5 bg-white rounded-full absolute top-1 shadow-sm transition-transform", pushEnabled ? "right-1" : "left-1 opacity-50")}></div>
              </button>
            </div>

            <div className="p-6 flex items-center justify-between">
              <div className="flex items-center gap-4">
                <div className="w-10 h-10 rounded-xl bg-space-gray flex items-center justify-center text-electric-blue">
                  <Moon size={20} />
                </div>
                <div>
                  <p className="font-bold text-lg">深色模式</p>
                  <p className="text-sm text-text-secondary/70">目前系统默认强制深色模式</p>
                </div>
              </div>
              <button 
                onClick={() => alert('当前设计规范强制使用深色模式，此开关为演示UI。')}
                className={clsx("w-12 h-7 rounded-full relative transition-colors bg-status-go")}
              >
                <div className="w-5 h-5 bg-white rounded-full absolute top-1 right-1 shadow-sm"></div>
              </button>
            </div>

          </div>
        </section>

        {/* 关于与安全 */}
        <section>
          <h2 className="text-sm font-bold text-text-secondary/80 mb-4 uppercase tracking-wider pl-2">关于与安全</h2>
          <div className="bg-carbon rounded-3xl border border-white/5 shadow-xl overflow-hidden">
            
            <button className="w-full p-6 flex items-center justify-between border-b border-white/5 hover:bg-white/5 transition-colors text-left">
              <div className="flex items-center gap-4">
                <div className="w-10 h-10 rounded-xl bg-space-gray flex items-center justify-center text-text-secondary">
                  <Shield size={20} />
                </div>
                <p className="font-bold text-lg">隐私协议与免责声明</p>
              </div>
              <span className="text-text-secondary/50">→</span>
            </button>

            <button className="w-full p-6 flex items-center justify-between hover:bg-white/5 transition-colors text-left">
              <div className="flex items-center gap-4">
                <div className="w-10 h-10 rounded-xl bg-space-gray flex items-center justify-center text-text-secondary">
                  <HelpCircle size={20} />
                </div>
                <div>
                  <p className="font-bold text-lg">当前版本</p>
                  <p className="text-sm text-text-secondary/70">StrideWise v1.0.0-beta</p>
                </div>
              </div>
            </button>

          </div>
        </section>

        {/* 危险操作 */}
        <section className="pt-4">
          <button 
            onClick={() => {
              if (confirm('确认退出登录吗？')) {
                alert('已退出！实际接入需清除 Token 并跳回登录页。');
              }
            }}
            className="w-full p-4 rounded-2xl bg-status-stop/10 text-status-stop border border-status-stop/20 font-bold flex items-center justify-center gap-2 hover:bg-status-stop/20 transition-colors"
          >
            <LogOut size={20} />
            退出当前账号
          </button>
        </section>

      </div>
    </div>
  );
}