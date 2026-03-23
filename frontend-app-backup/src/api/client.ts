import axios from 'axios';
import type { AxiosResponse, InternalAxiosRequestConfig } from 'axios';

// 从环境变量读取 API URL，如果没有配置则默认指向本地后端的默认端口 8080
const baseURL = import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080';

export const apiClient = axios.create({
  baseURL,
  timeout: 10000,
  headers: {
    'Content-Type': 'application/json',
    // MVP 阶段使用固定的内部 Token
    'X-Internal-Token': 'stridewise-dev-token' 
  },
});

// 请求拦截器
apiClient.interceptors.request.use(
  (config: InternalAxiosRequestConfig) => {
    // 可以在这里统一处理一些请求前逻辑
    return config;
  },
  (error) => {
    return Promise.reject(error);
  }
);

// 响应拦截器
apiClient.interceptors.response.use(
  (response: AxiosResponse) => {
    // 根据您的 OpenAPI 规范，这里可以尝试做 Envelope 的拆包
    // 但为了灵活性，暂且保留整个 response.data 让具体方法去拆
    return response;
  },
  (error) => {
    console.error('API Error:', error.response?.data || error.message);
    // 这里可以接入统一的全局错误提示 Toast
    return Promise.reject(error);
  }
);
