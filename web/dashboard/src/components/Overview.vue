<template>
  <v-container fluid>
    <v-row>
      <v-col>
        <v-card elevation="2" class="mb-2">
          <v-card-title>Instance configuration</v-card-title>
          <v-card-subtitle>The operational configuration of running watchdog instances</v-card-subtitle>
          <v-card-text>
            <pre class="code-block">{{ instanceConfig }}</pre>
          </v-card-text>
        </v-card>
        <v-card elevation="2">
          <v-card-title>Cluster configuration</v-card-title>
          <v-card-subtitle>The nodes running watchdog instances & their location</v-card-subtitle>
          <v-card-text>
            <pre class="code-block">{{ clusterConfig }}</pre>
          </v-card-text>
        </v-card>
      </v-col>
      <v-col>
        <v-card elevation="2">
          <v-card-title>Signatures</v-card-title>
          <v-card-subtitle>The latest 5 signatures that the running network has provided to the external blockchain.</v-card-subtitle>
          <v-list>
            <v-card elevation="2" class="mr-2 ml-2">
              <template v-for="(s, id) in $store.getters.signatures(5)">
                <v-list-item :key="id" class="pa-2">
                  <pre class="code-block">{{ s }}</pre>
                </v-list-item>

                <v-divider v-if="id < $store.getters.signatures(5).length - 1"></v-divider>
              </template>
            </v-card>
          </v-list>
        </v-card>
      </v-col>
    </v-row>
  </v-container>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';

@Component
export default class Overview extends Vue {
  get clusterConfig() {
    return this.$store.getters.clusterConfig;
  }
  get instanceConfig() {
    return this.$store.getters.instanceConfig;
  }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
</style>
