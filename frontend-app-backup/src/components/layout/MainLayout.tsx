import { Outlet, NavLink } from 'react-router-dom';
import { Home, Activity, User } from 'lucide-react';
import { clsx } from 'clsx';

export default function MainLayout() {
  const navItems = [
    { path: '/', label: '今日', icon: Home },
    { path: '/log', label: '记录', icon: Activity },
    { path: '/profile', label: '我的', icon: User },
  ];

  return (
    <div className="flex flex-col h-screen max-w-md mx-auto bg-space-obsidian relative">
      {/* 顶部主内容区，留出底部导航的高度 */}
      <main className="flex-1 overflow-y-auto pb-20">
        <Outlet />
      </main>

      {/* 底部导航栏 */}
      <nav className="absolute bottom-0 w-full bg-space-gray/90 backdrop-blur-md border-t border-carbon">
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