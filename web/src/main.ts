/* eslint-disable simple-import-sort/imports */
import { createApp } from 'vue';
import TDesign from 'tdesign-vue-next';

import App from './App.vue';
import router from './router';
import { store } from './store';
import i18n from './locales';
import { registerPermissionDirective } from './directives/permission';

import 'tdesign-vue-next/es/style/index.css';
import '@/style/index.less';
import './permission';

const app = createApp(App);

app.use(TDesign);
app.use(store);
app.use(router);
app.use(i18n);
// 权限指令只消费 bootstrap 权限快照，不引入第二套前端鉴权真值。
registerPermissionDirective(app);

app.mount('#app');
