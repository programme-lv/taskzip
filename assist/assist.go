package assist

import (
	"bytes"
	"context"
	"errors"
	"mime"
	"os"
	"path/filepath"
	"strings"

	openai "github.com/openai/openai-go/v2"
	"github.com/openai/openai-go/v2/option"
	"github.com/openai/openai-go/v2/responses"
	"github.com/openai/openai-go/v2/shared"
	"github.com/programme-lv/taskzip/common/etrace"
)

type File struct {
	Content []byte
	Fname   string
}

var ErrOpenAIAPIKeyNotSet = etrace.NewError("OPENAI_API_KEY not set")

func AskChatGpt(prompt string, attached []File) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", etrace.Trace(ErrOpenAIAPIKeyNotSet)
	}
	ctx := context.Background()
	client := openai.NewClient(option.WithAPIKey(apiKey))

	// 1) Upload files (purpose=assistants)
	fileIDs := make([]string, 0, len(attached))
	for _, f := range attached {
		mimeType := mime.TypeByExtension(filepath.Ext(f.Fname))
		fo, err := client.Files.New(ctx, openai.FileNewParams{
			File: openai.File(
				bytes.NewReader(f.Content),
				safeFilename(f.Fname), // .typ -> .txt so it can be indexed
				mimeType,
			),
			Purpose:      "assistants",
			ExpiresAfter: openai.FileNewParamsExpiresAfter{},
		})
		if err != nil {
			return "", etrace.Trace(etrace.Wrap("create file", err))
		}
		fileIDs = append(fileIDs, fo.ID)
	}

	// 2) Create a vector store and attach those files
	vs, err := client.VectorStores.New(ctx, openai.VectorStoreNewParams{
		FileIDs: fileIDs,
	})
	if err != nil {
		return "", etrace.Trace(etrace.Wrap("create vector store", err))
	}

	// 3) Responses API + file_search tool pointing at our vector store
	params := responses.ResponseNewParams{
		Model: openai.ChatModelGPT5Nano, // pick your model
		Input: responses.ResponseNewParamsInputUnion{
			OfString: openai.String(prompt),
		},
		MaxOutputTokens: openai.Int(10000),
		Tools: []responses.ToolUnionParam{
			{OfFileSearch: &responses.FileSearchToolParam{
				VectorStoreIDs: []string{vs.ID},
			}},
		},
	}

	resp, err := client.Responses.New(
		ctx,
		params,
	)
	if err != nil {
		return "", etrace.Trace(etrace.Wrap("responses.new", err))
	}

	md := strings.TrimSpace(resp.OutputText()) // helper to extract just the text :contentReference[oaicite:2]{index=2}
	if md == "" {
		return "", etrace.Trace(errors.New("empty output_text from model"))
	}

	// 4) Delete files and vector store
	for _, fileID := range fileIDs {
		_, err := client.Files.Delete(ctx, fileID)
		if err != nil {
			return "", etrace.Trace(etrace.Wrap("delete file", err))
		}
	}
	_, err = client.VectorStores.Delete(ctx, vs.ID)
	if err != nil {
		return "", etrace.Trace(etrace.Wrap("delete vector store", err))
	}

	return md, nil
}

// AskChatGptSimple uses the Chat Completions SDK without attachments.
// Returns trimmed assistant message content.
func AskChatGptSimple(prompt string) (string, error) {
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return "", etrace.Trace(ErrOpenAIAPIKeyNotSet)
	}
	ctx := context.Background()
	client := openai.NewClient(option.WithAPIKey(apiKey))

	params := openai.ChatCompletionNewParams{
		Model: shared.ChatModelGPT4_1,
		Messages: []openai.ChatCompletionMessageParamUnion{{
			OfUser: &openai.ChatCompletionUserMessageParam{
				Content: openai.ChatCompletionUserMessageParamContentUnion{
					OfString: openai.String(prompt),
				},
			},
		}},
	}

	res, err := client.Chat.Completions.New(ctx, params)
	if err != nil {
		return "", etrace.Trace(etrace.Wrap("chat.completions.new", err))
	}
	if len(res.Choices) == 0 {
		return "", etrace.Trace(errors.New("empty completion"))
	}
	if strings.TrimSpace(res.Choices[0].Message.Content) != "" {
		return strings.TrimSpace(res.Choices[0].Message.Content), nil
	}
	return "", etrace.Trace(errors.New("no text content in completion"))
}

func safeFilename(fname string) string {
	fname = filepath.Base(fname)
	ext := filepath.Ext(fname)
	extNoDot := strings.TrimPrefix(ext, ".")
	// Keep supported types (e.g. .pdf) as-is; rewrite unknown (e.g. .typ) to .txt
	if extIsSupported(extNoDot) {
		if strings.EqualFold(ext, ".typ") {
			base := strings.TrimSuffix(fname, ext)
			return base + ".txt"
		}
		return fname
	}
	base := strings.TrimSuffix(fname, ext)
	return base + ".txt"
}

func extIsSupported(ext string) bool {
	// ext should NOT include the dot
	supported := map[string]struct{}{
		"c": {}, "cpp": {}, "css": {}, "csv": {}, "doc": {}, "docx": {}, "gif": {}, "go": {}, "html": {},
		"java": {}, "jpeg": {}, "jpg": {}, "js": {}, "json": {}, "md": {}, "pdf": {}, "php": {}, "pkl": {},
		"png": {}, "pptx": {}, "py": {}, "rb": {}, "tar": {}, "tex": {}, "ts": {}, "txt": {}, "webp": {},
		"xlsx": {}, "xml": {}, "zip": {},
	}
	_, ok := supported[strings.ToLower(ext)]
	return ok
}
