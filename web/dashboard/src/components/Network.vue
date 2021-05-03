<template>
  <v-container fluid>
    <v-row>
      <v-col cols="9">
        <v-sheet elevation="5" color="blue-grey lighten-5">
          <d3-network v-if="hasNodes" :net-nodes="nodeData" :net-links="linkData" :options="graphOptions" @node-click="selectNode" @link-click="toggleLink"/>
          <span class="text-caption graph-footnote">Rendered using <a href="https://github.com/emiliorizzo/vue-d3-network">vue-d3-network</a>.</span>
        </v-sheet>
      </v-col>
      <v-col cols="3">
        <v-card elevation="2" v-if="selectedNode">
          <v-card-title>Selected: {{selectedNode}}</v-card-title>
          <v-card-subtitle>State: <strong>{{nodes[selectedNode] ? nodes[selectedNode].state : 'down'}}</strong></v-card-subtitle>
          <v-card-actions>
            <v-btn text color="primary" @click="startNode(selectedNode)" :disabled="loading" :loading="startLoading">Start</v-btn>
            <v-btn text color="warning" @click="stopNode(selectedNode)" :disabled="loading" :loading="stopLoading">Stop</v-btn>
          </v-card-actions>

          <v-divider></v-divider>

          <v-card-title>Links</v-card-title>
          <v-list>
            <v-list-item v-for="other in others(selectedNode)" :key="other.id">
              <div v-if="networkIsActive(selectedNode, other.id)">
                <v-icon color="success">mdi-check-bold</v-icon>
                {{ other.id }}
                <v-btn text color="warning" @click="networkBreak(selectedNode, other.id)"
                       :disabled="!canLinkBeModified(selectedNode, other.id)" :loading="networkLoading[other.id]">Break</v-btn>
              </div>
              <div v-else>
                <v-icon color="error">mdi-close</v-icon>
                {{ other.id }}
                <v-btn text color="primary" @click="networkRepair(selectedNode, other.id)"
                       :disabled="!canLinkBeModified(selectedNode, other.id)" :loading="networkLoading[other.id]">Repair</v-btn>
              </div>
            </v-list-item>
          </v-list>

        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script lang="ts">
import {Component, Vue} from 'vue-property-decorator';
import {mapGetters} from "vuex";
import {NodeId, NodesData, NodeData} from "@/plugins/store";
import D3Network from 'vue-d3-network'
import axios from "axios";

@Component({
  computed: mapGetters(["nodes", "hasNodes", "others", "networkIsActive", "node"]),
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
  networkLoading : { [key: string]: boolean } = {}
  graphWatcher : any = null

  mounted() {
    this.refreshGraphData();

    this.graphWatcher = this.$store.watch(
        (state) => {
          const parts = [];

          // Extract the relevant bits of the state that should trigger a graph update.
          // This is pretty heavy on the browser, and could be improved.
          for (let node of Object.values<any>(state.nodes)) {
            parts.push(node ? {
              id: node.id,
              blacklist: node.blacklist || [],
              leading: node.state === 'leading',
              down: node.state === 'down',
            } : null)
          }

          return JSON.stringify(parts)
        },
        () => this.refreshGraphData()
    )
  }

  destroyed() {
    this.graphWatcher && this.graphWatcher();
  }

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
    this.refreshGraphData()
  }

  toggleLink(_ : any, link : any) {
    if (this.loading || !this.canLinkBeModified(link.source.id, link.target.id)) return;

    if (this.$store.getters.networkIsActive(link.source.id, link.target.id)) {
      this.networkBreak(link.source.id, link.target.id);
    } else {
      this.networkRepair(link.source.id, link.target.id);
    }
  }

  networkBreak(source : NodeId, target : NodeId) {
    this.networkLoading[target] = true

    axios.get(`/network-break?id=${source}&other=${target}`)
        .finally(() => this.networkLoading[target] = false)
  }

  networkRepair(source : NodeId, target : NodeId) {
    this.networkLoading[target] = true

    axios.get(`/network-repair?id=${source}&other=${target}`)
        .finally(() => this.networkLoading[target] = false)
  }

  refreshGraphData() {
    let currentId;
    let nodes = this.$store.getters.nodes;

    for (const [id, nodeData] of Object.entries<NodeData>(nodes)) {
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

      if (node.id === this.selectedNode) {
        node._cssClass += ' selected';
      }

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

        let link;

        if (linkIndex < 0) {
          link = {
            id,
            tid: idParts[0],
            sid: idParts[1],
          };

          this.linkData.push(link)
        } else {
          link = this.linkData[linkIndex]
        }

        if (!this.$store.getters.networkIsActive(currentId, otherId)) {
          link._color = 'red';
        } else {
          link._color = 'green';
        }

        this.$set(this.linkData, linkIndex, link);
      }
    }
  }

  canLinkBeModified(id : NodeId, other : NodeId) : boolean {
    return !this.loading && !this.networkLoading[other] && this.$store.getters.node(id).state !== 'down' && this.$store.getters.node(other).state !== 'down'
  }

  get graphOptions() {
    return {
      force: 5000,
      nodeLabels: true,
      linkWidth: 10,
      forces: {
        X: 0,
        Y: 0,
        Link: true
      }
    }
  }
}
</script>

<!-- Styles for our graph -->
<style lang="scss">
  .net {
    user-select: none;
  }

  .node-label {
    font-weight: bold;
    font-size: 1rem;
  }

  .node {
    fill: #dcfaf3;
    stroke: rgba(18,120,98,.7);
    stroke-width: 3px;
    stroke-opacity: 0.7;
    transition: fill .5s ease;

    &.leading {
      stroke: rgba(120, 18, 91, 0.7);
    }

    &.selected {
      stroke-opacity: 1.0;
      stroke-width: 6px;
    }

    &.down {
      stroke: rgba(243, 0, 0, 0.7);
    }
  }

  .link {
    opacity: 0.3;
  }

  .graph-footnote {
    position: relative;
    left: -3px;
    float: right;
    top: -20px
  }
</style>
