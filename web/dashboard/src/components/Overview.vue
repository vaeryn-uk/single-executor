<template>
  <v-card elevation="2">
    <v-card-title>Signatures</v-card-title>
    <v-card-subtitle>The latest 5 signatures that the running network has provided to the external blockchain.</v-card-subtitle>
    <v-list>
      <v-list-item v-for="(s, id) in signatures" :key="id">
        <pre>{{ s }}</pre>
      </v-list-item>
    </v-list>
  </v-card>
</template>

<script lang="ts">
import { Component, Prop, Vue } from 'vue-property-decorator';
import axios from "axios";

@Component
export default class Overview extends Vue {
  signatures : any = []
  mounted() {
    window.setInterval(
        () => {
          axios.get('http://localhost:8080')
              .then((response) => {
                this.signatures = response.data.sort((x : any, y : any) => {
                  if (x.signedAt === y.signedAt) return 0;

                  return x.signedAt > y.signedAt ? -1 : 1;
                }).slice(0, 5)
              })
        },
        500
    )
  }
}
</script>

<!-- Add "scoped" attribute to limit CSS to this component only -->
<style scoped lang="scss">

</style>
