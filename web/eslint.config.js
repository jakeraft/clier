import tsParser from "@typescript-eslint/parser";
import tsPlugin from "@typescript-eslint/eslint-plugin";
import importPlugin from "eslint-plugin-import-x";
import eslintReact from "@eslint-react/eslint-plugin";
import unicorn from "eslint-plugin-unicorn";
import sonarjs from "eslint-plugin-sonarjs";
import security from "eslint-plugin-security";
import noRawTypography from "./eslint-rules/no-raw-typography.js";
import noRawLayout from "./eslint-rules/no-raw-layout.js";

export default [
  // shadcn/ui components are auto-generated — do not lint
  { ignores: ["src/components/ui/**"] },
  unicorn.configs["flat/recommended"],
  sonarjs.configs.recommended,
  security.configs.recommended,
  {
    files: ["src/**/*.{ts,tsx}"],
    ...eslintReact.configs["recommended-typescript"],
    languageOptions: {
      parser: tsParser,
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
        ecmaFeatures: { jsx: true },
      },
    },
    plugins: {
      ...eslintReact.configs["recommended-typescript"].plugins,
      "@typescript-eslint": tsPlugin,
      "import-x": importPlugin,
      "project-conventions": { rules: { "no-raw-typography": noRawTypography, "no-raw-layout": noRawLayout } },
    },
    settings: {
      "import/resolver": {
        typescript: {
          alwaysTryTypes: true,
        },
      },
    },
    rules: {
      ...eslintReact.configs["recommended-typescript"].rules,

      // --- Unicorn overrides ---
      "unicorn/prevent-abbreviations": [
        "error",
        { allowList: { Ref: true, ref: true, Ref$: true, Props: true, props: true } },
      ],
      "unicorn/filename-case": ["error", { case: "kebabCase" }],

      // --- TypeScript strict rules ---
      "@typescript-eslint/no-floating-promises": "error",
      "@typescript-eslint/no-misused-promises": "error",
      "@typescript-eslint/await-thenable": "error",
      "@typescript-eslint/no-unnecessary-type-assertion": "error",
      "@typescript-eslint/no-unsafe-argument": "error",
      "@typescript-eslint/no-unsafe-assignment": "error",
      "@typescript-eslint/no-unsafe-call": "error",
      "@typescript-eslint/no-unsafe-member-access": "error",
      "@typescript-eslint/no-unsafe-return": "error",

      // --- Project conventions ---
      "project-conventions/no-raw-typography": "error",
      "project-conventions/no-raw-layout": "error",

      // --- Console ---
      "no-console": "error",

      // --- Import hygiene ---
      "import-x/no-default-export": "error",
      "import-x/no-duplicates": "error",
      "import-x/first": "error",
      "import-x/no-cycle": "error",
      "import-x/order": [
        "error",
        {
          groups: ["builtin", "external", "internal", ["parent", "sibling", "index"]],
          pathGroups: [
            { pattern: "@/api",                group: "internal", position: "before" },
            { pattern: "@/api/**",             group: "internal", position: "before" },
            { pattern: "@/lib/**",             group: "internal", position: "before" },
            { pattern: "@/components/**",      group: "internal", position: "after" },
            { pattern: "@/hooks/**",           group: "internal", position: "after" },
          ],
          pathGroupsExcludedImportTypes: ["builtin"],
          "newlines-between": "never",
        },
      ],
    },
  },
  // app.tsx is the Vite/React entry — default export required
  {
    files: ["src/app.tsx"],
    rules: { "import-x/no-default-export": "off" },
  },
  {
    files: ["src/lib/logger.ts"],
    rules: { "no-console": "off" },
  },
];
