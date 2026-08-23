import { createRouter, createWebHistory } from 'vue-router'
import { getToken } from './api'

export const router = createRouter({
  history: createWebHistory(),
  routes: [
    { path: '/login', component: () => import('./pages/Login.vue') },
    { path: '/', component: () => import('./pages/Teams.vue') },
    { path: '/trips/:id', component: () => import('./pages/Editor.vue') },
    { path: '/live/:id', component: () => import('./pages/Live.vue') },
    { path: '/journal/:id', component: () => import('./pages/Journal.vue') },
  ],
})

router.beforeEach((to) => {
  if (to.path !== '/login' && !getToken()) return '/login'
})
