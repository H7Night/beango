<template>
  <v-app>
    <v-container class="fill-height align-center justify-center" fluid>
      <v-row class="w-100" justify="center" align="center">
        <v-col cols="12" md="8">
          <v-row align="center" justify="center" class="mb-4" no-gutters>
            <v-col cols="4">
              <v-combobox
                  v-model="selected"
                  :items="['alipay', 'wechat']"
                  label="选择类型"
                  dense
              ></v-combobox>
            </v-col>
            <v-col cols="4">
              <v-file-input
                  label="上传文件"
                  dense
                  v-model="file"
                  :disabled="!selected"
                  accept=".csv"
              ></v-file-input>
            </v-col>
            <v-col cols="2">
              <v-btn
                  color="primary"
                  class="mt-1"
                  :disabled="!file || !selected"
                  @click="uploadFile"
              >
                上传
              </v-btn>
            </v-col>
          </v-row>

          <v-textarea
              label="输出结果"
              rows="20"
              auto-grow
              v-model="output"
              outlined
          ></v-textarea>
        </v-col>
      </v-row>
    </v-container>
  </v-app>
</template>

<script lang="ts" setup>
import {ref} from 'vue'
import axios from 'axios'

const selected = ref<string | null>(null)
const file = ref<File | null>(null)
const output = ref<string>('')

const uploadFile = async () => {
  if (!file.value || !selected.value) return

  const formData = new FormData()
  formData.append('file', file.value)

  const url =
      selected.value === 'alipay'
          ? 'http://127.0.0.1:10777/upload/alipay_csv'
          : 'http://127.0.0.1:10777/upload/wechat_csv'

  try {
    const response = await axios.post(url, formData, {
      headers: {'Content-Type': 'multipart/form-data'},
      responseType: 'text'
    })
    output.value = response.data
  } catch (error: any) {
    output.value = `请求失败：${error?.response?.data || error.message}`
  }
}
</script>

<style scoped>
.v-textarea {
  margin-top: 20px;
}
</style>
