import type { Config } from "tailwindcss";

/**
 * 与 Ant Design Vue（CSS-in-JS）共存：为所有工具类声明附加 `!important`，减少被 `.ant-*` 覆盖。
 * @see https://tailwindcss.com/docs/configuration#important
 */
export default {
  important: true,
} satisfies Config;
