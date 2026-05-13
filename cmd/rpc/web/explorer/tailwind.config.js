/** @type {import('tailwindcss').Config} */
export default {
    content: [
        "./index.html",
        "./src/**/*.{js,ts,jsx,tsx}",
    ],
    theme: {
        extend: {
            fontFamily: {
                sans: ["Inter", "system-ui", "sans-serif"],
            },
            colors: {
                primary: "#45ca46",
                'primary-light': "#86EFAC",
                card: "#1a1a1a",
                background: "#0f0f0f",
                red: "#EF4444",
                navbar: "#141414",
                back: "#6b7280",
                input: '#242424'
            },
        },
    },
    plugins: [],
    safelist: [
        'bg-background',
        'bg-card',
        'text-primary',
        'bg-primary',
        'border-primary-light',
        'text-red',
        'bg-red',
        'bg-navbar',
        'bg-back',
        'bg-input',
    ],
}
