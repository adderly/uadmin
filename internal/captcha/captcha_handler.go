package captcha

import (
	"fmt"
	"github.com/kataras/iris/v12"
	"github.com/rotisserie/eris"
	"github.com/uadmin/uadmin/internal/captcha/cache"
	"github.com/uadmin/uadmin/internal/captcha/logic/captdata"
	"github.com/uadmin/uadmin/internal/captcha/logic/checkdata"
	"log"
	"net/http"
	"strconv"
	"time"
)

var (
	CAPTCHA_ERROR   = 1
	CAPTCHA_SUCCESS = 0
	CAPTCHA_INVALID = 2
)

var (
	CAPTCHA_VERSION          = "1.0.0"
	CAPTCHA_MSG_OK           = ""
	CAPTCHA_MSG_INVALID_KEY  = ""
	CAPTCHA_MSG_INVALID_DATA = ""
)

type CaptchaType uint

const (
	CLICK        CaptchaType = iota
	ROTATE                   = 2
	SLIDE                    = 3
	CLICK_SHAPE              = 4
	SLIDE_REGION             = 5
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

// CaptchaHttpHandler
func CaptchaHttpHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		err, _, rs := CaptchaGen(w, r)
		if err != nil {
			_, _ = fmt.Fprintf(w, string(rs))
			w.WriteHeader(http.StatusOK)
		} else {
			_, _ = fmt.Fprintf(w, err.Error())
			w.WriteHeader(http.StatusBadRequest)
		}

		return
	} else if r.Method == "POST" {
		err, code := CaptchaCheck(w, r)
		if err != nil {
			_, _ = fmt.Fprintf(w, string(code))
			w.WriteHeader(http.StatusOK)
		} else {
			_, _ = fmt.Fprintf(w, err.Error())
			w.WriteHeader(http.StatusBadRequest)
		}
		return
	}
}

func CaptchaGen(w http.ResponseWriter, req *http.Request) (error, int, []byte) {
	_ = req.ParseForm()

	captchaType := req.Form.Get("captchaType")

	captchaTypeVal, err := strconv.Atoi(captchaType)
	if err != nil {
		return eris.New("illegal captchaType"), CAPTCHA_ERROR, nil
	}

	captchaT := CaptchaType(captchaTypeVal)
	var er error
	var code = CAPTCHA_ERROR
	var rs []byte

	// all functions need to handle data and return error messages with captchacode
	switch captchaT {
	case CLICK:
		er, code, rs = captdata.GenClickBasicCaptData(w, req)
		break
	case CLICK_SHAPE:
		er, code, rs = captdata.GenClickShapesCaptData(w, req)
		break
	case SLIDE:
		er, code, rs = captdata.GenSlideBasicCaptData(w, req)
		break
	case SLIDE_REGION:
		er, code, rs = captdata.GenSlideRegionCaptData(w, req)
		break
	case ROTATE:
		er, code, rs = captdata.GenRotateBasicCaptData(w, req)
		break
	default:
		er = eris.New("InvalidCaptchaType")
		code = CAPTCHA_ERROR
	}

	if code != CAPTCHA_SUCCESS {
		return er, CAPTCHA_ERROR, nil
	}

	return nil, CAPTCHA_SUCCESS, rs
}

func CaptchaCheck(w http.ResponseWriter, req *http.Request) (error, int) {
	_ = req.ParseForm()

	key := req.Form.Get("key")
	data := req.Form.Get("data")
	captchaType := req.Form.Get("captchaType")

	log.Println(key)
	if key == "" || data == "" {
		return eris.New("Request does not have validation key"), CAPTCHA_ERROR
	}

	cacheDataByte := cache.ReadCache(key)
	if len(cacheDataByte) == 0 {
		return eris.New("illegal key"), CAPTCHA_ERROR
	}

	captchaTypeVal, err := strconv.Atoi(captchaType)
	if err != nil {
		return eris.New("illegal captchaType"), CAPTCHA_ERROR
	}

	captchaT := CaptchaType(captchaTypeVal)
	var er error
	var code = CAPTCHA_ERROR

	// all functions need to handle data and return error messages with captchacode
	switch captchaT {
	case CLICK:
	case CLICK_SHAPE:
		er, code = checkdata.CheckClickCaptcha(data, cacheDataByte)
		break
	case SLIDE:
	case SLIDE_REGION:
		er, code = checkdata.CheckSlideCaptcha(data, cacheDataByte)
		break
	case ROTATE:
		er, code = checkdata.CheckRotateCaptcha(data, cacheDataByte)
		break
	default:
		er = eris.New("InvalidCaptchaType")
		code = CAPTCHA_ERROR
	}

	if code != CAPTCHA_SUCCESS {
		return er, CAPTCHA_ERROR
	}

	return nil, CAPTCHA_SUCCESS
}

// WithCaptchaMiddleware Function that receive all requests and handle the type of captcha accordingly.
func WithCaptchaMiddleware(ctx iris.Context) {
	t := time.Now()
	// Set a shared variable between handlers
	ctx.Values().Set("framework", "iris")

	// before request

	req := ctx.Request()

	_ = req.ParseForm()

	key := req.Form.Get("key")
	data := req.Form.Get("data")
	captchaType := req.Form.Get("captchaType")

	log.Println(key)
	if key == "" || data == "" {
		ctx.Values().Set("message", "Request does not have validation key")
		ctx.StatusCode(iris.StatusUnauthorized)
		return
	}

	cacheDataByte := cache.ReadCache(key)
	if len(cacheDataByte) == 0 {
		ctx.Values().Set("code", CAPTCHA_ERROR)
		ctx.Values().Set("message", "illegal key")
		ctx.StatusCode(iris.StatusBadRequest)
		return
	}

	captchaTypeVal, err := strconv.Atoi(captchaType)
	if err != nil {
		ctx.Values().Set("code", CAPTCHA_ERROR)
		ctx.Values().Set("message", "illegal captchaType")
		ctx.StatusCode(iris.StatusBadRequest)
		return
	}

	captchaT := CaptchaType(captchaTypeVal)
	var er error
	var code = CAPTCHA_ERROR

	// all functions need to handle data and return error messages with captchacode
	switch captchaT {
	case CLICK:
	case CLICK_SHAPE:
		er, code = checkdata.CheckClickCaptcha(data, cacheDataByte)
		break
	case SLIDE:
	case SLIDE_REGION:
		er, code = checkdata.CheckSlideCaptcha(data, cacheDataByte)
		break
	case ROTATE:
		er, code = checkdata.CheckRotateCaptcha(data, cacheDataByte)
		break
	default:
		er = eris.New("InvalidCaptchaType")
		code = CAPTCHA_ERROR
	}

	if code != CAPTCHA_SUCCESS {
		ctx.Values().Set("code", code)
		ctx.Values().Set("message", er.Error())
		ctx.StatusCode(iris.StatusBadRequest)
		return
	}

	ctx.Next()

	// after request
	latency := time.Since(t)
	log.Print(latency)

	// access the status we are sending
	status := ctx.GetStatusCode()
	log.Println(status)
}
