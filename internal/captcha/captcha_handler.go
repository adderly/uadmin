package captcha

import (
	"github.com/uadmin/uadmin/internal/captcha/logic/captdata"
	"github.com/uadmin/uadmin/internal/captcha/logic/checkdata"
	"net/http"
)

func CaptchaHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		captdata.GetClickBasicCaptData(w, r)
		return
	} else if r.Method == "POST" {
		checkdata.CheckClickData(w, r)
		return
	}
}
