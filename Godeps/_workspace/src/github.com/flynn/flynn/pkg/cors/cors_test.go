// Copyright 2014 Google Inc. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cors

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func serveHTTP(w http.ResponseWriter, req *http.Request, opts *Options) {
	opts.Handler(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})).ServeHTTP(w, req)
}

func Test_AllowAll(t *testing.T) {
	recorder := httptest.NewRecorder()
	origin := "https://bar.foo.com"
	r, _ := http.NewRequest("PUT", "foo", nil)
	r.Header.Add("Origin", origin)
	serveHTTP(recorder, r, &Options{
		AllowAllOrigins: true,
	})

	headerValue := recorder.HeaderMap.Get(headerAllowOrigin)
	if headerValue != origin {
		t.Errorf("Allow-Origin header should be %v, found %v", origin, headerValue)
	}
}

func Test_AllowRegexMatch(t *testing.T) {
	recorder := httptest.NewRecorder()
	origin := "https://bar.foo.com"
	r, _ := http.NewRequest("PUT", "foo", nil)
	r.Header.Add("Origin", origin)
	serveHTTP(recorder, r, &Options{
		AllowOrigins: []string{"https://aaa.com", "https://*.foo.com"},
	})

	headerValue := recorder.HeaderMap.Get(headerAllowOrigin)
	if headerValue != origin {
		t.Errorf("Allow-Origin header should be %v, found %v", origin, headerValue)
	}
}

func Test_AllowRegexNoMatch(t *testing.T) {
	recorder := httptest.NewRecorder()
	origin := "https://ww.foo.com.evil.com"
	r, _ := http.NewRequest("PUT", "foo", nil)
	r.Header.Add("Origin", origin)
	serveHTTP(recorder, r, &Options{
		AllowOrigins: []string{"https://*.foo.com"},
	})

	headerValue := recorder.HeaderMap.Get(headerAllowOrigin)
	if headerValue != "" {
		t.Errorf("Allow-Origin header should not exist, found %v", headerValue)
	}
}

func Test_OtherHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()
	r, _ := http.NewRequest("PUT", "foo", nil)
	origin := "https://ww.foo.com.evil.com"
	r.Header.Add("Origin", origin)
	serveHTTP(recorder, r, &Options{
		AllowAllOrigins:  true,
		AllowCredentials: true,
		AllowMethods:     []string{"PATCH", "GET"},
		AllowHeaders:     []string{"Origin", "X-whatever"},
		ExposeHeaders:    []string{"Content-Length", "Hello"},
		MaxAge:           5 * time.Minute,
	})

	credentialsVal := recorder.HeaderMap.Get(headerAllowCredentials)
	methodsVal := recorder.HeaderMap.Get(headerAllowMethods)
	headersVal := recorder.HeaderMap.Get(headerAllowHeaders)
	exposedHeadersVal := recorder.HeaderMap.Get(headerExposeHeaders)
	maxAgeVal := recorder.HeaderMap.Get(headerMaxAge)

	if credentialsVal != "true" {
		t.Errorf("Allow-Credentials is expected to be true, found %v", credentialsVal)
	}

	if methodsVal != "PATCH,GET" {
		t.Errorf("Allow-Methods is expected to be PATCH,GET; found %v", methodsVal)
	}

	if headersVal != "Origin,X-whatever" {
		t.Errorf("Allow-Headers is expected to be Origin,X-whatever; found %v", headersVal)
	}

	if exposedHeadersVal != "Content-Length,Hello" {
		t.Errorf("Expose-Headers are expected to be Content-Length,Hello. Found %v", exposedHeadersVal)
	}

	if maxAgeVal != "300" {
		t.Errorf("Max-Age is expected to be 300, found %v", maxAgeVal)
	}
}

func Test_DefaultAllowHeaders(t *testing.T) {
	recorder := httptest.NewRecorder()
	r, _ := http.NewRequest("PUT", "foo", nil)
	origin := "https://ww.foo.com.evil.com"
	r.Header.Add("Origin", origin)
	serveHTTP(recorder, r, &Options{
		AllowAllOrigins: true,
	})

	headersVal := recorder.HeaderMap.Get(headerAllowHeaders)
	if headersVal != "Origin,Accept,Content-Type,Authorization" {
		t.Errorf("Allow-Headers is expected to be Origin,Accept,Content-Type,Authorization; found %v", headersVal)
	}
}

func Test_Preflight(t *testing.T) {
	recorder := httptest.NewRecorder()
	r, _ := http.NewRequest("OPTIONS", "foo", nil)
	r.Header.Add(headerRequestMethod, "PUT")
	r.Header.Add(headerRequestHeaders, "X-whatever, x-casesensitive")
	origin := "https://bar.foo.com"
	r.Header.Add("Origin", origin)
	serveHTTP(recorder, r, &Options{
		AllowAllOrigins: true,
		AllowMethods:    []string{"PUT", "PATCH"},
		AllowHeaders:    []string{"Origin", "X-whatever", "X-CaseSensitive"},
	})

	methodsVal := recorder.HeaderMap.Get(headerAllowMethods)
	headersVal := recorder.HeaderMap.Get(headerAllowHeaders)
	originVal := recorder.HeaderMap.Get(headerAllowOrigin)

	if methodsVal != "PUT,PATCH" {
		t.Errorf("Allow-Methods is expected to be PUT,PATCH, found %v", methodsVal)
	}

	if !strings.Contains(headersVal, "X-whatever") {
		t.Errorf("Allow-Headers is expected to contain X-whatever, found %v", headersVal)
	}

	if !strings.Contains(headersVal, "X-CaseSensitive") {
		t.Errorf("Allow-Headers is expected to contain x-casesensitive, found %v", headersVal)
	}

	if originVal != origin {
		t.Errorf("Allow-Origin is expected to be %v, found %v", origin, originVal)
	}
}

func Benchmark_WithoutCORS(b *testing.B) {
	recorder := httptest.NewRecorder()
	b.ResetTimer()
	for i := 0; i < 100; i++ {
		http.NewRequest("PUT", "foo", nil)
		recorder.WriteHeader(200)
	}
}

func Benchmark_WithCORS(b *testing.B) {
	recorder := httptest.NewRecorder()
	b.ResetTimer()
	for i := 0; i < 100; i++ {
		r, _ := http.NewRequest("PUT", "foo", nil)
		serveHTTP(recorder, r, &Options{
			AllowAllOrigins:  true,
			AllowCredentials: true,
			AllowMethods:     []string{"PATCH", "GET"},
			AllowHeaders:     []string{"Origin", "X-whatever"},
			MaxAge:           5 * time.Minute,
		})
	}
}
