package api

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lukaszraczylo/go-telegram/client"
	"github.com/stretchr/testify/require"
)

func TestDownloadFile_HappyPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/getFile"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"ok":true,"result":{"file_id":"abc","file_unique_id":"u","file_size":11,"file_path":"documents/hello.txt"}}`))
		case strings.HasPrefix(r.URL.Path, "/file/bot"):
			_, _ = w.Write([]byte("hello world"))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	bot := client.New("123:abc", client.WithBaseURL(srv.URL))
	rc, file, err := DownloadFile(context.Background(), bot, "abc")
	require.NoError(t, err)
	defer rc.Close()
	require.Equal(t, "documents/hello.txt", file.FilePath)
	body, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, "hello world", string(body))
}

func TestDownloadFile_GetFileFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":false,"error_code":400,"description":"Bad Request: invalid file_id"}`))
	}))
	t.Cleanup(srv.Close)

	bot := client.New("t", client.WithBaseURL(srv.URL))
	_, _, err := DownloadFile(context.Background(), bot, "bad")
	require.Error(t, err)
	require.Contains(t, err.Error(), "getFile")
}

func TestDownloadFile_NoFilePath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// result without file_path
		_, _ = w.Write([]byte(`{"ok":true,"result":{"file_id":"abc","file_unique_id":"u"}}`))
	}))
	t.Cleanup(srv.Close)

	bot := client.New("t", client.WithBaseURL(srv.URL))
	_, _, err := DownloadFile(context.Background(), bot, "abc")
	require.Error(t, err)
	require.Contains(t, err.Error(), "no download path")
}

func TestDownloadFileByPath_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/file/bot") {
			w.WriteHeader(http.StatusForbidden)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(srv.Close)

	bot := client.New("t", client.WithBaseURL(srv.URL))
	_, err := DownloadFileByPath(context.Background(), bot, "secret/file")
	require.Error(t, err)
	require.Contains(t, err.Error(), "403")
}
