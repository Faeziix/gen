import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { StreamableHTTPServerTransport } from "@modelcontextprotocol/sdk/server/streamableHttp.js";
import { put } from "@vercel/blob";
import { z } from "zod";
import { readFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join, basename } from "node:path";
import { generate } from "../src/generate.js";
import express from "express";
import type { Request, Response } from "express";

export const maxDuration = 240;

const MODEL_ALIASES: Record<string, string> = {
  flash: "gemini-3.1-flash-image-preview",
  pro: "gemini-3-pro-image-preview",
  flash25: "gemini-2.5-flash-image",
};

function createServer() {
  const server = new McpServer({ name: "gen-image", version: "1.0.0" });

  server.tool(
    "generate_image",
    "Generate AI images using Gemini. Returns images inline. Supports 1–4 images in parallel.",
    {
      prompt: z.string().min(1).describe("Text prompt describing the image to generate"),
      model: z
        .enum(["flash", "pro", "flash25"])
        .optional()
        .default("flash")
        .describe("Model alias — flash (fast, default), pro (higher quality), flash25 (efficient)"),
      aspect: z
        .string()
        .optional()
        .describe("Aspect ratio: 1:1, 16:9, 9:16, 3:2, 2:3, 4:3, 3:4, 4:5, 5:4, 21:9"),
      size: z
        .enum(["512", "1K", "2K", "4K"])
        .optional()
        .default("1K")
        .describe("Output resolution — 512 (flash only), 1K, 2K, 4K"),
      count: z
        .number()
        .int()
        .min(1)
        .max(4)
        .optional()
        .default(1)
        .describe("Number of image variants to generate in parallel (1–4)"),
    },
    async ({ prompt, model, aspect, size, count }) => {
      try {
        const ts = Date.now();
        const outPrefix = join(tmpdir(), `gen-${ts}`);
        const resolvedModel = MODEL_ALIASES[model ?? "flash"] ?? MODEL_ALIASES.flash;

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

        if (files.length === 0) {
          return {
            content: [{ type: "text" as const, text: "No images were generated. Try a different prompt." }],
            isError: true,
          };
        }

        const urls = await Promise.all(
          files.map(async (f) => {
            const data = readFileSync(f);
            const blob = await put(`gen/${basename(f)}`, data, { access: "public" });
            return blob.url;
          }),
        );

        return {
          content: urls.map((url) => ({ type: "text" as const, text: url })),
        };
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        return {
          content: [{ type: "text" as const, text: `Image generation failed: ${message}` }],
          isError: true,
        };
      }
    },
  );

  return server;
}

const app = express();
app.use(express.json());

app.use((req, res, next) => {
  res.setHeader("Access-Control-Allow-Origin", "*");
  res.setHeader("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS");
  res.setHeader("Access-Control-Allow-Headers", "Content-Type, Accept, Mcp-Session-Id");
  if (req.method === "OPTIONS") {
    res.sendStatus(204);
    return;
  }
  next();
});

async function handleMcp(req: Request, res: Response) {
  const transport = new StreamableHTTPServerTransport({ sessionIdGenerator: undefined });
  const server = createServer();
  await server.connect(transport);
  await transport.handleRequest(req, res, req.body);
}

app.post("/api/mcp", handleMcp);
app.delete("/api/mcp", handleMcp);
app.get("/api/mcp", (_req, res) => {
  res.status(405).set("Allow", "POST, DELETE").json({ error: "SSE not supported in stateless mode. Use POST." });
});

export default app;
