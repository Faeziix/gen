#!/usr/bin/env bun
import { Command } from "commander";
import { readFileSync } from "fs";
import { generate } from "../src/generate.js";

const program = new Command();

program
  .name("gen")
  .description("Generate AI images via Gemini with optional reference images")
  .argument("[prompt]", "Text prompt describing the desired image")
  .option("-i, --image <path...>", "Reference image(s) — repeatable")
  .option("-o, --out <path>", "Output path prefix", `out/gen-${Date.now()}`)
  .option("-n, --count <n>", "Number of variants to generate", "1")
  .option("-m, --model <id>", "Gemini model ID", "gemini-3.1-flash-image-preview")
  .option("-a, --aspect <ratio>", "Aspect ratio, e.g. 1:1, 16:9, 9:16")
  .option("-s, --size <size>", "Image size: 1K or 2K", "1K")
  .option("--stdin", "Read prompt from stdin")
  .option("--json", "Output JSON with generated file paths")
  .action(async (promptArg: string | undefined, options) => {
    let prompt = promptArg ?? "";

    if (options.stdin) {
      prompt = readFileSync("/dev/stdin", "utf8").trim();
    }

    if (!prompt) {
      console.error("Prompt required. Pass as argument or use --stdin.");
      process.exit(1);
    }

    const files = await generate({
      prompt,
      images: options.image ?? [],
      out: options.out,
      count: parseInt(options.count, 10),
      model: options.model,
      aspect: options.aspect ?? "",
      size: options.size,
      json: Boolean(options.json),
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
