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

export interface NodeData {
  id: number
  state: string
}

export type NodesData = { [key: number]: NodeData }

interface StoreState {
  clusterInfo: ClusterInfo | null
  nodes: NodesData
}

export default new Vuex.Store<StoreState>({
  state: {
    nodes: {},
    clusterInfo: null
  },
  mutations: {
    updateNode(state, {id, data}) {
      Vue.set(state.nodes, id, data)
    },
    clusterInfo(state, info) {
      Vue.set(state, 'clusterInfo', info)
    }
  },
  getters: {
    nodes(state) : NodesData {
      return state.nodes
    },
    hasNodes(state) : boolean {
      return Object.values(state.nodes).length > 0
    }
  },
  actions: {
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
    async fetchNodeState({commit, state, dispatch}, id) {
      await dispatch('resolveClusterInfo')

      try {
        let response = await axios.get(`/node-state?id=${id}`);

        await commit('updateNode', {id, data: response.data})

        return state
      } catch (err) {
        await commit('updateNode', {id, data: null})
      }
    },
    async fetchNodeStates({dispatch}) {
      let info = await dispatch('resolveClusterInfo');
      return Promise.all(info.nodes.map((node : ClusterNode) => dispatch('fetchNodeState', node.id)))
    },
  },
  modules: {

  }
})
