// Copyright 2023 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package web

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"forgejo.org/modules/setting"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRouteMock(t *testing.T) {
	setting.IsInTesting = true

	r := NewRoute()
	middleware1 := func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("X-Test-Middleware1", "m1")
	}
	middleware2 := func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("X-Test-Middleware2", "m2")
	}
	handler := func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("X-Test-Handler", "h")
	}
	r.Get("/foo", middleware1, RouteMockPoint("mock-point"), middleware2, handler)

	// normal request
	recorder := httptest.NewRecorder()
	req, err := http.NewRequest("GET", "http://localhost:8000/foo", nil)
	require.NoError(t, err)
	r.ServeHTTP(recorder, req)
	assert.Len(t, recorder.Header(), 3)
	assert.Equal(t, "m1", recorder.Header().Get("X-Test-Middleware1"))
	assert.Equal(t, "m2", recorder.Header().Get("X-Test-Middleware2"))
	assert.Equal(t, "h", recorder.Header().Get("X-Test-Handler"))
	RouteMockReset()

	// mock at "mock-point"
	RouteMock("mock-point", func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("X-Test-MockPoint", "a")
		resp.WriteHeader(http.StatusOK)
	})
	recorder = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "http://localhost:8000/foo", nil)
	require.NoError(t, err)
	r.ServeHTTP(recorder, req)
	assert.Len(t, recorder.Header(), 2)
	assert.Equal(t, "m1", recorder.Header().Get("X-Test-Middleware1"))
	assert.Equal(t, "a", recorder.Header().Get("X-Test-MockPoint"))
	RouteMockReset()

	// mock at MockAfterMiddlewares
	RouteMock(MockAfterMiddlewares, func(resp http.ResponseWriter, req *http.Request) {
		resp.Header().Set("X-Test-MockPoint", "b")
		resp.WriteHeader(http.StatusOK)
	})
	recorder = httptest.NewRecorder()
	req, err = http.NewRequest("GET", "http://localhost:8000/foo", nil)
	require.NoError(t, err)
	r.ServeHTTP(recorder, req)
	assert.Len(t, recorder.Header(), 3)
	assert.Equal(t, "m1", recorder.Header().Get("X-Test-Middleware1"))
	assert.Equal(t, "m2", recorder.Header().Get("X-Test-Middleware2"))
	assert.Equal(t, "b", recorder.Header().Get("X-Test-MockPoint"))
	RouteMockReset()
}
