<template>
  <v-container fluid>
    <v-data-table :items="events" :headers="headers" dense :search="search" group-by="term" show-group-by sort-by="time" hide-default-footer disable-pagination>
      <template v-slot:top>
        <v-text-field
            v-model="search"
            label="Search"
            class="mx-4"
        ></v-text-field>
      </template>
    </v-data-table>
  </v-container>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';

@Component
export default class Overview extends Vue {
  search : string = ""

  get headers() {
    return [
      {text: 'Time', value: 'time', width: "20%"},
      {text: 'Node', value: 'nodeId', width: "10%"},
      {text: 'Term', value: 'term', width: "10%"},
      {text: 'Event', value: 'event'},
    ]
  }

  get events() {
    return this.$store.getters.events.map((e : any) => ({...e, nodeId: `nodeId ${e.nodeId}`}))
  }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">
</style>
