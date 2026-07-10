package handlers

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/max-marek-projects/loyalty-system/internal/auth"
	"github.com/stretchr/testify/require"
)

const secretKey string = "12345"
const testUserID int64 = 12345

// test single request
func testRequest(t *testing.T, ts *httptest.Server, method, path, body string, authorize bool) (*http.Response, string) {
	t.Helper()

	var reader io.Reader
	if body != "" {
		reader = bytes.NewBufferString(body)
	}

	req, err := http.NewRequest(method, ts.URL+path, reader)
	require.NoError(t, err)

	if authorize {
		w := httptest.NewRecorder()
		err := auth.SetUserCookie(w, testUserID, secretKey)
		require.NoError(t, err)
		cookie := w.Result().Cookies()[0]
		req.AddCookie(cookie)
	}

	client := ts.Client()
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return http.ErrUseLastResponse
	}
	resp, err := client.Do(req)
	require.NoError(t, err)

	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	return resp, string(respBody)
}
