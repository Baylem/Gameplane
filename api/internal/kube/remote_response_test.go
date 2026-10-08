package kube

import (
	"bytes"
	"compress/gzip"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

type responseRoundTripper func(*http.Request) (*http.Response, error)

func (f responseRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }

type countedResponse struct {
	io.Reader
	read   int
	closed bool
}

func (r *countedResponse) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	r.read += n
	return n, err
}

func (r *countedResponse) Close() error { r.closed = true; return nil }

func TestRemoteResponseBoundsVersionBeforeBuffering(t *testing.T) {
	for _, compressed := range []bool{false, true} {
		t.Run(map[bool]string{false: "chunked", true: "gzip"}[compressed], func(t *testing.T) {
			payload := []byte(strings.Repeat("x", 128*1024))
			header := http.Header{}
			if compressed {
				var encoded bytes.Buffer
				writer := gzip.NewWriter(&encoded)
				if _, err := writer.Write(payload); err != nil {
					t.Fatal(err)
				}
				if err := writer.Close(); err != nil {
					t.Fatal(err)
				}
				payload = encoded.Bytes()
				header.Set("Content-Encoding", "gzip")
			}
			body := &countedResponse{Reader: bytes.NewReader(payload)}
			transport := boundedRemoteTransport(responseRoundTripper(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: header, Body: body, ContentLength: -1}, nil
			}))
			req, err := http.NewRequest(http.MethodGet, "https://cluster.example/version", nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := transport.RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			data, err := io.ReadAll(resp.Body)
			if !errors.Is(err, errRemoteResponseTooLarge) || len(data) > 64*1024 {
				t.Fatalf("oversize version was buffered: bytes=%d error=%v", len(data), err)
			}
			if err := resp.Body.Close(); err != nil {
				t.Fatal(err)
			}
			if !body.closed {
				t.Fatal("remote response was not closed")
			}
			if !compressed && body.read > 64*1024+1 {
				t.Fatalf("read %d bytes before rejecting", body.read)
			}
		})
	}
}

func TestRemoteResponseWatchBoundsIndividualEvents(t *testing.T) {
	for _, tc := range []struct {
		name, payload string
		wantError     bool
	}{
		{"multiple bounded frames", strings.Repeat(`{"type":"ADDED","object":{"padding":"`+strings.Repeat("x", 600*1024)+`"}}`+"\n", 3), false},
		{"oversize frame", `{"type":"ADDED","object":{"padding":"` + strings.Repeat("x", 2*1024*1024) + `"}}`, true},
		{"escaped braces", `{"object":{"padding":"` + strings.Repeat(`\"{}\\`, 1024) + `"}}` + "\n", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body := &countedResponse{Reader: strings.NewReader(tc.payload)}
			transport := boundedRemoteTransport(responseRoundTripper(func(*http.Request) (*http.Response, error) {
				return &http.Response{StatusCode: 200, Header: http.Header{}, Body: body}, nil
			}))
			req, err := http.NewRequest(http.MethodGet, "https://cluster.example/apis/gameplane.local/v1alpha1/gameservers?watch=true", nil)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := transport.RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			_, err = io.Copy(io.Discard, resp.Body)
			if errors.Is(err, errRemoteResponseTooLarge) != tc.wantError {
				t.Fatalf("watch result: %v", err)
			}
			if tc.wantError && body.read > 1024*1024+1 {
				t.Fatalf("oversize event read %d bytes", body.read)
			}
			if err := resp.Body.Close(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestRemoteResponseLeavesUserLogStreamsUnbounded(t *testing.T) {
	payload := strings.Repeat("log line\n", 1024*1024)
	transport := boundedRemoteTransport(responseRoundTripper(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: 200, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(payload))}, nil
	}))
	req, err := http.NewRequest(http.MethodGet, "https://cluster.example/api/v1/namespaces/games/pods/game/log?follow=true", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := transport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	n, err := io.Copy(io.Discard, resp.Body)
	if err != nil || n != int64(len(payload)) {
		t.Fatalf("log stream was truncated: bytes=%d error=%v", n, err)
	}
	if err := resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
}
