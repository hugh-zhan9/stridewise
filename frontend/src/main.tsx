import { StrictMode } from 'react'
import { createRoot } from 'react-dom/client'
import App from './App.tsx'
import './index.css'
import { setupMock } from './api/mock.ts'

// 在开发环境下启动 API Mock拦截
// setupMock(); // 已关闭 Mock，现在将请求真实的后端接口

createRoot(document.getElementById('root')!).render(
  <StrictMode>
    <App />
  </StrictMode>,
)