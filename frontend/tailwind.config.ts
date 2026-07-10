import type { Config } from "tailwindcss";

export default {
  content: ["./index.html", "./src/**/*.{ts,tsx}"],
  theme: {
    extend: {
      colors: {
        hive: {
          bg: "#0D1117",
          surface: "#161B22",
          border: "#30363D",
          blue: "#58A6FF",
          green: "#3ECF8E",
          red: "#F85149",
          yellow: "#D29922",
          text: "#E6EDF3",
          muted: "#8B949E",
        },
      },
    },
  },
  plugins: [],
} satisfies Config;
