#!/usr/bin/env bun
import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { SSEServerTransport } from "@modelcontextprotocol/sdk/server/sse.js";
import express from "express";
import { z } from "zod";
import { mkdirSync, resolve as fsResolve } from "node:fs";
import { resolve } from "node:path";
import { generate } from "../src/generate.js";

const PORT = Number(process.env.PORT ?? 3000);
const BASE_URL = process.env.BASE_URL ?? `http://localhost:${PORT}`;
const IMAGES_DIR = resolve("public/images");

mkdirSync(IMAGES_DIR, { recursive: true });

const MODEL_ALIASES: Record<string, string> = {
  flash: "gemini-3.1-flash-image-preview",
  pro: "gemini-3-pro-image-preview",
  flash25: "gemini-2.5-flash-image",
};

const app = express();
app.use(express.json());
app.use("/images", express.static(IMAGES_DIR));

const transports = new Map<string, SSEServerTransport>();

app.get("/sse", async (req, res) => {
  const transport = new SSEServerTransport("/message", res);
  const server = new McpServer({ name: "gen-image", version: "1.0.0" });

  server.tool(
    "generate_image",
    "Generate an AI image using Gemini and return a URL to the result",
    {
      prompt: z.string().describe("Text prompt describing the image to generate"),
      model: z
        .enum(["flash", "pro", "flash25"])
        .optional()
        .default("flash")
        .describe("Model: flash (default), pro, flash25"),
      aspect: z
        .string()
        .optional()
        .describe("Aspect ratio e.g. 16:9, 1:1, 3:2, 9:16"),
      size: z
        .enum(["512", "1K", "2K", "4K"])
        .optional()
        .default("1K")
        .describe("Output resolution"),
      count: z
        .number()
        .int()
        .min(1)
        .max(4)
        .optional()
        .default(1)
        .describe("Number of variants to generate"),
    },
    async ({ prompt, model, aspect, size, count }) => {
      const ts = Date.now();
      const outPrefix = `${IMAGES_DIR}/${ts}`;
      const resolvedModel = MODEL_ALIASES[model ?? "flash"] ?? model ?? MODEL_ALIASES.flash;

      const files = await generate({
        prompt,
        images: [],
        out: outPrefix,
        count: count ?? 1,
        model: resolvedModel,
        aspect: aspect ?? "",
        size: size ?? "1K",
        json: false,
        thinking: undefined,
        showThoughts: false,
      });

      const urls = files.map((f) => {
        const filename = f.replace(IMAGES_DIR + "/", "");
        return `${BASE_URL}/images/${filename}`;
      });

      return {
        content: urls.map((url) => ({
          type: "text" as const,
          text: url,
        })),
      };
    },
  );

  transports.set(transport.sessionId, transport);
  res.on("close", () => transports.delete(transport.sessionId));

  await server.connect(transport);
});

app.post("/message", async (req, res) => {
  const sessionId = req.query.sessionId as string;
  const transport = transports.get(sessionId);
  if (!transport) {
    res.status(404).send("Session not found");
    return;
  }
  await transport.handlePostMessage(req, res, req.body);
});

app.listen(PORT, () => {
  process.stderr.write(`gen MCP server on :${PORT}\n`);
  process.stderr.write(`SSE:     ${BASE_URL}/sse\n`);
  process.stderr.write(`Images:  ${BASE_URL}/images/\n`);
});
