/** @type {import('tailwindcss').Config} */
export default {
  content: ['./src/**/*.{html,js,svelte,ts}'],
  darkMode: 'class',
  theme: {
    extend: {
      colors: {
        // Use CSS variables with RGB for proper theming
        background: {
          DEFAULT: 'rgb(var(--color-background) / <alpha-value>)',
          secondary: 'rgb(var(--color-background-secondary) / <alpha-value>)',
          tertiary: 'rgb(var(--color-background-tertiary) / <alpha-value>)'
        },
        foreground: {
          DEFAULT: 'rgb(var(--color-foreground) / <alpha-value>)',
          muted: 'rgb(var(--color-foreground-muted) / <alpha-value>)',
          bright: 'rgb(var(--color-foreground-bright) / <alpha-value>)'
        },
        primary: {
          DEFAULT: 'rgb(var(--color-primary) / <alpha-value>)',
          hover: 'rgb(var(--color-primary-hover) / <alpha-value>)'
        },
        border: 'rgb(var(--color-border) / <alpha-value>)',
        accent: {
          cyan: 'rgb(var(--color-accent-cyan) / <alpha-value>)',
          green: 'rgb(var(--color-accent-green) / <alpha-value>)',
          orange: 'rgb(var(--color-accent-orange) / <alpha-value>)',
          red: 'rgb(var(--color-accent-red) / <alpha-value>)',
          purple: 'rgb(var(--color-accent-purple) / <alpha-value>)',
          yellow: 'rgb(var(--color-accent-yellow) / <alpha-value>)'
        },
        // Container states
        running: 'rgb(var(--color-running) / <alpha-value>)',
        stopped: 'rgb(var(--color-stopped) / <alpha-value>)',
        paused: 'rgb(var(--color-paused) / <alpha-value>)',
        restarting: 'rgb(var(--color-restarting) / <alpha-value>)'
      },
      fontFamily: {
        sans: ['Inter', 'system-ui', 'sans-serif'],
        mono: ['JetBrains Mono', 'Fira Code', 'monospace']
      },
      animation: {
        'pulse-slow': 'pulse 3s cubic-bezier(0.4, 0, 0.6, 1) infinite',
        'fade-in': 'fadeIn 0.3s ease-in-out',
        'slide-up': 'slideUp 0.3s ease-out'
      },
      keyframes: {
        fadeIn: {
          '0%': { opacity: '0' },
          '100%': { opacity: '1' }
        },
        slideUp: {
          '0%': { opacity: '0', transform: 'translateY(10px)' },
          '100%': { opacity: '1', transform: 'translateY(0)' }
        }
      }
    }
  },
  plugins: [
    require('@tailwindcss/forms')
  ]
}
