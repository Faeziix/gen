# Project: gen

## Overview
- **Type**: CLI tool
- **Stack**: Bun + TypeScript, `@google/genai` SDK
- **Package Manager**: bun
- **Started**: 2026-05-01

## Architecture Decisions
- `bin/gen.ts` — commander entry point, shebang `#!/usr/bin/env bun`
- `src/generate.ts` — core: builds multipart contents[], streams response, saves images
- `src/io.ts` — file I/O: read image → base64 inlineData, write binary output
- `src/client.ts` — GoogleGenAI factory, validates GEMINI_API_KEY
- `src/types.ts` — GenerateOptions interface

## Preferences & Rules
- Use `bun` always, never npm/npx
- No .env loader — GEMINI_API_KEY must be exported in shell
- Output images go to `out/` by default (gitignored)
- Model commentary streamed to stderr; file paths printed to stdout

## Patterns & Conventions
- Reference images passed via `-i <path>` (repeatable)
- Multi-image: all images sent as `inlineData` parts before the text prompt part
- Output prefix via `-o`; suffix `-<idx>.ext` appended automatically
- `--json` flag for machine-readable `{ files: [...] }` output

## Learnings & Corrections

## Dependencies & Tooling
- `@google/genai` ^1.0.0 — Gemini SDK
- `commander` — CLI arg parsing
- `kleur` — colored stderr output
- `mime` — MIME type detection and extension lookup

## Component Registry
- `src/client.ts` — createClient()
- `src/io.ts` — readImageAsInlineData(), saveBinary()
- `src/generate.ts` — generate()
- `bin/gen.ts` — CLI entry

## Current State
- Scaffold complete, deps installed
- Ready for testing with GEMINI_API_KEY
