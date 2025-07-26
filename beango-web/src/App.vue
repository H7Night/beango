<template>
  <v-app>
    <v-container class="fill-height align-center justify-center" fluid>
      <v-row class="w-100" justify="center" align="center">
        <v-col cols="12" md="8">
          <v-row align="center" justify="center" class="mb-4" no-gutters>
            <v-col cols="3">
              <v-combobox
                v-model="selected"
                :items="['alipay', 'wechat']"
                :label="!selected ? '选择类型' : ''"
                dense
                density="compact"
                hide-details
                style="min-height: 40px; height: 40px; width: 100%;"
              ></v-combobox>
            </v-col>
            <v-col cols="3">
              <v-select
                v-model="uploadType"
                :items="['csv', 'zip']"
                label="上传格式"
                dense
                density="compact"
                hide-details
                style="min-height: 40px; height: 40px; width: 100%;"
              ></v-select>
            </v-col>
            <v-col cols="3" v-if="uploadType === 'zip'">
              <v-text-field
                v-model="password"
                label="解压密码"
                dense
                density="compact"
                hide-details
                type="password"
                style="min-height: 40px; height: 40px; width: 100%;"
              ></v-text-field>
            </v-col>
            <v-col cols="2">
              <v-btn
                color="primary"
                class="mt-1"
                :disabled="!file || !selected || (uploadType === 'zip' && !password)"
                @click="uploadFile"
              >
                上传
              </v-btn>
            </v-col>
          </v-row>
          <!-- 文件选择框单独一行，居中显示 -->
          <v-row justify="center" align="center" class="mb-2">
            <v-col cols="auto" class="d-flex justify-center">
              <v-file-input
                :label="!file ? '上传文件' : ''"
                dense
                density="compact"
                v-model="file"
                :disabled="!selected"
                :accept="uploadType === 'zip' ? '.zip' : '.csv'"
                hide-details
                :show-size="false"
                style="min-width: 240px; max-width: 400px; width: auto;"
                class="custom-file-input"
              >
                <template #selection>
                  <span class="file-label" v-if="file">
                    <span style="margin-left: 4px;">
                      {{
                        file.name.length > 8
                          ? file.name.slice(0, 6) + '...'
                          : file.name
                      }}
                    </span>
                  </span>
                </template>
              </v-file-input>
            </v-col>
          </v-row>

          <!-- 日志滚动展示 -->
          <v-card class="mt-4" outlined>
            <v-card-title class="d-flex justify-space-between align-center">
              输出日志
              <v-btn small color="error" variant="text" @click="clearLog">清空</v-btn>
            </v-card-title>
            <v-card-text style="max-height: 400px; overflow-y: auto;">
              <pre
                style="font-family: monospace; font-size: 14px; margin: 0; text-align: left;"
                v-html="highlightedOutput"
              ></pre>
            </v-card-text>
          </v-card>
        </v-col>
      </v-row>
    </v-container>
  </v-app>
</template>

<script lang="ts" setup>
import {ref, computed} from 'vue'
import axios from 'axios'

const selected = ref<string | null>(null)
const file = ref<File | null>(null)
const password = ref<string>('')
const uploadType = ref<'csv' | 'zip'>('csv')
const output = ref<string>('')

const highlightedOutput = computed(() => {
  if (!output.value) return ''
  let json = output.value
    .replace(/(&)/g, '&amp;')
    .replace(/(>)/g, '&gt;')
    .replace(/(<)/g, '&lt;')
    .replace(/("(\\u[a-zA-Z0-9]{4}|\\[^u]|[^\\"])*"(\s*:)?|\b(true|false|null)\b|-?\d+(?:\.\d*)?(?:[eE][+\-]?\d+)?)/g, match => {
      let cls = 'number'
      if (/^"/.test(match)) {
        if (/:$/.test(match)) {
          cls = 'key'
        } else {
          cls = 'string'
        }
      } else if (/true|false/.test(match)) {
        cls = 'boolean'
      } else if (/null/.test(match)) {
        cls = 'null'
      }
      return `<span class="${cls}">${match}</span>`
    })
  return json
})

const uploadFile = async () => {
  if (!file.value || !selected.value) return

  const formData = new FormData()
  formData.append('file', file.value)
  if (uploadType.value === 'zip') {
    formData.append('password', password.value)
  }

  let url = ''
  if (selected.value === 'alipay') {
    url = uploadType.value === 'zip'
      ? 'http://127.0.0.1:10777/upload/alipay_zip'
      : 'http://127.0.0.1:10777/upload/alipay_csv'
  } else if (selected.value === 'wechat') {
    url = 'http://127.0.0.1:10777/upload/wechat_csv'
  }

  try {
    const response = await axios.post(url, formData, {
      headers: {'Content-Type': 'multipart/form-data'},
      responseType: 'json'
    })
    output.value = JSON.stringify(response.data, null, 2)
  } catch (error: any) {
    output.value = `请求失败：${error?.response?.data || error.message}`
  }
}

const clearLog = () => {
  output.value = ''
}
</script>

<style scoped>
pre { background: transparent; }
.key { color: #569cd6; }
.string { color: #d69d85; }
.number { color: #b5cea8; }
.boolean { color: #4ec9b0; }
.null { color: #9cdcfe; }
</style>
