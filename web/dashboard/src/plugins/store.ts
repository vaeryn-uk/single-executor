import Vue from 'vue'
import Vuex, {StoreOptions} from 'vuex'
import axios from "axios";

Vue.use(Vuex)

interface ClusterNode {
  id: number
}

interface ClusterInfo {
  nodes: Array<ClusterNode>
}

interface EventData {
  node: number
  event: string
  term: number
  time: string
}

export interface NodeData {
  id: number
  events: EventData[]
  state: string
  blacklist: number[]
}

export type NodesData = { [key: number]: NodeData }

interface StoreState {
  clusterInfo: ClusterInfo | null
  nodes: NodesData
  signatures: any[]
  clusterConfig: string|null
  instanceConfig: string|null
}

export type NodeId = string | number;

export default new Vuex.Store<StoreState>({
  state: {
    nodes: {},
    clusterInfo: null,
    signatures: [],
    instanceConfig: null,
    clusterConfig: null,
  },
  mutations: {
    updateNode(state, {id, data}) {
      if (!state.nodes[id] || JSON.stringify(state.nodes[id]) !== JSON.stringify(data)) {
        Vue.set(state.nodes, id, data)
      }
    },
    clusterInfo(state, info) {
      Vue.set(state, 'clusterInfo', info)
    },
    signature(state, signature) {
      state.signatures.unshift(signature)

      if (state.signatures.length > 10) {
        state.signatures = state.signatures.slice(0, 10)
      }
    },
    clusterConfig(state, config) {
      state.clusterConfig = config;
    },
    instanceConfig(state, config) {
      state.instanceConfig = config;
    }
  },
  getters: {
    nodes(state) : NodesData {
      return state.nodes
    },
    events(state) : EventData[] {
      let events : EventData[] = [];

      for (let node of Object.values(state.nodes)) {
        if (node.events) {
          events = [...events, ...node.events];
        }
      }

      return events.sort((a, b) => {
        if (a.time === b.time) {
          return 0;
        }

        return a.time < b.time ? -1 : 1
      });
    },
    clusterConfig(state) : string|null {
      return state.clusterConfig;
    },
    instanceConfig(state) : string|null {
      return state.instanceConfig;
    },
    hasNodes(state) : boolean {
      return Object.values(state.nodes).length > 0
    },
    node: (state) => (id : NodeId) : NodeData|null => state.nodes[<number>id] || null,
    others: (state) => (id : NodeId) => Object.values<NodeData>(state.nodes).filter((el) => el.id != id),
    networkIsActive: (state, getters) => (to : NodeId, from : NodeId) : boolean => {
      return !getters.node(to)?.blacklist?.includes(parseInt(<string>from, 10)) && getters.node(to)?.state !== 'down' && getters.node(from)?.state !== 'down'
    },
    signatures: (state) => (n : number) => state.signatures.slice(0, n)
  },
  actions: {
    async clusterConfig({commit}) {
      commit('clusterConfig', (await axios.get("/config/cluster")).data)
    },
    async instanceConfig({commit}) {
      commit('instanceConfig', (await axios.get("/config/instance")).data)
    },
    async resolveClusterInfo({dispatch, state}) : Promise<ClusterInfo> {
      let clusterInfo : ClusterInfo;

      if (state.clusterInfo) {
        clusterInfo = state.clusterInfo;
      } else {
        clusterInfo = await dispatch('fetchClusterInfo')
      }

      return clusterInfo;
    },
    async fetchClusterInfo({commit, dispatch, state}) {
      let response = await axios.get("/cluster-info")

      commit('clusterInfo', response.data);

      return state.clusterInfo;
    },
    async streamNodeState({commit, state, dispatch}, id) {
      await dispatch('resolveClusterInfo')

      const evtSource = new EventSource(`/node-state?id=${id}`)

      evtSource.onmessage = function(event) {
        commit('updateNode', {id, data: JSON.parse(event.data)})
      }
    },
    async streamSignatures({commit, state, dispatch}) {
      await dispatch('resolveClusterInfo')

      const evtSource = new EventSource(`http://localhost:8080`)  // TODO: don't hardcode URL

      evtSource.onmessage = function(event) {
        commit('signature', JSON.parse(event.data))
      }
    },
    async streamNodeStates({dispatch}) {
      let info = await dispatch('resolveClusterInfo');
      return Promise.all(info.nodes.map((node : ClusterNode) => dispatch('streamNodeState', node.id)))
    },
  },
  modules: {

  }
})
