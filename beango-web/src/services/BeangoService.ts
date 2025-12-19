import axios from 'axios';

const API_BASE_URL = 'http://127.0.0.1:10777';

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