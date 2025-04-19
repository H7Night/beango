<script setup lang="ts">
import HelloWorld from './components/HelloWorld.vue'
</script>

<template>
  <v-app>
    <v-app-bar app>
      <v-toolbar-title>My Vuetify App</v-toolbar-title>
    </v-app-bar>
    <v-main>
      <v-container>
        <h1>Welcome to Vuetify!</h1>
        <v-btn color="primary">Click Me</v-btn>
        <v-icon>mdi-home</v-icon> </v-container>
    </v-main>
  </v-app>
</template>

<style scoped>
.logo {
  height: 6em;
  padding: 1.5em;
  will-change: filter;
  transition: filter 300ms;
}
.logo:hover {
  filter: drop-shadow(0 0 2em #646cffaa);
}
.logo.vue:hover {
  filter: drop-shadow(0 0 2em #42b883aa);
}
</style>
