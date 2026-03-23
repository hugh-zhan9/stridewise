import { Outlet, NavLink } from 'react-router-dom';
import { Home, Activity, User, Settings, Wind } from 'lucide-react';
import { clsx } from 'clsx';

export default function MainLayout() {
  const navItems = [
    { path: '/', label: '今日建议', icon: Home },
    { path: '/log', label: '成长轨迹', icon: Activity },
    { path: '/profile', label: '我的档案', icon: User },
  ];

  return (
    <div className="flex h-screen bg-space-obsidian text-text-primary overflow-hidden">
      {/* 左侧边导航栏 (Desktop) */}
      <aside className="w-64 bg-carbon border-r border-white/5 flex flex-col hidden md:flex shrink-0">
        <div className="p-6 flex items-center gap-3">
          <div className="w-8 h-8 rounded bg-electric-blue flex items-center justify-center shadow-lg shadow-electric-blue/20">
            <Wind size={20} className="text-white" />
          </div>
          <span className="font-bold text-xl tracking-wider">StrideWise</span>
        </div>
        
        <nav className="flex-1 px-4 space-y-2 mt-4">
          {navItems.map((item) => {
            const Icon = item.icon;
            return (
              <NavLink
                key={item.path}
                to={item.path}
                className={({ isActive }) =>
                  clsx(
                    'flex items-center gap-3 px-4 py-3 rounded-xl transition-all duration-200',
                    isActive 
                      ? 'bg-electric-blue/10 text-electric-blue font-medium' 
                      : 'text-text-secondary/70 hover:bg-white/5 hover:text-text-primary'
                  )
                }
              >
                {({ isActive }) => (
                  <>
                    <Icon size={20} strokeWidth={isActive ? 2.5 : 2} />
                    <span>{item.label}</span>
                  </>
                )}
              </NavLink>
            );
          })}
        </nav>
        
        <div className="p-4 border-t border-white/5">
          <NavLink
            to="/settings"
            className={({ isActive }) =>
              clsx(
                'flex items-center gap-3 px-4 py-3 w-full rounded-xl transition-all duration-200',
                isActive
                  ? 'bg-electric-blue/10 text-electric-blue font-medium'
                  : 'text-text-secondary/70 hover:bg-white/5 hover:text-text-primary'
              )
            }
          >
            {({ isActive }) => (
              <>
                <Settings size={20} strokeWidth={isActive ? 2.5 : 2} />
                <span>设置</span>
              </>
            )}
          </NavLink>
        </div>
      </aside>

      {/* 顶部导航栏 (Mobile Fallback) - 如果浏览器窗口变小，依然提供一个简易顶部栏 */}
      <header className="md:hidden flex items-center justify-between p-4 bg-carbon border-b border-white/5 absolute top-0 w-full z-10">
         <div className="flex items-center gap-2">
            <Wind size={20} className="text-electric-blue" />
            <span className="font-bold">StrideWise</span>
         </div>
      </header>

      {/* 右侧主内容区 */}
      <main className="flex-1 overflow-y-auto relative md:pt-0 pt-16">
        <div className="max-w-7xl mx-auto w-full p-4 md:p-8 pb-24 md:pb-8">
          <Outlet />
        </div>
      </main>

      {/* 底部导航栏 (Mobile Fallback) */}
      <nav className="md:hidden absolute bottom-0 w-full bg-space-gray/90 backdrop-blur-md border-t border-carbon z-10">
        <div className="flex justify-around items-center h-16 px-6">
          {navItems.map((item) => {
            const Icon = item.icon;
            return (
              <NavLink
                key={item.path}
                to={item.path}
                className={({ isActive }) =>
                  clsx(
                    'flex flex-col items-center justify-center w-16 h-full gap-1 transition-colors duration-200',
                    isActive ? 'text-electric-blue' : 'text-text-secondary/60 hover:text-text-secondary'
                  )
                }
              >
                <Icon size={24} strokeWidth={2} />
                <span className="text-[10px] font-medium">{item.label}</span>
              </NavLink>
            );
          })}
        </div>
      </nav>
    </div>
  );
}