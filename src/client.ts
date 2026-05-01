import { GoogleGenAI } from "@google/genai";

export function createClient(): GoogleGenAI {
  const apiKey = process.env["GEMINI_API_KEY"];
  if (!apiKey) {
    console.error("GEMINI_API_KEY not set. Export it before running gen.");
    process.exit(1);
  }
  return new GoogleGenAI({ apiKey });
}
