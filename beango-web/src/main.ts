import { createApp } from "vue";
import "./style.css";
import App from "./App.vue";
import "@mdi/font/css/materialdesignicons.css";
import vuetify from "./plugins/vuetify.ts";

const app = createApp(App);
app.use(vuetify);
app.mount("#app");
