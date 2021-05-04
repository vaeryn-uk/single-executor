<template>
  <v-app>
    <v-navigation-drawer app>
      <v-list-item>
        <v-list-item-content>
          <v-list-item-title class="title">
            Single executor
          </v-list-item-title>
          <v-list-item-subtitle>
            Dashboard
          </v-list-item-subtitle>
        </v-list-item-content>
      </v-list-item>

      <v-divider></v-divider>

      <v-list
          dense
          nav
      >
        <v-list-item-group
            v-model="selectedPageIndex"
            color="primary"
            @change=""
        >
          <v-list-item
              v-for="(page, i) in pages"
              :key="i"
              :href="$router.resolve(page.path).href"
          >
            <v-list-item-icon>
              <v-icon v-text="page.icon"></v-icon>
            </v-list-item-icon>

            <v-list-item-content>
              <v-list-item-title v-text="page.title"></v-list-item-title>
            </v-list-item-content>
          </v-list-item>
        </v-list-item-group>
      </v-list>
    </v-navigation-drawer>

    <v-app-bar app>
      <v-toolbar-title>{{ currentPage.title }}</v-toolbar-title>
    </v-app-bar>

    <!-- Sizes your content based upon application components -->
    <v-main>

      <!-- Provides the application the proper gutter -->
      <v-container fluid>

        <!-- If using vue-router -->
        <router-view></router-view>
      </v-container>
    </v-main>

    <v-footer app>
      <!-- -->
    </v-footer>
  </v-app>
</template>

<script lang="ts">
import { Component, Vue } from 'vue-property-decorator';

@Component
export default class App extends Vue {
  pages = [
    { title: 'Overview', icon: 'mdi-view-dashboard', path: '/' },
    { title: 'Nodes', icon: 'mdi-cloud-braces', path: '/nodes' },
    { title: 'Network', icon: 'mdi-graph', path: '/network' },
  ]
  selectedPageIndex : number = 0

  mounted() {
    this.selectedPageIndex = this.pageIndexOfCurrentRoute
    this.$store.dispatch('streamNodeStates')
    this.$store.dispatch('streamSignatures')
  }

  get currentPage() {
    return this.pages[this.selectedPageIndex]
  }

  get pageIndexOfCurrentRoute() : number {
    return this.pages.findIndex((p) => p.path == this.$router.currentRoute.path)
  }
}
</script>

<style lang="scss">

</style>
