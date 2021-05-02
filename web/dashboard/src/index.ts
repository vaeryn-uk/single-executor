import Vue from 'vue'
import App from './App.vue'

// Plugins
import store from './plugins/store'
import vuetify from './plugins/vuetify'
import router from './plugins/router'

new Vue({
    vuetify,
    store,
    router,
    render: h => h(App)
}).$mount('#app')
