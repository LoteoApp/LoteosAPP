import js from '@eslint/js'
import globals from 'globals'
import reactHooks from 'eslint-plugin-react-hooks'
import reactRefresh from 'eslint-plugin-react-refresh'
import babelParser from '@babel/eslint-parser'
import noRawTailwindColors from './eslint-rules/no-raw-tailwind-colors.js'

export default [
  { ignores: ['dist', 'coverage'] },
  js.configs.recommended,
  {
    files: ['**/*.{ts,tsx}'],
    languageOptions: {
      ecmaVersion: 2022,
      sourceType: 'module',
      globals: globals.browser,
      parser: babelParser,
      parserOptions: {
        requireConfigFile: false,
        babelOptions: {
          presets: ['@babel/preset-react', '@babel/preset-typescript'],
        },
      },
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
      local: { rules: { 'no-raw-tailwind-colors': noRawTailwindColors } },
    },
    rules: {
      ...reactHooks.configs['recommended-latest'].rules,
      'react-refresh/only-export-components': ['warn', { allowConstantExport: true }],
      'local/no-raw-tailwind-colors': 'error',
      // Type-only syntax (interfaces, type aliases) isn't understood by
      // these core JS rules without full type info; tsc already covers them.
      'no-unused-vars': 'off',
      'no-undef': 'off',
    },
  },
]
