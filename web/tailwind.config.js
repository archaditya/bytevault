/** @type {import('tailwindcss').Config} */
module.exports = {
  darkMode: ["class"],
  content: [
    "./app/**/*.{ts,tsx}",
    "./components/**/*.{ts,tsx}",
    "./features/**/*.{ts,tsx}",
  ],
  theme: {
    container: {
      center: true,
      padding: "1.5rem",
      screens: { "2xl": "1400px" },
    },
    extend: {
      fontFamily: {
        sans: ["var(--font-geist-sans)", "system-ui", "sans-serif"],
        mono: ["var(--font-geist-mono)", "ui-monospace", "monospace"],
      },
      colors: {
        bg: {
          DEFAULT: "#0A0A0B",
          surface: "#131316",
          raised: "#1A1A1E",
          overlay: "#1F1F23",
        },
        border: {
          DEFAULT: "#1F1F23",
          subtle: "#17171A",
          strong: "#2C2C32",
        },
        ink: {
          DEFAULT: "#FAFAFA",
          muted: "#8A8F98",
          faint: "#5C5F66",
        },
        accent: {
          DEFAULT: "#5E6AD2",
          dim: "#454E9E",
          bright: "#7C87E8",
        },
        live: {
          DEFAULT: "#F5A623",
          dim: "#8A6420",
        },
        success: { DEFAULT: "#4CB782", dim: "#1E3A2C" },
        danger: { DEFAULT: "#E5484D", dim: "#3A1F21" },
        info: { DEFAULT: "#5E9DD2", dim: "#1E2D3A" },
      },
      borderRadius: {
        sm: "5px",
        DEFAULT: "8px",
        md: "10px",
        lg: "14px",
        xl: "18px",
      },
      keyframes: {
        "pulse-live": {
          "0%, 100%": { opacity: "1" },
          "50%": { opacity: "0.45" },
        },
        "chunk-fail": {
          "0%, 100%": { backgroundColor: "transparent" },
          "50%": { backgroundColor: "rgba(229,72,77,0.55)" },
        },
        "fade-up": {
          "0%": { opacity: "0", transform: "translateY(8px)" },
          "100%": { opacity: "1", transform: "translateY(0)" },
        },
        shimmer: {
          "0%": { backgroundPosition: "-200% 0" },
          "100%": { backgroundPosition: "200% 0" },
        },
      },
      animation: {
        "pulse-live": "pulse-live 1.6s ease-in-out infinite",
        "chunk-fail": "chunk-fail 0.9s ease-in-out 2",
        "fade-up": "fade-up 0.4s ease-out",
        shimmer: "shimmer 2.5s linear infinite",
      },
      backgroundImage: {
        "grid-pattern":
          "linear-gradient(rgba(255,255,255,0.035) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.035) 1px, transparent 1px)",
      },
    },
  },
  plugins: [require("tailwindcss-animate")],
};
