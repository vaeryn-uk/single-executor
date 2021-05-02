import Vue from 'vue'
import VueRouter from 'vue-router'

// Page components.
import Overview from "../components/Overview.vue";
import Nodes from "../components/Nodes.vue";
import Network from "../components/Network.vue";

Vue.use(VueRouter)

const routes = [
    { path: '/', component: Overview },
    { path: '/nodes', component: Nodes },
    { path: '/network', component: Network },
]

export default new VueRouter({
    routes
})
