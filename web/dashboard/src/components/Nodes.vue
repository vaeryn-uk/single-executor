<template>
  <v-container fluid>
    <v-row justify="center">
      <v-col>
        <v-card>
          <v-tabs vertical>
            <v-tab v-for="(node, id) in nodes" :key="id">
              Node {{ id }}
            </v-tab>

            <v-tab-item v-for="(node, id) in nodes" :key="id">
              <v-card flat>
                <v-card-text class="node-state-container">
                  <pre class="code-block">{{ node }}</pre>
                </v-card-text>
              </v-card>
            </v-tab-item>
          </v-tabs>
        </v-card>
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
  .node-state-container {
    max-height: 75vh; overflow: auto
  }
</style>
