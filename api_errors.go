package uadmin

import (
	"fmt"
	"github.com/rotisserie/eris"
	"net/http"
)

var (
	E_UsernameTaken   = eris.New("username already taken")
	E_InvalidPassword = eris.New("Password errors")
)

const (
	contentTypeHeader = "Content-Type"
	jsonContentType   = "application/json; charset=utf-8"
	errStatus         = "error"
)

func RespondAndLogError(w http.ResponseWriter, r *http.Request, code int, errMsg string, err error) {
	// log original error
	logError(r, errMsg, err)

	if errMsg == "" {
		errMsg = fmt.Sprintf("%d. %s", code, http.StatusText(code))
	}
	w.Header().Set(contentTypeHeader, jsonContentType)
	w.WriteHeader(code)
	ReturnJSON(w, r, map[string]interface{}{
		"status":  errStatus,
		"err_msg": errMsg,
	})
}

func logError(r *http.Request, msg string, err error) {
	method := r.Method
	uri := r.RequestURI
	logMessage := fmt.Sprintf("failed [%s] to [%s], msg: %s", method, uri, msg)
	Trail(ERROR, logMessage, err)
}
