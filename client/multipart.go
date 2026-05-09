package client

import (
	"context"
	"github.com/goccy/go-json"
	"io"
	"mime/multipart"
	"net/http"
)

// multipartRequest is implemented by request structs that may carry an
// InputFile. The codegen emits this interface for any method whose IR
// MethodDecl.HasFiles is true.
//
// HasFile returns true if at least one file field is set; if false, the
// request is sent as plain JSON via the regular Call path.
//
// MultipartFiles returns one entry per file field that should be uploaded.
// The accompanying scalar/object fields are returned by MultipartFields.
type multipartRequest interface {
	HasFile() bool
	MultipartFiles() []MultipartFile
	MultipartFields() map[string]string
}

// MultipartFile describes a single file part in a multipart upload.
type MultipartFile struct {
	FieldName string
	Filename  string
	Reader    io.Reader
}

// callMultipart performs a multipart/form-data POST. It is invoked by Call
// when the request implements multipartRequest and HasFile() is true.
func callMultipart[Resp any](ctx context.Context, b *Bot, method string, mp multipartRequest) (Resp, error) {
	var zero Resp

	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	// Stream-write the multipart body in a goroutine so we don't buffer
	// large files in memory.
	go func() {
		defer func() { _ = pw.Close() }()
		defer func() { _ = mw.Close() }()
		for k, v := range mp.MultipartFields() {
			if err := mw.WriteField(k, v); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
		}
		for _, f := range mp.MultipartFiles() {
			part, err := mw.CreateFormFile(f.FieldName, f.Filename)
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			if _, err := io.Copy(part, f.Reader); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
		}
	}()

	url := b.base + "/bot" + b.token + "/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
	if err != nil {
		_ = pr.CloseWithError(err)
		return zero, &NetworkError{Err: err}
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	resp, err := b.http.Do(req)
	if err != nil {
		_ = pr.CloseWithError(err)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return zero, ctxErr
		}
		return zero, &NetworkError{Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = pr.CloseWithError(err)
		return zero, &NetworkError{Err: err}
	}
	return decodeResult[Resp](b.codec, raw)
}

// callMultipartRaw is callMultipart's sibling that returns the raw result
// JSON instead of decoding into a typed value. Used by generated method
// wrappers whose return type is a sealed-interface union.
func callMultipartRaw(ctx context.Context, b *Bot, method string, mp multipartRequest) (json.RawMessage, error) {
	pr, pw := io.Pipe()
	mw := multipart.NewWriter(pw)

	go func() {
		defer func() { _ = pw.Close() }()
		defer func() { _ = mw.Close() }()
		for k, v := range mp.MultipartFields() {
			if err := mw.WriteField(k, v); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
		}
		for _, f := range mp.MultipartFiles() {
			part, err := mw.CreateFormFile(f.FieldName, f.Filename)
			if err != nil {
				_ = pw.CloseWithError(err)
				return
			}
			if _, err := io.Copy(part, f.Reader); err != nil {
				_ = pw.CloseWithError(err)
				return
			}
		}
	}()

	url := b.base + "/bot" + b.token + "/" + method
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, pr)
	if err != nil {
		_ = pr.CloseWithError(err)
		return nil, &NetworkError{Err: err}
	}
	req.Header.Set("Content-Type", mw.FormDataContentType())
	req.Header.Set("Accept", "application/json")

	resp, err := b.http.Do(req)
	if err != nil {
		_ = pr.CloseWithError(err)
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, &NetworkError{Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		_ = pr.CloseWithError(err)
		return nil, &NetworkError{Err: err}
	}
	return decodeResultRaw(b.codec, raw)
}
