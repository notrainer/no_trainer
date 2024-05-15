const colors = require('tailwindcss/colors')

module.exports = {
  darkMode: 'class',
  content: ["./templates/**/*.html", "./node_modules/flowbite/**/*.js"], // path to your HTML templates
  theme: {
    extend: {},
    colors: {
      primary: colors.blue
    }
  },
  plugins: [
    require('flowbite/plugin')
  ],
};
