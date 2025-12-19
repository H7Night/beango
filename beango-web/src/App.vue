<template>
  <v-app>
    <v-container fluid>
      <v-tabs v-model="activeTab" bg-color="primary" dark>
        <v-tab value="log">日志</v-tab>
      </v-tabs>

      <v-window v-model="activeTab">
        <!-- 日志 Tab -->
        <v-window-item value="log">
          <LogTab 
            ref="logTabRef" 
            @upload="handleUpload"
            @clear="handleClearLog"
          />
        </v-window-item>
      </v-window>
    </v-container>
  </v-app>
</template>

<script lang="ts" setup>
import { ref } from "vue";
import LogTab from "./components/LogTab.vue";
import { uploadFile } from "./services/BeangoService";

const activeTab = ref("log");
const logTabRef = ref<InstanceType<typeof LogTab>>();

const handleUpload = async (payload: { file: File, type: string }) => {
  const { file, type } = payload;

  try {
    // 使用 API 服务
    const responseData = await uploadFile(file, type as 'alipay' | 'wechat');
    
    const output = JSON.stringify(responseData, null, 2);
    logTabRef.value?.setOutput(output);
  } catch (error: any) {
    const errorMessage = `请求失败：${error?.response?.data || error.message || error}`;
    logTabRef.value?.setOutput(errorMessage);
  }
};

const handleClearLog = () => {
  logTabRef.value?.setOutput("");
};
</script>