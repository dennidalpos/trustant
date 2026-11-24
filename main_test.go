// Copyright 2025-2026 Nuvolaris Inc
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <https://www.gnu.org/licenses/>.

package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNoStoreHTMLPreventsStaleConfigurationFlow(t *testing.T) {
	handler := noStoreHTML(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for _, path := range []string{"/", "/applist.html", "/configure.html?reselect=1"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if got := response.Header().Get("Cache-Control"); got != "no-store" {
			t.Fatalf("%s Cache-Control = %q, want no-store", path, got)
		}
	}

	// Only documents carry inline boot logic capable of redirecting between
	// configuration pages, so they are never stored. Stylesheets and scripts may
	// be cached but must be revalidated, otherwise a stale CSS or JS file is
	// paired with an upgraded page.
	for _, path := range []string{"/trustant-ui.css", "/tailwind.js", "/js/shared.js"} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if got := response.Header().Get("Cache-Control"); got != "no-cache" {
			t.Fatalf("%s Cache-Control = %q, want no-cache", path, got)
		}
	}

	// Other assets keep the file server default.
	request := httptest.NewRequest(http.MethodGet, "/trustant-logo.svg", nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if got := response.Header().Get("Cache-Control"); got != "" {
		t.Fatalf("SVG Cache-Control = %q, want the file server default", got)
	}
}
