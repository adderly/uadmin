package captcha

import (
	"github.com/kataras/iris/v12"
	"github.com/uadmin/uadmin/internal/captcha/logic/captdata"
	"github.com/uadmin/uadmin/internal/captcha/logic/checkdata"
	"log"
	"net/http"
	"time"
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

// Create middleware for requiring request with captcha code

// Function that receive all requests and handle the type of captcha accordingly.
func UadminIrisAuthMidleware(ctx iris.Context) {
	t := time.Now()

	// Set a shared variable between handlers
	ctx.Values().Set("framework", "iris")

	// before request

	//req := ctx.Request()

	//s := uadmin.IsAuthenticated(req)
	//log.Println(s)
	//if s == nil {
	//	ctx.Values().Set("message", "this is the error message")
	//	ctx.StatusCode(iris.StatusUnauthorized)
	//	return
	//} else {
	//	SetContextSession(ctx, s)
	//}

	ctx.Next()

	// after request
	latency := time.Since(t)
	log.Print(latency)

	// access the status we are sending
	status := ctx.GetStatusCode()
	log.Println(status)
}

func ValidateCaptchaData(req *http.Request) error {
	return nil
}
