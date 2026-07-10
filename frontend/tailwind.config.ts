import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        hive: {
          bg: "var(--hive-bg)",
          surface: "var(--hive-surface)",
          border: "var(--hive-border)",
          blue: "var(--hive-blue)",
          green: "var(--hive-green)",
          red: "var(--hive-red)",
          yellow: "var(--hive-yellow)",
          text: "var(--hive-text)",
          muted: "var(--hive-muted)",
          "canvas-bg": "var(--hive-canvas-bg)",
        },
      },
      transitionProperty: {
        "colors-bg": "background-color, border-color, color",
      },
    },
  },
  plugins: [],
} satisfies Config;
