// eslint.config.mjs

// Plain typescript-eslint: the assets are framework-agnostic TypeScript. Later
// blocks override the stylistic preset, so per-file relaxations come last.

import js from '@eslint/js'
import globals from 'globals'
import stylistic from '@stylistic/eslint-plugin'
import ts from 'typescript-eslint'

export default ts.config(
  {
    // Lint the assets, not the generated copies.
    ignores: ['node_modules', 'examples/frontendgen/gen', '.frontendgen-check'],
  },
  js.configs.recommended,
  ts.configs.recommended,
  {
    // Everything linted here runs in a webview.
    languageOptions: { globals: globals.browser },
  },
  {
    // tsc (`just typecheck`) checks globals in TypeScript.
    files: ['**/*.ts'],
    rules: { 'no-undef': 'off' },
  },
  stylistic.configs.customize({
    // Matches the s99ab layer style.
    quotes: 'single',
    semi: false,
    braceStyle: '1tbs',
    commaDangle: 'always-multiline',
  }),
  {
    // Repo tooling, not shipped code.
    files: ['scripts/**/*.mjs', 'eslint.config.mjs'],
    languageOptions: { globals: globals.nodeBuiltin },
  },
  {
    files: ['webbench/probe.js'],
    languageOptions: {
      // webbench substitutes this token before injecting the probe.
      globals: { __FRAME_MS__: 'readonly' },
    },
    rules: {
      // Reflowing the measured inner loops would change what is measured.
      '@stylistic/max-statements-per-line': 'off',
      // The probe catches page errors broadly and ignores them.
      '@typescript-eslint/no-unused-vars': ['error', { caughtErrors: 'none' }],
    },
  },
)
