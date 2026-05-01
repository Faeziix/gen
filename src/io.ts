import { readFileSync, mkdirSync, writeFileSync, existsSync } from "fs";
import { dirname } from "path";
import mime from "mime";

export interface InlineImagePart {
  inlineData: {
    mimeType: string;
    data: string;
  };
}

export function readImageAsInlineData(imagePath: string): InlineImagePart {
  if (!existsSync(imagePath)) {
    console.error(`Image not found: ${imagePath}`);
    process.exit(1);
  }
  const mimeType = mime.getType(imagePath);
  if (!mimeType || !mimeType.startsWith("image/")) {
    console.error(`Not a supported image type: ${imagePath}`);
    process.exit(1);
  }
  const data = readFileSync(imagePath).toString("base64");
  return { inlineData: { mimeType, data } };
}

export function saveBinary(filePath: string, data: Buffer): void {
  mkdirSync(dirname(filePath), { recursive: true });
  writeFileSync(filePath, data);
}
