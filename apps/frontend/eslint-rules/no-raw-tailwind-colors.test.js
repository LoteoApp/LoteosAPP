import { describe, it } from 'vitest'
import { RuleTester } from 'eslint'
import babelParser from '@babel/eslint-parser'
import rule from './no-raw-tailwind-colors.js'

// vitest doesn't expose describe/it as globals in this project's config, but
// RuleTester.run relies on them being global; wire them up explicitly.
RuleTester.describe = describe
RuleTester.it = it

const ruleTester = new RuleTester({
  languageOptions: {
    ecmaVersion: 2022,
    sourceType: 'module',
    parser: babelParser,
    parserOptions: {
      requireConfigFile: false,
      babelOptions: {
        presets: ['@babel/preset-react', '@babel/preset-typescript'],
      },
    },
  },
})

// Filename must end in .tsx so @babel/preset-typescript enables JSX parsing,
// matching how eslint.config.js lints real .tsx files.
const filename = 'Component.tsx'

ruleTester.run('no-raw-tailwind-colors', rule, {
  valid: [
    { code: '<div className="text-destructive" />', filename },
    { code: '<div className="bg-primary text-muted-foreground" />', filename },
    { code: '<div className={`p-2 ${active ? "bg-primary" : "bg-secondary"}`} />', filename },
    { code: '<div className={condition ? "text-destructive" : "text-foreground"} />', filename },
    { code: '<div className={active && "bg-primary"} />', filename },
    { code: '<div className={[base, "text-foreground"]} />', filename },
    { code: '<div className={{ "text-foreground": active, [dynamicKey]: true }} />', filename },
    { code: 'cn("p-2", "text-foreground")', filename },
    { code: 'clsx("p-2", isActive && "bg-primary")', filename },
    { code: 'twMerge("p-2", "text-foreground")', filename },
    { code: '<div className={someClassNameVariable} />', filename },
    { code: '<div id="text-rose-300" />', filename },
  ],
  invalid: [
    {
      code: '<div className="text-rose-300" />',
      filename,
      errors: [{ messageId: 'rawColor', data: { token: 'text-rose-300' } }],
    },
    {
      code: '<div className={`p-2 bg-slate-950`} />',
      filename,
      errors: [{ messageId: 'rawColor', data: { token: 'bg-slate-950' } }],
    },
    {
      code: '<div className={active ? "text-rose-300" : "text-emerald-500"} />',
      filename,
      errors: [
        { messageId: 'rawColor', data: { token: 'text-rose-300' } },
        { messageId: 'rawColor', data: { token: 'text-emerald-500' } },
      ],
    },
    {
      code: '<div className={active && "bg-cyan-400"} />',
      filename,
      errors: [{ messageId: 'rawColor', data: { token: 'bg-cyan-400' } }],
    },
    {
      code: '<div className={[base, "text-rose-300"]} />',
      filename,
      errors: [{ messageId: 'rawColor', data: { token: 'text-rose-300' } }],
    },
    {
      code: '<div className={{ "text-rose-300": active }} />',
      filename,
      errors: [{ messageId: 'rawColor', data: { token: 'text-rose-300' } }],
    },
    {
      code: '<div className={{ "bg-amber-400": active }} />',
      filename,
      errors: [{ messageId: 'rawColor', data: { token: 'bg-amber-400' } }],
    },
    {
      code: 'cn("p-2", "text-rose-300")',
      filename,
      errors: [{ messageId: 'rawColor', data: { token: 'text-rose-300' } }],
    },
    {
      code: 'clsx("p-2", isActive && "bg-cyan-400")',
      filename,
      errors: [{ messageId: 'rawColor', data: { token: 'bg-cyan-400' } }],
    },
    {
      code: 'twMerge("p-2", "text-emerald-500")',
      filename,
      errors: [{ messageId: 'rawColor', data: { token: 'text-emerald-500' } }],
    },
    {
      code: '<div className="hover:text-rose-300" />',
      filename,
      errors: [{ messageId: 'rawColor', data: { token: 'hover:text-rose-300' } }],
    },
  ],
})
