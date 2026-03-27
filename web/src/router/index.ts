import { createRouter, createWebHistory } from "vue-router";
import { useAuthStore } from "../stores/auth";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/login",
      name: "login",
      component: () => import("../views/Login.vue"),
      meta: { public: true },
    },
    {
      path: "/",
      component: () => import("../layouts/MainLayout.vue"),
      meta: { requiresAuth: true },
      children: [
        { path: "", redirect: "/dashboard" },
        { path: "dashboard", name: "dashboard", component: () => import("../views/Dashboard.vue") },
        { path: "instances", name: "instances", component: () => import("../views/Instances.vue") },
        { path: "upstream", name: "upstream", component: () => import("../views/Upstream.vue") },
        { path: "forward", name: "forward", component: () => import("../views/ForwardRules.vue") },
        { path: "logs", name: "logs", component: () => import("../views/Logs.vue") },
        {
          path: "system-logs",
          name: "system-logs",
          component: () => import("../views/SystemLogs.vue"),
        },
        { path: "cache", name: "cache", component: () => import("../views/CacheList.vue") },
        { path: "profile", redirect: "/settings" },
        {
          path: "settings",
          name: "settings",
          component: () => import("../views/SystemSettings.vue"),
        },
      ],
    },
  ],
});

router.beforeEach((to) => {
  const auth = useAuthStore();
  if (to.meta.public) return true;
  if (to.meta.requiresAuth && !auth.token) {
    return { name: "login" };
  }
  return true;
});

export default router;
