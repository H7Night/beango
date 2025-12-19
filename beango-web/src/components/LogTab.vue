<template>
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
              style="height: 44px; width: 100%;" @click="emitUpload">
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
</template>

<script lang="ts" setup>
import { ref, computed } from "vue";

interface LogTabEmits {
  (e: 'upload', payload: { file: File, type: string }): void
  (e: 'clear'): void
}

const emit = defineEmits<LogTabEmits>()

// 上传相关
const selected = ref<string | null>(null);
const file = ref<File | null>(null);
const output = ref<string>("");

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

const emitUpload = () => {
  if (!file.value || !selected.value) return;
  emit('upload', { file: file.value, type: selected.value });
};

const clearLog = () => {
  output.value = "";
  emit('clear');
};

// 暴露方法供父组件调用
defineExpose({
  setOutput: (value: string) => {
    output.value = value;
  }
});
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