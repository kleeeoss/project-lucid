import type { Config } from "tailwindcss";

const config: Config = {
    content: [
        "./src/pages/**/*.{js,ts,jsx,tsx,mdx}",
        "./src/components/**/*.{js,ts,jsx,tsx,mdx}",
        "./src/app/**/*.{js,ts,jsx,tsx,mdx}",
    ],
    darkMode: "class",
    theme: {
        extend: {
            colors: {
                background: "#090d16",
                surface: "#0f172a",
                border: "#1e293b",
                primary: "#38bdf8",
                danger: "#ef4444",
                warning: "#f59e0b",
                success: "#10b981",
            },
        },
    },
    plugins: [],
};
export default config;