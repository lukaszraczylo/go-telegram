package api

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/lukaszraczylo/go-telegram/client"
)

// DownloadFile fetches the contents of a Telegram-hosted file given a
// previously-uploaded file_id. It calls GetFile to resolve the file's
// download path, then issues an HTTP GET to the file CDN endpoint.
//
// The returned io.ReadCloser must be closed by the caller. The size of
// the file is reported via *File.FileSize when known.
//
// For files larger than 20 MB, Telegram requires a self-hosted Bot API
// server (default api.telegram.org has a 20 MB limit on getFile).
func DownloadFile(ctx context.Context, b *client.Bot, fileID string) (io.ReadCloser, *File, error) {
	f, err := GetFile(ctx, b, &GetFileParams{FileID: fileID})
	if err != nil {
		return nil, nil, fmt.Errorf("getFile: %w", err)
	}
	if f == nil || f.FilePath == "" {
		return nil, f, fmt.Errorf("telegram: file %q has no download path", fileID)
	}
	rc, err := DownloadFileByPath(ctx, b, f.FilePath)
	if err != nil {
		return nil, f, err
	}
	return rc, f, nil
}

// DownloadFileByPath fetches a file by its file_path (typically obtained
// from a prior File response). Useful when the caller already has a
// *File and wants to skip the GetFile round-trip.
func DownloadFileByPath(ctx context.Context, b *client.Bot, filePath string) (io.ReadCloser, error) {
	url := fmt.Sprintf("%s/file/bot%s/%s", b.BaseURL(), b.Token(), filePath)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := b.HTTP().Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, fmt.Errorf("download: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		_ = resp.Body.Close()
		return nil, fmt.Errorf("download: HTTP %d", resp.StatusCode)
	}
	return resp.Body, nil
}
