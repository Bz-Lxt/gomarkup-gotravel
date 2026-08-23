/** @type {import('tailwindcss').Config} */
export default {
  content: ['./index.html', './src/**/*.{vue,ts}'],
  theme: {
    extend: {
      colors: {
        ink: '#0E1612',
        moss: '#2F6F4E',
        lantern: '#E8A04A',
        paper: '#F3E8D4',
        clay: '#C46A2B',
        dusk: '#1A2A22',
      },
      fontFamily: {
        display: ['Fraunces', 'Noto Serif SC', 'serif'],
        sans: ['Outfit', 'Noto Sans SC', 'sans-serif'],
      },
    },
  },
  plugins: [],
}
