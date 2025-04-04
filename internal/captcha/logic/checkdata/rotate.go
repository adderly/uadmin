package checkdata

import (
	"encoding/json"
	"fmt"
	"github.com/rotisserie/eris"
	"github.com/uadmin/uadmin/internal/captcha/cache"
	"net/http"
	"strconv"

	"github.com/wenlng/go-captcha/v2/rotate"
)

// CheckRotateData .
func CheckRotateData(w http.ResponseWriter, r *http.Request) {
	code := 1
	_ = r.ParseForm()
	angle := r.Form.Get("angle")
	key := r.Form.Get("key")
	if angle == "" || key == "" {
		bt, _ := json.Marshal(map[string]interface{}{
			"code":    code,
			"message": "angle or key param is empty",
		})
		_, _ = fmt.Fprintf(w, string(bt))
		return
	}

	cacheDataByte := cache.ReadCache(key)
	if len(cacheDataByte) == 0 {
		bt, _ := json.Marshal(map[string]interface{}{
			"code":    code,
			"message": "illegal key",
		})
		_, _ = fmt.Fprintf(w, string(bt))
		return
	}

	var dct *rotate.Block
	if err := json.Unmarshal(cacheDataByte, &dct); err != nil {
		bt, _ := json.Marshal(map[string]interface{}{
			"code":    code,
			"message": "illegal key",
		})
		_, _ = fmt.Fprintf(w, string(bt))
		return
	}

	sAngle, _ := strconv.ParseFloat(fmt.Sprintf("%v", angle), 64)
	chkRet := rotate.CheckAngle(int64(sAngle), int64(dct.Angle), 2)

	if chkRet {
		code = 0
	}

	bt, _ := json.Marshal(map[string]interface{}{
		"code": code,
	})
	_, _ = fmt.Fprintf(w, string(bt))
	return
}

// CheckRotateCaptcha .
func CheckRotateCaptcha(dataAngle string, cacheDataByte []byte) (error, int) {
	code := 1

	var dct *rotate.Block
	if err := json.Unmarshal(cacheDataByte, &dct); err != nil {
		return eris.New("illegal key rotate"), code
	}

	sAngle, _ := strconv.ParseFloat(fmt.Sprintf("%v", dataAngle), 64)
	chkRet := rotate.CheckAngle(int64(sAngle), int64(dct.Angle), 2)

	// ret == ok
	if chkRet {
		return nil, 0
	}

	return eris.New("invalid data provided"), code
}
