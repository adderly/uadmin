package captdata

import (
	"encoding/json"
	"fmt"
	"github.com/rotisserie/eris"
	"github.com/uadmin/uadmin/internal/captcha/cache"
	"github.com/uadmin/uadmin/internal/captcha/helper"
	"github.com/wenlng/go-captcha/v2/base/option"
	"log"
	"net/http"

	"github.com/wenlng/go-captcha-assets/resources/images"
	"github.com/wenlng/go-captcha/v2/rotate"
)

var rotateBasicCapt rotate.Captcha

func init() {
	builder := rotate.NewBuilder(rotate.WithRangeAnglePos([]option.RangeVal{
		{Min: 20, Max: 330},
	}))

	// background images
	imgs, err := images.GetImages()
	if err != nil {
		log.Fatalln(err)
	}

	// set resources
	builder.SetResources(
		rotate.WithImages(imgs),
	)

	rotateBasicCapt = builder.Make()
}

// GetRotateBasicCaptData .
func GetRotateBasicCaptData(w http.ResponseWriter, r *http.Request) {
	captData, err := rotateBasicCapt.Generate()
	if err != nil {
		log.Fatalln(err)
	}

	blockData := captData.GetData()
	if blockData == nil {
		bt, _ := json.Marshal(map[string]interface{}{
			"code":    1,
			"message": "gen captcha data failed",
		})
		_, _ = fmt.Fprintf(w, string(bt))
		return
	}

	var masterImageBase64, thumbImageBase64 string
	masterImageBase64, err = captData.GetMasterImage().ToBase64()
	if err != nil {
		bt, _ := json.Marshal(map[string]interface{}{
			"code":    1,
			"message": "base64 data failed",
		})
		_, _ = fmt.Fprintf(w, string(bt))
		return
	}

	thumbImageBase64, err = captData.GetThumbImage().ToBase64()
	if err != nil {
		bt, _ := json.Marshal(map[string]interface{}{
			"code":    1,
			"message": "base64 data failed",
		})
		_, _ = fmt.Fprintf(w, string(bt))
		return
	}

	dotsByte, _ := json.Marshal(blockData)
	key := helper.StringToMD5(string(dotsByte))
	cache.WriteCache(key, dotsByte)

	bt, _ := json.Marshal(map[string]interface{}{
		"code":         0,
		"captcha_key":  key,
		"image_base64": masterImageBase64,
		"thumb_base64": thumbImageBase64,
	})
	_, _ = fmt.Fprintf(w, string(bt))
}

// GenRotateBasicCaptData .
func GenRotateBasicCaptData(w http.ResponseWriter, r *http.Request) (error, int, []byte) {
	captData, err := rotateBasicCapt.Generate()
	if err != nil {
		return err, 1, nil
	}

	blockData := captData.GetData()
	if blockData == nil {

		return eris.New("gen captcha data failed"), 1, nil
	}

	var masterImageBase64, thumbImageBase64 string
	masterImageBase64, err = captData.GetMasterImage().ToBase64()
	if err != nil {

		return eris.New("base64 data failed"), 1, nil
	}

	thumbImageBase64, err = captData.GetThumbImage().ToBase64()
	if err != nil {

		return eris.New("base64 data failed"), 1, nil
	}

	dotsByte, _ := json.Marshal(blockData)
	key := helper.StringToMD5(string(dotsByte))
	cache.WriteCache(key, dotsByte)

	bt, _ := json.Marshal(map[string]interface{}{
		"code":         0,
		"captcha_key":  key,
		"image_base64": masterImageBase64,
		"thumb_base64": thumbImageBase64,
	})
	return nil, 0, bt
}
