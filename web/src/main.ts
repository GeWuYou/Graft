import { createApp } from 'vue';

import App from './App.vue';
import { setupApp } from './app/setup-app';
import './styles/global.css';
import 'tdesign-vue-next/es/style/index.css';
import 'uno.css';

const app = createApp(App);

setupApp(app);

app.mount('#app');
