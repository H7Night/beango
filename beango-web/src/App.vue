<template>
  <v-app>
    <v-container fluid>
      <v-tabs v-model="activeTab" bg-color="primary" dark>
        <v-tab value="log">日志</v-tab>
        <v-tab value="data">数据</v-tab>
      </v-tabs>

      <v-window v-model="activeTab">
        <!-- 日志 Tab -->
        <v-window-item value="log">
          <v-container class="fill-height align-center justify-center" fluid>
            <v-row class="w-100" justify="center" align="center">
              <v-col cols="12" md="8">
                <!-- 上传控件 -->
                <v-row align="center" justify="center" class="mb-4" no-gutters>
                  <v-col cols="4">
                    <v-combobox v-model="selected" :items="['alipay', 'wechat']" :label="!selected ? '选择类型' : ''" dense
                      density="compact" hide-details style="min-height: 44px; height: 44px; width: 100%;" />
                  </v-col>
                  <v-col cols="4">
                    <v-file-input :label="!file ? '上传文件' : ''" dense density="compact" v-model="file"
                      :disabled="!selected" accept=".csv,.xlsx" hide-details :show-size="false" prepend-icon=""
                      style="min-height: 44px; height: 44px; width: 100%;" class="custom-file-input">
                      <template #selection>
                        <span class="file-label" v-if="file">
                          <span style="margin-left: 4px">
                            {{
                              file.name.length > 8
                                ? file.name.slice(0, 6) + "..."
                                : file.name
                            }}
                          </span>
                        </span>
                      </template>
                    </v-file-input>
                  </v-col>
                  <v-col cols="2">
                    <v-btn color="primary" class="mt-1" :disabled="!file || !selected"
                      style="height: 44px; width: 100%;" @click="uploadFile">
                      上传
                    </v-btn>
                  </v-col>
                </v-row>

                <!-- 日志 -->
                <v-card class="mt-4" outlined>
                  <v-card-title class="d-flex justify-space-between align-center">
                    输出日志
                    <v-btn small color="error" variant="text" @click="clearLog">清空</v-btn>
                  </v-card-title>
                  <v-card-text style="max-height: 400px; overflow-y: auto">
                    <pre style="font-family: monospace; font-size: 14px; margin: 0; text-align: left;"
                      v-html="highlightedOutput"></pre>
                  </v-card-text>
                </v-card>
              </v-col>
            </v-row>
          </v-container>
        </v-window-item>

        <!-- 数据 Tab -->
        <v-window-item value="data">
          <v-container fluid>
            <v-row>
              <!-- 左边目录树 -->
              <v-col cols="3">
                <v-card outlined>
                  <v-card-title>目录结构</v-card-title>
                  <v-card-text style="max-height: 600px; overflow-y: auto">
                    <v-treeview :items="fileTree" item-title="name" item-value="path" activatable open-on-click
                      @update:active="handleFileClick" />
                  </v-card-text>
                </v-card>
              </v-col>

              <!-- 右边文件内容 -->
              <v-col cols="9">
                <v-card outlined>
                  <v-card-title>文件内容</v-card-title>
                  <v-card-text style="max-height: 600px; overflow-y: auto">
                    <pre style="font-family: monospace; font-size: 14px; white-space: pre-wrap;">
              {{ fileContent }}
            </pre>
                  </v-card-text>
                </v-card>
              </v-col>
            </v-row>
          </v-container>
        </v-window-item>
      </v-window>
    </v-container>
  </v-app>
</template>

<script lang="ts" setup>
import { ref, computed, onMounted } from "vue";
import axios from "axios";

const activeTab = ref("log");

// 上传相关
const selected = ref<string | null>(null);
const file = ref<File | null>(null);
const output = ref<string>("");

// 数据模块相关
const fileTree = ref<any[]>([]);
const fileContent = ref<string>("");

// JSON 高亮
const highlightedOutput = computed(() => {
  if (!output.value) return "";
  let json = output.value
    .replace(/(&)/g, "&amp;")
    .replace(/(>)/g, "&gt;")
    .replace(/(<)/g, "&lt;")
    .replace(
      /("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g,
      (match) => {
        let cls = "number";
        if (/^"/.test(match)) {
          if (/:$/.test(match)) {
            cls = "key";
          } else {
            cls = "string";
          }
        } else if (/true|false/.test(match)) {
          cls = "boolean";
        } else if (/null/.test(match)) {
          cls = "null";
        }
        return `<span class="${cls}">${match}</span>`;
      }
    );
  return json;
});

// 上传文件
const uploadFile = async () => {
  if (!file.value || !selected.value) return;
  const formData = new FormData();
  formData.append("file", file.value);

  let url = "";
  if (selected.value === "alipay") {
    url = "http://127.0.0.1:10777/upload/alipay_csv";
  } else if (selected.value === "wechat") {
    url = "http://127.0.0.1:10777/upload/wechat_csv";
  }

  try {
    const response = await axios.post(url, formData, {
      headers: { "Content-Type": "multipart/form-data" },
      responseType: "json",
    });
    output.value = JSON.stringify(response.data, null, 2);

    // 上传成功后，刷新文件树
    loadFileTree();
  } catch (error: any) {
    output.value = `请求失败：${error?.response?.data || error.message}`;
  }
};

// 加载目录树
const loadFileTree = async () => {
  try {
    const response = await axios.get("http://127.0.0.1:10777/api/files/tree");
    fileTree.value = response.data.fileTree || [];
  } catch (error) {
    console.error("获取目录树失败", error);
  }
};

// 点击文件时获取内容
const handleFileClick = async (active: string[]) => {
  if (!active || active.length === 0) return;
  const path = active[0];
  try {
    const response = await axios.get("http://127.0.0.1:10777/api/files/content", {
      params: { path },
    });
    fileContent.value = response.data.content;
  } catch (error) {
    fileContent.value = "读取文件失败";
  }
};

onMounted(() => {
  loadFileTree();
});

const clearLog = () => {
  output.value = "";
};
</script>

<style scoped>
pre {
  background: transparent;
}

.key {
  color: #569cd6;
}

.string {
  color: #d69d85;
}

.number {
  color: #b5cea8;
}

.boolean {
  color: #4ec9b0;
}

.null {
  color: #9cdcfe;
}

.custom-file-input .v-input__prepend {
  display: none !important;
}

.file-label {
  display: flex;
  align-items: center;
  width: 100%;
  overflow: hidden;
  white-space: nowrap;
  text-overflow: ellipsis;
}
</style>
