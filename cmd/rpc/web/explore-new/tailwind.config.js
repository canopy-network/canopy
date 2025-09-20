/** @type {import('tailwindcss').Config} */
export default {
    content: [
        "./index.html",
        "./src/**/*.{js,ts,jsx,tsx}",
    ],
    theme: {
        extend: {
            fontFamily: {
                sans: ["Roboto Flex", "ui-sans-serif", "system-ui", "-apple-system", "Segoe UI", "Roboto", "Noto Sans", "Ubuntu", "Cantarell", "Helvetica Neue", "Arial", "Apple Color Emoji", "Segoe UI Emoji", "Segoe UI Symbol"],
            },
            colors: {
                primary: "#4ADE80",
                'primary-light': "#86EFAC", // Un tono más claro para el borde
                card: "#22232E",
                background: "#1A1B23",
                red: "#EF4444",
                navbar: "#14151C",
                back: "#9CA3AF",
                input: '#2B2C38'
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
