#!/usr/bin/env bun
import { Command } from "commander";
import { readFileSync } from "fs";
import { generate } from "../src/generate.js";

const MODEL_ALIASES: Record<string, string> = {
  flash: "gemini-3.1-flash-image-preview",
  pro: "gemini-3-pro-image-preview",
  flash25: "gemini-2.5-flash-image",
};

const program = new Command();

program
  .name("gen")
  .description("Generate AI images via Gemini with optional reference images")
  .argument("[prompt]", "Text prompt describing the desired image")
  .option("-i, --image <path...>", "Reference image(s) — repeatable")
  .option("-o, --out <path>", "Output path prefix", `out/gen-${Date.now()}`)
  .option("-n, --count <n>", "Number of variants to generate", "1")
  .option(
    "-m, --model <id>",
    "Model ID or alias: flash, pro, flash25 (full IDs also accepted)",
    "gemini-3.1-flash-image-preview",
  )
  .option(
    "-a, --aspect <ratio>",
    "Aspect ratio: 1:1, 2:3, 3:2, 3:4, 4:3, 4:5, 5:4, 9:16, 16:9, 21:9 | flash-only: 1:4, 4:1, 1:8, 8:1",
  )
  .option("-s, --size <size>", "Image size: 512 (flash only), 1K, 2K, 4K", "1K")
  .option(
    "--thinking <level>",
    "Thinking level: minimal or high (flash only)",
  )
  .option("--show-thoughts", "Include model thoughts in stderr output (flash only)")
  .option("--stdin", "Read prompt from stdin")
  .option("--json", "Output JSON with generated file paths")
  .action(async (promptArg: string | undefined, options) => {
    let prompt = promptArg ?? "";

    if (options.stdin) {
      prompt = readFileSync(process.stdin.fd, "utf8").trim();
    }

    if (!prompt) {
      console.error("Prompt required. Pass as argument or use --stdin.");
      process.exit(1);
    }

    const model = MODEL_ALIASES[options.model] ?? options.model;

    const files = await generate({
      prompt,
      images: options.image ?? [],
      out: options.out,
      count: parseInt(options.count, 10),
      model,
      aspect: options.aspect ?? "",
      size: options.size,
      json: Boolean(options.json),
      thinking: options.thinking,
      showThoughts: Boolean(options.showThoughts),
    });

    if (options.json) {
      console.log(JSON.stringify({ files }));
    } else {
      for (const f of files) {
        console.log(f);
      }
    }
  });

program.parse();
