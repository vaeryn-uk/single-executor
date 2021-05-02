<template>
  <v-container>
    <v-row justify="center">
      <v-col v-for="(node, id) in nodes" :key="id">
        <v-sheet elevation="3" class="pa-5">
          <v-card-title>
            Node {{ id }}
          </v-card-title>
          <pre v-if="node !== null" class="node-state">{{ node }}</pre>
          <pre v-else>Cannot retrieve data. Node down?</pre>
        </v-sheet>
      </v-col>
    </v-row>
  </v-container>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';

@Component
export default class Nodes extends Vue {
  get nodes() {
    return this.$store.getters.nodes;
  }
}
</script>

<style scoped lang="scss">
.node-state {
  overflow: auto;
  max-height: 25vh;
}
</style>
