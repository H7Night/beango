import axios from 'axios';

const API_BASE_URL = 'http://127.0.0.1:10777';

export interface AccountMap {
  ID: number;
  keyword: string;
  account: string;
  type: string;
  createdAt: string;
  updated_at: string;
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

export const createAccountMap = async (accountMap: Omit<AccountMap, 'ID' | 'createdAt' | 'updated_at'>): Promise<AccountMap> => {
  const response = await axios.post(`${API_BASE_URL}/account_map/create`, accountMap);
  return response.data.data;
};

export const updateAccountMap = async (id: number, accountMap: Omit<AccountMap, 'ID' | 'createdAt' | 'updated_at'>): Promise<AccountMap> => {
  const response = await axios.put(`${API_BASE_URL}/account_map/update/${id}`, accountMap);
  return response.data.data;
};

export const deleteAccountMap = async (id: number): Promise<number> => {
  const response = await axios.delete(`${API_BASE_URL}/account_map/delete/${id}`);
  return response.data.data;
};