# gen — CLI Reference

## Usage

```
gen [options] <prompt>
gen --stdin [options]
```

## Options

| Flag | Default | Description |
|------|---------|-------------|
| `-i, --image <path...>` | — | Reference image(s). Repeatable. |
| `-o, --out <prefix>` | `out/gen-<timestamp>` | Output path prefix |
| `-n, --count <n>` | `1` | Number of variants |
| `-m, --model <id>` | `gemini-3.1-flash-image-preview` | Gemini model |
| `-a, --aspect <ratio>` | — | e.g. `1:1`, `16:9`, `9:16` |
| `-s, --size <size>` | `1K` | `1K` or `2K` |
| `--stdin` | — | Read prompt from stdin |
| `--json` | — | Output `{ "files": [...] }` to stdout |

## Environment

`GEMINI_API_KEY` must be exported:
```sh
export GEMINI_API_KEY=your_key_here
```

## Examples

```sh
# Single reference, default output
gen "make this dark chocolate" -i product.jpg

# Two references, named output
gen "merge the style of A with the subject of B" -i style.png -i subject.jpg -o out/merged

# 3 variants with aspect ratio
gen "watercolor version" -i ref.jpg -n 3 -a 16:9

# Pipe prompt from file
cat prompt.txt | gen --stdin -i ref.jpg

# Machine-readable output
gen "neon version" -i ref.jpg --json
```

## Install globally

```sh
bun link          # from the gen/ repo dir
bun link gen      # from any other dir
```
