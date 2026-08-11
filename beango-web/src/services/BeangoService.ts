import axios from 'axios';

// API 地址：优先环境变量 VITE_API_BASE_URL（部署时可通过 .env 配置），
// 未配置时走同源相对路径（生产环境前端由后端 / 提供，开发模式用 vite proxy）。
const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? '';

export interface AccountMap {
  keyword: string;
  account: string;
  type: string;
}

export const uploadFile = async (file: File, type: 'alipay' | 'wechat') => {
  const formData = new FormData();
  formData.append("file", file);

  const endpoint = type === "alipay" ? "/upload/alipay_csv" : "/upload/wechat_csv";
  const url = `${API_BASE_URL}${endpoint}`;

  const response = await axios.post(url, formData, {
    headers: { "Content-Type": "multipart/form-data" },
    responseType: "json",
  });
  
  return response.data;
};

export const getAllAccountMaps = async (): Promise<AccountMap[]> => {
  const response = await axios.get(`${API_BASE_URL}/account_map`);
  return response.data.data;
};

export const createAccountMap = async (accountMap: AccountMap): Promise<AccountMap> => {
  const response = await axios.post(`${API_BASE_URL}/account_map/create`, accountMap);
  return response.data.data;
};

export const updateAccountMap = async (keyword: string, accountMap: AccountMap): Promise<AccountMap> => {
  const response = await axios.put(`${API_BASE_URL}/account_map/update/${encodeURIComponent(keyword)}`, accountMap);
  return response.data.data;
};

export const deleteAccountMap = async (keyword: string): Promise<string> => {
  const response = await axios.delete(`${API_BASE_URL}/account_map/delete/${encodeURIComponent(keyword)}`);
  return response.data.data;
};
