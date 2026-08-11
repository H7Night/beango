import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'
import vuetify from 'vite-plugin-vuetify';

// https://vite.dev/config/
export default defineConfig({
    plugins: [
        vue(),
        vuetify({autoImport: true})
    ],
    server: {
        // 开发模式代理到后端（端口可通过环境变量 BEANGO_PORT 覆盖）
        proxy: {
            '/upload': `http://127.0.0.1:${process.env.BEANGO_PORT ?? '10777'}`,
            '/account_map': `http://127.0.0.1:${process.env.BEANGO_PORT ?? '10777'}`,
            '/beango_config': `http://127.0.0.1:${process.env.BEANGO_PORT ?? '10777'}`,
        }
    },
})
