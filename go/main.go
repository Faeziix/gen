package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const baseURL = "https://generativelanguage.googleapis.com/v1beta"

var modelAliases = map[string]string{
	"flash":   "gemini-3.1-flash-image-preview",
	"pro":     "gemini-3-pro-image-preview",
	"flash25": "gemini-2.5-flash-image",
}

var extToMime = map[string]string{
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".gif":  "image/gif",
	".webp": "image/webp",
}

var mimeToExt = map[string]string{
	"image/jpeg": "jpg",
	"image/png":  "png",
	"image/gif":  "gif",
	"image/webp": "webp",
}

type stringSlice []string

func (s *stringSlice) String() string     { return strings.Join(*s, ", ") }
func (s *stringSlice) Set(v string) error { *s = append(*s, v); return nil }

// Interactions API (gemini-3.x models)

type inputPart struct {
	Type     string `json:"type"`
	Text     string `json:"text,omitempty"`
	FileURI  string `json:"fileUri,omitempty"`
	URI      string `json:"uri,omitempty"`
	MimeType string `json:"mime_type,omitempty"`
	Data     string `json:"data,omitempty"`
}

type imageConfig struct {
	AspectRatio string `json:"aspect_ratio,omitempty"`
	ImageSize   string `json:"image_size,omitempty"`
}

type thinkingConfig struct {
	ThinkingLevel   string `json:"thinking_level,omitempty"`
	IncludeThoughts bool   `json:"include_thoughts,omitempty"`
}

type generationConfig struct {
	ImageConfig    *imageConfig    `json:"image_config,omitempty"`
	ThinkingConfig *thinkingConfig `json:"thinking_config,omitempty"`
}

type interactionRequest struct {
	Model              string            `json:"model"`
	Input              interface{}       `json:"input"`
	ResponseModalities []string          `json:"response_modalities,omitempty"`
	GenerationConfig   *generationConfig `json:"generation_config,omitempty"`
}

type outputPart struct {
	Type     string `json:"type"`
	Text     string `json:"text"`
	MimeType string `json:"mime_type"`
	Data     string `json:"data"`
}

type interactionResponse struct {
	Outputs []outputPart `json:"outputs"`
}

// Legacy generateContent API (gemini-2.x models)

type legacyPart struct {
	Text       string      `json:"text,omitempty"`
	InlineData *inlineData `json:"inlineData,omitempty"`
}

type inlineData struct {
	MimeType string `json:"mimeType"`
	Data     string `json:"data"`
}

type legacyImageConfig struct {
	AspectRatio string `json:"aspectRatio,omitempty"`
	ImageSize   string `json:"imageSize,omitempty"`
}

type legacyGenerationConfig struct {
	ResponseModalities    []string           `json:"responseModalities"`
	ImageGenerationConfig *legacyImageConfig `json:"imageGenerationConfig,omitempty"`
	ThinkingConfig        *thinkingConfig    `json:"thinkingConfig,omitempty"`
}

type legacyRequest struct {
	Contents []struct {
		Role  string       `json:"role"`
		Parts []legacyPart `json:"parts"`
	} `json:"contents"`
	GenerationConfig legacyGenerationConfig `json:"generationConfig"`
}

type streamChunk struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text       string `json:"text"`
				InlineData *struct {
					MimeType string `json:"mimeType"`
					Data     string `json:"data"`
				} `json:"inlineData"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func uploadFile(apiKey, path string) (string, error) {
	mimeType, ok := extToMime[strings.ToLower(filepath.Ext(path))]
	if !ok {
		return "", fmt.Errorf("unsupported image type: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("cannot read image: %s", path)
	}

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	fw, err := w.CreateFormFile("file", filepath.Base(path))
	if err != nil {
		return "", err
	}
	fw.Write(data)
	w.Close()

	url := fmt.Sprintf("%s/files?key=%s", baseURL, apiKey)
	req, _ := http.NewRequest("POST", url, &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())
	req.Header.Set("X-Goog-Upload-Protocol", "multipart")
	req.Header.Set("X-Goog-Upload-Command", "upload, finalize")
	req.Header.Set("X-Goog-Upload-Header-Content-Type", mimeType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("file upload %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		File struct {
			URI string `json:"uri"`
		} `json:"file"`
	}
	if err := json.Unmarshal(body, &result); err != nil || result.File.URI == "" {
		return "", fmt.Errorf("parse upload response: %v\nbody: %s", err, string(body))
	}
	return result.File.URI, nil
}

func saveImage(mimeType, b64data, outPrefix string, runIdx, total, fileIndex int, jsonMode bool) (string, bool) {
	ext := mimeToExt[mimeType]
	if ext == "" {
		ext = "png"
	}
	suffix := ""
	if total > 1 {
		suffix = fmt.Sprintf("-%d-%d", runIdx, fileIndex)
	} else if fileIndex > 0 {
		suffix = fmt.Sprintf("-%d", fileIndex)
	}
	outPath := fmt.Sprintf("%s%s.%s", outPrefix, suffix, ext)

	imgData, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "decode error: %v\n", err)
		return "", false
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		fmt.Fprintf(os.Stderr, "mkdir error: %v\n", err)
		return "", false
	}
	if err := os.WriteFile(outPath, imgData, 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		return "", false
	}
	if !jsonMode {
		fmt.Fprintf(os.Stderr, "saved: %s\n", outPath)
	}
	return outPath, true
}

func generateInteractions(apiKey, model, outPrefix string, imagePaths []string, prompt string, imgCfg *imageConfig, thinkCfg *thinkingConfig, runIdx, total int, jsonMode bool) ([]string, error) {
	if total > 1 && !jsonMode {
		fmt.Fprintf(os.Stderr, "[%d/%d] generating...\n", runIdx+1, total)
	}

	var parts []inputPart

	for _, imgPath := range imagePaths {
		mimeType, ok := extToMime[strings.ToLower(filepath.Ext(imgPath))]
		if !ok {
			return nil, fmt.Errorf("unsupported image type: %s", imgPath)
		}

		fmt.Fprintf(os.Stderr, "uploading: %s\n", imgPath)
		uri, err := uploadFile(apiKey, imgPath)
		if err != nil {
			fmt.Fprintf(os.Stderr, "upload failed (%v), using inline data\n", err)
			data, rerr := os.ReadFile(imgPath)
			if rerr != nil {
				return nil, fmt.Errorf("cannot read image: %s", imgPath)
			}
			parts = append(parts, inputPart{
				Type:     "image",
				MimeType: mimeType,
				Data:     base64.StdEncoding.EncodeToString(data),
			})
		} else {
			parts = append(parts, inputPart{Type: "image", FileURI: uri, MimeType: mimeType})
		}
	}

	parts = append(parts, inputPart{Type: "text", Text: prompt})

	var genCfg *generationConfig
	if imgCfg != nil || thinkCfg != nil {
		genCfg = &generationConfig{ImageConfig: imgCfg, ThinkingConfig: thinkCfg}
	}

	var input interface{}
	if len(parts) == 1 && parts[0].Type == "text" {
		input = parts[0].Text
	} else {
		input = parts
	}

	ireq := interactionRequest{
		Model:              model,
		Input:              input,
		ResponseModalities: []string{"image", "text"},
		GenerationConfig:   genCfg,
	}

	body, _ := json.Marshal(ireq)

	url := fmt.Sprintf("%s/interactions?key=%s", baseURL, apiKey)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(respBody))
	}

	var iresp interactionResponse
	if err := json.Unmarshal(respBody, &iresp); err != nil {
		return nil, fmt.Errorf("parse response: %v\nbody: %s", err, string(respBody))
	}

	var files []string
	fileIndex := 0
	for _, out := range iresp.Outputs {
		outType := strings.ToLower(out.Type)
		switch {
		case outType == "image" || (out.Data != "" && out.MimeType != "" && strings.HasPrefix(out.MimeType, "image/")):
			if path, ok := saveImage(out.MimeType, out.Data, outPrefix, runIdx, total, fileIndex, jsonMode); ok {
				files = append(files, path)
				fileIndex++
			}
		case outType == "text" || out.Text != "":
			if strings.TrimSpace(out.Text) != "" {
				fmt.Fprintf(os.Stderr, "model: %s\n", strings.TrimSpace(out.Text))
			}
		}
	}
	return files, nil
}

func generateLegacy(apiKey, model, outPrefix string, imagePaths []string, prompt string, cfg legacyGenerationConfig, runIdx, total int, jsonMode bool) ([]string, error) {
	if total > 1 && !jsonMode {
		fmt.Fprintf(os.Stderr, "[%d/%d] generating...\n", runIdx+1, total)
	}

	var parts []legacyPart
	for _, imgPath := range imagePaths {
		mimeType, ok := extToMime[strings.ToLower(filepath.Ext(imgPath))]
		if !ok {
			return nil, fmt.Errorf("unsupported image type: %s", imgPath)
		}
		data, err := os.ReadFile(imgPath)
		if err != nil {
			return nil, fmt.Errorf("cannot read image: %s", imgPath)
		}
		parts = append(parts, legacyPart{InlineData: &inlineData{MimeType: mimeType, Data: base64.StdEncoding.EncodeToString(data)}})
	}
	parts = append(parts, legacyPart{Text: prompt})

	req := legacyRequest{
		Contents: []struct {
			Role  string       `json:"role"`
			Parts []legacyPart `json:"parts"`
		}{{Role: "user", Parts: parts}},
		GenerationConfig: cfg,
	}

	body, _ := json.Marshal(req)
	url := fmt.Sprintf("%s/models/%s:streamGenerateContent?alt=sse&key=%s", baseURL, model, apiKey)

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(errBody))
	}

	var files []string
	fileIndex := 0
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 10*1024*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		payload := strings.TrimPrefix(line, "data: ")
		if payload == "[DONE]" {
			break
		}

		var chunk streamChunk
		if json.Unmarshal([]byte(payload), &chunk) != nil {
			continue
		}

		for _, candidate := range chunk.Candidates {
			for _, p := range candidate.Content.Parts {
				if p.InlineData != nil {
					if path, ok := saveImage(p.InlineData.MimeType, p.InlineData.Data, outPrefix, runIdx, total, fileIndex, jsonMode); ok {
						files = append(files, path)
						fileIndex++
					}
				} else if strings.TrimSpace(p.Text) != "" {
					fmt.Fprintf(os.Stderr, "model: %s\n", strings.TrimSpace(p.Text))
				}
			}
		}
	}
	return files, scanner.Err()
}

// valueFlags take the next arg as their value (not booleans)
var valueFlags = map[string]bool{"i": true, "o": true, "n": true, "m": true, "a": true, "s": true, "thinking": true}

func reorderArgs(args []string) []string {
	var flags, positional []string
	i := 1
	for i < len(args) {
		arg := args[i]
		if arg == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if !strings.HasPrefix(arg, "-") {
			positional = append(positional, arg)
			i++
			continue
		}
		name := strings.TrimLeft(arg, "-")
		if idx := strings.IndexByte(name, '='); idx >= 0 {
			name = name[:idx]
		}
		flags = append(flags, arg)
		if valueFlags[name] && !strings.Contains(arg, "=") && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
		i++
	}
	return append([]string{args[0]}, append(flags, positional...)...)
}

func isLegacyModel(model string) bool {
	return strings.Contains(model, "2.5") || strings.Contains(model, "2.0") || strings.Contains(model, "2-")
}

func main() {
	var (
		images       stringSlice
		outPrefix    string
		count        int
		model        string
		aspect       string
		size         string
		thinking     string
		showThoughts bool
		useStdin     bool
		jsonMode     bool
	)

	flag.Var(&images, "i", "Reference image (repeatable)")
	flag.StringVar(&outPrefix, "o", fmt.Sprintf("out/gen-%d", time.Now().UnixMilli()), "Output path prefix")
	flag.IntVar(&count, "n", 1, "Number of variants")
	flag.StringVar(&model, "m", "gemini-3.1-flash-image-preview", "Model ID or alias: flash, pro, flash25")
	flag.StringVar(&aspect, "a", "", "Aspect ratio: 1:1, 16:9, 9:16, 21:9, etc.")
	flag.StringVar(&size, "s", "1K", "Image size: 512, 1K, 2K, 4K")
	flag.StringVar(&thinking, "thinking", "", "Thinking level: minimal or high (flash only)")
	flag.BoolVar(&showThoughts, "show-thoughts", false, "Include model thoughts in stderr")
	flag.BoolVar(&useStdin, "stdin", false, "Read prompt from stdin")
	flag.BoolVar(&jsonMode, "json", false, "Output JSON {files: [...]}")

	os.Args = reorderArgs(os.Args)
	flag.Parse()

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		fmt.Fprintln(os.Stderr, "GEMINI_API_KEY not set.")
		os.Exit(1)
	}

	var prompt string
	if useStdin {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			fmt.Fprintf(os.Stderr, "stdin error: %v\n", err)
			os.Exit(1)
		}
		prompt = strings.TrimSpace(string(data))
	} else {
		args := flag.Args()
		if len(args) == 0 {
			fmt.Fprintln(os.Stderr, "Prompt required. Pass as argument or use --stdin.")
			os.Exit(1)
		}
		prompt = strings.Join(args, " ")
	}

	if alias, ok := modelAliases[model]; ok {
		model = alias
	}

	type result struct {
		files []string
		err   error
	}

	ch := make(chan result, count)

	if isLegacyModel(model) {
		var legacyImgCfg *legacyImageConfig
		if aspect != "" || size != "" {
			legacyImgCfg = &legacyImageConfig{AspectRatio: aspect, ImageSize: size}
		}
		var thinkCfg *thinkingConfig
		if thinking != "" || showThoughts {
			thinkCfg = &thinkingConfig{ThinkingLevel: thinking, IncludeThoughts: showThoughts}
		}
		cfg := legacyGenerationConfig{
			ResponseModalities:    []string{"IMAGE", "TEXT"},
			ImageGenerationConfig: legacyImgCfg,
			ThinkingConfig:        thinkCfg,
		}
		for i := 0; i < count; i++ {
			go func(idx int) {
				files, err := generateLegacy(apiKey, model, outPrefix, []string(images), prompt, cfg, idx, count, jsonMode)
				ch <- result{files, err}
			}(i)
		}
	} else {
		var imgCfg *imageConfig
		if aspect != "" || size != "" {
			imgCfg = &imageConfig{AspectRatio: aspect, ImageSize: strings.ToLower(size)}
		}
		var thinkCfg *thinkingConfig
		if thinking != "" || showThoughts {
			thinkCfg = &thinkingConfig{ThinkingLevel: thinking, IncludeThoughts: showThoughts}
		}
		for i := 0; i < count; i++ {
			go func(idx int) {
				files, err := generateInteractions(apiKey, model, outPrefix, []string(images), prompt, imgCfg, thinkCfg, idx, count, jsonMode)
				ch <- result{files, err}
			}(i)
		}
	}

	var allFiles []string
	for i := 0; i < count; i++ {
		r := <-ch
		if r.err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", r.err)
			os.Exit(1)
		}
		allFiles = append(allFiles, r.files...)
	}

	if jsonMode {
		out, _ := json.Marshal(map[string][]string{"files": allFiles})
		fmt.Println(string(out))
	} else {
		for _, f := range allFiles {
			fmt.Println(f)
		}
	}
}
