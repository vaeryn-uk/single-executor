<template>
  <v-container>
    <v-row>
      <v-col>
        <d3-network v-if="hasNodes" :net-nodes="nodeData" :net-links="linkData" :options="graphOptions" @node-click="selectNode"/>
      </v-col>
      <v-col cols="3">
        <v-card elevation="2" v-if="selectedNode">
          <v-card-title>Selected: {{selectedNode}}</v-card-title>
          <v-card-subtitle>State: <strong>{{nodes[selectedNode] ? nodes[selectedNode].state : 'down'}}</strong></v-card-subtitle>
          <v-card-actions>
            <v-btn text color="primary" @click="startNode(selectedNode)" :disabled="loading" :loading="startLoading">Start</v-btn>
            <v-btn text color="primary" @click="stopNode(selectedNode)" :disabled="loading" :loading="stopLoading">Stop</v-btn>
          </v-card-actions>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';
import {mapGetters} from "vuex";
import {NodesData, NodeData} from "@/plugins/store";
import D3Network from 'vue-d3-network'
import axios from "axios";

@Component({
  computed: mapGetters(["nodes", "hasNodes"]),
  components: { D3Network }
})
export default class Network extends Vue {
  nodes!: NodesData
  hasNodes!: boolean

  linkData : any = []
  nodeData : any = []
  selectedNode : string | null = null
  loading : boolean = false
  startLoading : boolean = false
  stopLoading : boolean = false

  stopNode(id : string) {
    if (this.loading) return;

    this.loading = this.stopLoading = true;
    axios.get(`/node-stop?id=${id}`).finally(() => this.loading = this.stopLoading = false)
  }

  startNode(id : string) {
    if (this.loading) return;

    this.loading = this.startLoading = true;
    axios.get(`/node-start?id=${id}`).finally(() => this.loading = this.startLoading = false)
  }
  selectNode(_ : any, node : any) {
    if (this.loading) return;
    this.selectedNode = node.id
  }

  get graphOptions() {
    return {
      force: 5000,
      nodeLabels: true,
      linkWidth: 5,
      forces: {
        X: 0,
        Y: 0,
        Link: true
      }
    }
  }

  mounted() {
    this.$store.subscribe((mutation) => {
      if (mutation.type === 'updateNode') {
        this.refreshGraphData(this.$store.getters.nodes)
      }
    })
  }

  refreshGraphData(nodes: NodesData) {
    let currentId;

    for (const [id, nodeData] of Object.entries(nodes)) {
      let nodeIndex = this.nodeData.findIndex((n : any) => n.id === id);
      let node;

      if (nodeIndex < 0) {
        node = {
          id: id,
          _size: 50,
        }

        this.nodeData.push(node)
      } else {
        node = this.nodeData[nodeIndex];
      }

      if (nodeData?.state === 'leading') {
        node.name = `${id} (leader)`
      } else {
        node.name = id
      }

      node._cssClass = nodeData ? nodeData.state : 'down';

      this.$set(this.nodeData, nodeIndex, node);

      // Can't just use `id` in the nested loop below. Have to use another variable to keep typescript happy.
      currentId = id;

      for (const [otherId, otherNode] of Object.entries(nodes)) {
        if (otherId === currentId) {
          continue;
        }

        let idParts : Array<string> = [otherId, currentId].sort()

        let id = idParts.join('-');

        let linkIndex = this.linkData.findIndex((l : any) => l.id === id);

        if (linkIndex < 0) {
          let link = {
            id,
            tid: idParts[0],
            sid: idParts[1],
          };

          this.linkData.push(link)
        }
      }
    }
  }
}
</script>

<!-- Styles for our graph -->
<style lang="scss">
  .node {
    fill: #dcfaf3;
    stroke: rgba(18,120,98,.7);
    stroke-width: 3px;
    transition: fill .5s ease;

    &.leading {
       stroke: rgba(120, 18, 91, 0.7);
    }

    &.down {
       stroke: rgba(243, 0, 0, 0.7);
    }
  }

  .link {
    opacity: 0.2;
    stroke: green;
  }
</style>
