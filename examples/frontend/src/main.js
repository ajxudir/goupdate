import { createApp } from 'vue';
import store from './store';

const app = createApp({
  template: '<div>Frontend Example</div>'
});

app.use(store);
app.mount('#app');
