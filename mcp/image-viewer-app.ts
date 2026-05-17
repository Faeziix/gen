import { App } from "@modelcontextprotocol/ext-apps";

interface ImageResult {
  url: string;
  prompt: string;
  model: string;
  aspect?: string;
  size?: string;
  count?: number;
}

const app = new App({ name: "Gen Image Viewer", version: "1.0.0" });

const loadingEl = document.getElementById("loading")!;
const contentEl = document.getElementById("content")!;
const galleryEl = document.getElementById("gallery")!;
const chipsEl = document.getElementById("chips")!;
const regenBtn = document.getElementById("regen") as HTMLButtonElement;
const errorEl = document.getElementById("error")!;

let lastArgs: Omit<ImageResult, "url"> | null = null;

function showImages(items: ImageResult[]) {
  const first = items[0];
  lastArgs = {
    prompt: first.prompt,
    model: first.model,
    aspect: first.aspect,
    size: first.size,
    count: items.length,
  };

  chipsEl.innerHTML = "";
  for (const label of [first.model, first.aspect, first.size].filter(Boolean) as string[]) {
    const chip = document.createElement("span");
    chip.className = "chip";
    chip.textContent = label;
    chipsEl.appendChild(chip);
  }

  galleryEl.innerHTML = "";
  galleryEl.className = `gallery${items.length > 1 ? " multi" : ""}`;

  for (const item of items) {
    const img = document.createElement("img");
    img.src = item.url;
    img.alt = item.prompt;
    img.addEventListener("click", () => app.openLink({ url: item.url }));
    galleryEl.appendChild(img);
  }

  loadingEl.style.display = "none";
  contentEl.style.display = "block";
  errorEl.style.display = "none";
}

function setLoading(on: boolean) {
  loadingEl.style.display = on ? "flex" : "none";
  contentEl.style.display = on ? "none" : "block";
  regenBtn.disabled = on;
}

function showError(msg: string) {
  loadingEl.style.display = "none";
  contentEl.style.display = "block";
  errorEl.style.display = "block";
  errorEl.textContent = msg;
}

app.ontoolresult = (result) => {
  try {
    const items: ImageResult[] = (result.content ?? [])
      .filter((c) => c.type === "text")
      .map((c) => JSON.parse((c as { type: "text"; text: string }).text));
    if (items.length > 0) showImages(items);
    else showError("No images returned");
  } catch {
    showError("Failed to parse image result");
  }
};

regenBtn.addEventListener("click", async () => {
  if (!lastArgs) return;
  setLoading(true);
  try {
    const result = await app.callServerTool({
      name: "generate_image",
      arguments: {
        prompt: lastArgs.prompt,
        model: lastArgs.model,
        aspect: lastArgs.aspect,
        size: lastArgs.size,
        count: lastArgs.count ?? 1,
      },
    });
    const items: ImageResult[] = (result.content ?? [])
      .filter((c) => c.type === "text")
      .map((c) => JSON.parse((c as { type: "text"; text: string }).text));
    if (items.length > 0) showImages(items);
    else showError("No images returned");
  } catch (e) {
    showError(`Regeneration failed: ${e}`);
    setLoading(false);
  }
});

app.connect();
