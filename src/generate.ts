import mime from "mime";
import kleur from "kleur";
import { createClient } from "./client.js";
import { readImageAsInlineData, saveBinary } from "./io.js";
import type { GenerateOptions } from "./types.js";

export async function generate(opts: GenerateOptions): Promise<string[]> {
  const ai = createClient();

  const imageParts = opts.images.map(readImageAsInlineData);
  const parts = [...imageParts, { text: opts.prompt }];
  const contents = [{ role: "user", parts }];

  const config = {
    responseModalities: ["IMAGE", "TEXT"],
    imageConfig: {
      ...(opts.aspect ? { aspectRatio: opts.aspect } : {}),
      imageSize: opts.size,
    },
  };

  const allFiles: string[] = [];

  for (let run = 0; run < opts.count; run++) {
    if (opts.count > 1 && !opts.json) {
      process.stderr.write(kleur.dim(`[${run + 1}/${opts.count}] generating...\n`));
    }

    const response = await ai.models.generateContentStream({
      model: opts.model,
      config,
      contents,
    });

    let fileIndex = 0;

    for await (const chunk of response) {
      if (!chunk.candidates?.[0]?.content?.parts) continue;

      for (const part of chunk.candidates[0].content.parts) {
        if (part.inlineData) {
          const ext = mime.getExtension(part.inlineData.mimeType ?? "image/png") ?? "png";
          const suffix = opts.count > 1 ? `-${run}-${fileIndex}` : fileIndex > 0 ? `-${fileIndex}` : "";
          const outPath = `${opts.out}${suffix}.${ext}`;
          const buffer = Buffer.from(part.inlineData.data ?? "", "base64");
          saveBinary(outPath, buffer);
          allFiles.push(outPath);
          if (!opts.json) {
            process.stderr.write(kleur.green(`saved: ${outPath}\n`));
          }
          fileIndex++;
        } else if (part.text && part.text.trim()) {
          process.stderr.write(kleur.dim(`model: ${part.text.trim()}\n`));
        }
      }
    }
  }

  return allFiles;
}
