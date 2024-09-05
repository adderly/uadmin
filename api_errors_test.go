package uadmin

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogError(t *testing.T) {
	request := &http.Request{
		Method:     "GET",
		RequestURI: "/test/data",
	}
	msg := "message"
	userErr := fmt.Errorf("original error")
	expectedMessage := fmt.Sprintf("failed [%s] to [%s], msg: %s", "GET", "/test/data", msg)

	stdOut := os.Stdout
	defer func() {
		os.Stdout = stdOut
	}()
	r, w, err := os.Pipe()
	require.Nil(t, err)
	os.Stdout = w

	logError(request, msg, userErr)

	err = w.Close()
	require.Nil(t, err)
	res, err := io.ReadAll(r)
	require.Nil(t, err)

	assert.Contains(t, string(res), expectedMessage)
	assert.Contains(t, string(res), userErr.Error())
}

func TestRespondAndLogError(t *testing.T) {
	type args struct {
		code   int
		errMsg string
	}
	testCases := []struct {
		name string
		args args
		exp  []string
	}{

		{
			name: "success. with err message",
			args: args{
				code:   http.StatusInternalServerError,
				errMsg: "internal",
			},
			exp: []string{`"status": "error"`, `"err_msg": "internal"`},
		},
		{
			name: "success. without err message",
			args: args{
				code: http.StatusBadRequest,
			},
			exp: []string{`"status": "error"`, `"err_msg": "400. Bad Request"`},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {

			w := httptest.NewRecorder()
			r := &http.Request{}

			RespondAndLogError(w, r, tc.args.code, tc.args.errMsg, nil)

			res := w.Result()
			res.Body.Close()
			data, err := io.ReadAll(res.Body)
			require.Nil(t, err)
			header := res.Header.Get("Content-Type")

			assert.Equal(t, res.StatusCode, tc.args.code)
			assert.Equal(t, "application/json; charset=utf-8", header)
			for _, mes := range tc.exp {
				assert.Contains(t, string(data), mes)
			}
		})
	}
}
