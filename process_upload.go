package uadmin

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/segmentio/fasthash/fnv1a"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/nfnt/resize"
)

// GetImageSizer can be inplemented for any model to customize the image size uploaded
// to that model
type GetImageSizer interface {
	GetImageSize() (int, int)
}

func processUpload(r *http.Request, f *FieldDefinition, modelName string, session *Session, s *ModelSchema) (val string) {
	base64Format := false
	// Get file description from http request
	httpFile, handler, err := r.FormFile(f.Name)
	if session.ThroughAPI {
		httpFile, handler, err = r.FormFile(f.ColumnName)
	}
	if err != nil {
		if r.Form.Get(f.Name+"-raw") != "" {
			base64Format = true
		} else {
			return ""
		}
	} else {
		defer httpFile.Close()
	}

	// return "", s if there is no file uploaded
	if !base64Format {
		if handler.Filename == "" {
			return ""
		}
	}

	if base64Format {
		filesize := float64(len(r.Form.Get(f.Name+"-raw"))-strings.Index(r.Form.Get(f.Name+"-raw"), "://")) * 0.75
		if int64(filesize) > MaxUploadFileSize {
			f.ErrMsg = fmt.Sprintf("File is too large. Maximum upload file size is: %d Mb", MaxUploadFileSize/1024/1024)
			return ""
		}
	} else {
		if handler.Size > MaxUploadFileSize {
			f.ErrMsg = fmt.Sprintf("File is too large. Maximum upload file size is: %d Mb", MaxUploadFileSize/1024/1024)
			return ""
		}
	}

	// Get the upload to path and create it if it doesn't exist
	uploadTo := "/media/" + f.Type + "s/"
	if f.UploadTo != "" {
		uploadTo = f.UploadTo
	}
	if _, err = os.Stat("." + uploadTo); os.IsNotExist(err) {
		err = os.MkdirAll("."+uploadTo, 0755)
		if err != nil {
			Trail(ERROR, "processForm.MkdirAll. %s", err)
			return ""
		}
	}

	// Generate local file name and create it
	var fName string
	var pathName string
	var fParts []string
	if base64Format {
		fName = r.Form.Get(f.Name + "-raw")[0:strings.Index(r.Form.Get(f.Name+"-raw"), "://")]
		fParts = strings.Split(fName, ".")
	} else {
		fName = handler.Filename
		fName = strings.Replace(fName, "/", "_", -1)
		fName = strings.Replace(fName, "\\", "_", -1)
		fName = strings.Replace(fName, "..", "_", -1)
		fParts = strings.Split(fName, ".")
	}
	fExt := strings.ToLower(fParts[len(fParts)-1])

	pathName = "." + uploadTo + modelName + "_" + f.Name + "_" + GenerateBase64(10) + "/"
	if f.Type == cIMAGE && len(fParts) > 1 {
		fName = strings.TrimSuffix(fName, "."+fExt) + "_raw." + fExt
	} else if f.Type == cIMAGE {
		f.ErrMsg = "Image file with no extension. Please use png, jpg, jpeg or gif."
		return ""
	}

	for _, err = os.Stat(pathName + fName); os.IsExist(err); {
		pathName = "." + uploadTo + modelName + "_" + f.Name + "_" + GenerateBase64(10) + "/"
	}

	// Sanitize the file name
	fName = pathName + path.Clean(fName)

	err = os.MkdirAll(pathName, 0755)
	if err != nil {
		Trail(ERROR, "processForm.MkdirAll. unable to create folder for uploaded file. %s", err)
		return ""
	}
	fRaw, err := os.OpenFile(fName, os.O_WRONLY|os.O_CREATE, DefaultMediaPermission)
	if err != nil {
		Trail(ERROR, "processForm.OpenFile. unable to create file. %s", err)
		return ""
	}

	// Copy http file to local
	if base64Format {
		data, err := base64.StdEncoding.DecodeString(r.Form.Get(f.Name + "-raw")[strings.Index(r.Form.Get(f.Name+"-raw"), "://")+3 : len(r.Form.Get(f.Name+"-raw"))])
		if err != nil {
			Trail(ERROR, "ProcessForm error decoding base64. %s", err)
			return ""
		}
		_, err = fRaw.Write(data)
		if err != nil {
			Trail(ERROR, "ProcessForm error writing file. %s", err)
			return ""
		}
	} else {
		_, err = io.Copy(fRaw, httpFile)
		if err != nil {
			Trail(ERROR, "ProcessForm error uploading http file. %s", err)
			return ""
		}
	}
	fRaw.Close()

	// store the file path to DB
	if f.Type == cFILE {
		val = fmt.Sprint(strings.TrimPrefix(fName, "."))

	} else {
		// If case it is an image, process it first
		fRaw, err = os.Open(fName)
		if err != nil {
			Trail(ERROR, "ProcessForm.Open %s", err)
			return ""
		}

		// decode jpeg,png,gif into image.Image
		var img image.Image
		if fExt == cJPG || fExt == cJPEG {
			img, err = jpeg.Decode(fRaw)
		} else if fExt == cPNG {
			img, err = png.Decode(fRaw)
		} else if fExt == cGIF {
			img, err = gif.Decode(fRaw)
		} else {
			f.ErrMsg = "Unknown image file extension. Please use, png, jpg/jpeg or gif"
			return ""
		}
		if err != nil {
			f.ErrMsg = "Unknown image format or image corrupted."
			Trail(WARNING, "ProcessForm.Decode %s", err)
			return ""
		}

		// Resize the image to fit max height, max width
		width := img.Bounds().Dx()
		height := img.Bounds().Dy()
		model, _ := NewModel(modelName, false)
		// Check if there is a custom image size
		if sizer, ok := model.Interface().(GetImageSizer); ok || height > MaxImageHeight {
			if ok {
				height, width = sizer.GetImageSize()
			} else {
				Ratio := float64(MaxImageHeight) / float64(height)
				width = int(float64(width) * Ratio)
				height = int(float64(height) * Ratio)
				if width > MaxImageWidth {
					Ratio = float64(MaxImageWidth) / float64(width)
					width = int(float64(width) * Ratio)
					height = int(float64(height) * Ratio)
				}
			}
			img = resize.Resize(uint(width), uint(height), img, resize.Lanczos3)
		}

		// Store the active file
		fActiveName := strings.Replace(fName, "_raw", "", -1)
		fActive, err := os.Create(fActiveName)
		if err != nil {
			Trail(ERROR, "ProcessForm.Create unable to create file for resized image. %s", err)
			return ""
		}
		defer fActive.Close()

		fRaw, err = os.OpenFile(fName, os.O_WRONLY, 0644)
		if err != nil {
			Trail(ERROR, "ProcessForm.Open %s", err)
			return ""
		}
		defer fRaw.Close()

		// write new image to file
		if fExt == cJPG || fExt == cJPEG {
			err = jpeg.Encode(fActive, img, nil)
			if err != nil {
				Trail(ERROR, "ProcessForm.Encode active jpg. %s", err)
				return ""
			}

			err = jpeg.Encode(fRaw, img, nil)
			if err != nil {
				Trail(ERROR, "ProcessForm.Encode raw jpg. %s", err)
				return ""
			}
		}

		if fExt == cPNG {
			err = png.Encode(fActive, img)
			if err != nil {
				Trail(ERROR, "ProcessForm.Encode active png. %s", err)
				return ""
			}

			err = png.Encode(fRaw, img)
			if err != nil {
				Trail(ERROR, "ProcessForm.Encode raw png. %s", err)
				return ""
			}
		}

		if fExt == cGIF {
			o := gif.Options{}
			err = gif.Encode(fActive, img, &o)
			if err != nil {
				Trail(ERROR, "ProcessForm.Encode active gif. %s", err)
				return ""
			}

			err = gif.Encode(fRaw, img, &o)
			if err != nil {
				Trail(ERROR, "ProcessForm.Encode raw gif. %s", err)
				return ""
			}
		}
		val = fmt.Sprint(strings.TrimPrefix(fActiveName, "."))
	}

	// Delete old file if it exists and there not required
	if !RetainMediaVersions {
		oldFileName := "." + fmt.Sprint(f.Value)
		oldFileParts := strings.Split(oldFileName, "/")
		os.RemoveAll(strings.Join(oldFileParts[0:len(oldFileParts)-1], "/"))
	}

	if PostUploadHandler != nil {
		val, err = PostUploadHandler(val, modelName, f)
	}

	return val
}

// UploadConf Simple way to pass information for the uploads, not relying solely on Schema
type UploadConf struct {
	FieldDef *FieldDefinition
	//A way to pass a sizer interface for custom resizing.
	ImageSizer GetImageSizer
	// Enable model resizer, the ImageSizer from conf have precedence over
	// the model one; if enabled. defaults to false
	ImageSizeFromModel bool
	//
	Overwrite bool
	//The folder name for the files to be stored in
	FolderName string

	// TODO: The filename will be a hash instead of the filename.ext -> 87sjs9s3dyj.ext
	HashedName bool

	//Hash seed, in case we want to get those names back
	HashSeed string

	// Max upload size, if the specific request should handle bigger files
	MaxUploadFileSize int64
}

func ProcessUpload(r *http.Request, uploadConf UploadConf, session *Session) (val string, errRs error) {

	// Get file description from http request
	var (
		httpFile multipart.File
		handler  *multipart.FileHeader
		err      error
		isBase64 bool
	)

	if session.ThroughAPI {
		httpFile, handler, err = r.FormFile(uploadConf.FieldDef.ColumnName)
	} else {
		httpFile, handler, err = r.FormFile(uploadConf.FieldDef.Name)
	}

	formRawString := r.Form.Get(uploadConf.FieldDef.Name + "-raw")

	if err != nil && formRawString != "" {
		isBase64 = true
	} else if err != nil && formRawString == "" {
		return "", errors.New("field name not found in form")
	} else if handler.Filename == "" {
		return "", errors.New("no file is uploaded")
	}

	defer httpFile.Close()

	maxFileSize := MaxUploadFileSize

	if uploadConf.MaxUploadFileSize > 0 {
		maxFileSize = uploadConf.MaxUploadFileSize
	}

	if isBase64 {
		getF := formRawString
		filesize := float64(len(getF)-strings.Index(getF, "://")) * 0.75
		if int64(filesize) > maxFileSize {
			uploadConf.FieldDef.ErrMsg = fmt.Sprintf("File is too large. Maximum upload file size is: %d Mb", MaxUploadFileSize/1024/1024)
			return "", errors.New(uploadConf.FieldDef.ErrMsg)
		}
	} else {
		if handler.Size > maxFileSize {
			uploadConf.FieldDef.ErrMsg = fmt.Sprintf("File is too large. Maximum upload file size is: %d Mb", MaxUploadFileSize/1024/1024)
			return "", errors.New(uploadConf.FieldDef.ErrMsg)
		}
	}

	// Get the upload to path and create it if it doesn't exist
	uploadTo := "/media/" + uploadConf.FieldDef.Type + "s/"
	if uploadConf.FieldDef.UploadTo != "" {
		uploadTo = uploadConf.FieldDef.UploadTo
	}

	//check if directory exists
	if _, err = os.Stat("." + uploadTo); os.IsNotExist(err) {
		err = os.MkdirAll("."+uploadTo, 0755)
		if err != nil {
			errorStr := fmt.Sprintf("processForm.MkdirAll. %s", err)
			Trail(ERROR, errorStr)
			return "", errors.New(errorStr)
		}
	}

	// Generate local file name and create it
	var fName string
	var pathName string
	var fParts []string
	if isBase64 {
		getF := formRawString
		fName = getF[0:strings.Index(getF, "://")]
		fParts = strings.Split(fName, ".")
	} else {
		fName = handler.Filename
		fName = strings.Replace(fName, "/", "_", -1)
		fName = strings.Replace(fName, "\\", "_", -1)
		fName = strings.Replace(fName, "..", "_", -1)
		fParts = strings.Split(fName, ".")
	}

	// The field type image, was uploaded without extension
	if uploadConf.FieldDef.Type == cIMAGE && len(fParts) < 1 {
		uploadConf.FieldDef.ErrMsg = "Image file with no extension. Please use png, jpg, jpeg or gif."
		return "", errors.New(uploadConf.FieldDef.ErrMsg)
	}

	fExt := strings.ToLower(fParts[len(fParts)-1])
	filenameNoExtension := strings.TrimSuffix(fName, "."+fExt)
	filenameWithExtension := fName

	if uploadConf.FieldDef.Type == cIMAGE && len(fParts) > 1 {
		filenameWithExtension = filenameNoExtension + "_raw." + fExt
	}

	if uploadConf.HashedName {
		filenameNoExtension = fashFile(filenameNoExtension)
		filenameWithExtension = fashFile(filenameNoExtension) + "." + fExt
	}

	//Use the specified folder
	if len(uploadConf.FolderName) > 0 {
		pathName = "." + uploadTo + uploadConf.FolderName + "_" + uploadConf.FieldDef.Name + "/"
	} else {
	nice:
		pathName = "." + uploadTo + uploadConf.FieldDef.ModelName + "_" + uploadConf.FieldDef.Name + "_" + GenerateBase64(10) + "/"
		//Finds a new path location if exists
		for _, err = os.Stat(pathName + filenameWithExtension); os.IsExist(err); {
			goto nice
		}
	}

	// Sanitize the file name
	filenameLocation := pathName + path.Clean(filenameWithExtension)

	err = os.MkdirAll(pathName, 0755)
	if err != nil {
		errorStr := fmt.Sprintf("processForm.MkdirAll. unable to create folder for uploaded file. %s", err)
		Trail(ERROR, errorStr)
		return "", errors.New(errorStr)
	}

	//Save original raw file
	fRaw, err := os.OpenFile(filenameLocation, os.O_WRONLY|os.O_CREATE, DefaultMediaPermission)
	if err != nil {
		errorStr := fmt.Sprintf("processForm.OpenFile. unable to create file. %s", err)
		Trail(ERROR, errorStr)
		return "", errors.New(errorStr)
	}

	// Copy http file to local
	if isBase64 {
		getF := formRawString
		data, err := base64.StdEncoding.DecodeString(getF[strings.Index(getF, "://")+3 : len(getF)])
		if err != nil {
			errorStr := fmt.Sprintf("ProcessForm error decoding base64. %s", err)
			Trail(ERROR, errorStr)
			return "", errors.New(errorStr)
		}
		_, err = fRaw.Write(data)
		if err != nil {
			errorStr := fmt.Sprintf("ProcessForm error writing file. %s", err)
			Trail(ERROR, errorStr)
			return "", errors.New(errorStr)
		}
	} else {
		_, err = io.Copy(fRaw, httpFile)
		if err != nil {
			errorStr := fmt.Sprintf("ProcessForm error uploading http file. %s", err)
			Trail(ERROR, errorStr)
			return "", errors.New(errorStr)
		}
	}
	fRaw.Close()

	// store the file path to DB
	if uploadConf.FieldDef.Type == cFILE {
		val = fmt.Sprint(strings.TrimPrefix(filenameLocation, "."))
	} else {
		// If case it is an image, process it first
		fRaw, err = os.Open(filenameLocation)
		if err != nil {
			errorStr := fmt.Sprintf("ProcessForm.Open %s", err)
			Trail(ERROR, errorStr)
			return "", errors.New(errorStr)
		}

		// decode jpeg,png,gif into image.Image
		var img image.Image
		if fExt == cJPG || fExt == cJPEG {
			img, err = jpeg.Decode(fRaw)
		} else if fExt == cPNG {
			img, err = png.Decode(fRaw)
		} else if fExt == cGIF {
			img, err = gif.Decode(fRaw)
		} else {
			uploadConf.FieldDef.ErrMsg = "Unknown image file extension. Please use, png, jpg/jpeg or gif"
			return "", errors.New(uploadConf.FieldDef.ErrMsg)
		}

		if err != nil {
			uploadConf.FieldDef.ErrMsg = "Unknown image format or image corrupted."
			Trail(WARNING, "ProcessForm.Decode %s", err)
			return "", errors.New(uploadConf.FieldDef.ErrMsg)
		}

		// Resize the image to fit max height, max width
		width := img.Bounds().Dx()
		height := img.Bounds().Dy()

		sizer := uploadConf.ImageSizer

		if sizer != nil {
			height, width = sizer.GetImageSize()
		} else if uploadConf.ImageSizeFromModel {
			model, _ := NewModel(uploadConf.FieldDef.ModelName, false)
			sizerModel, _ := model.Interface().(GetImageSizer)
			sizer = sizerModel
		}

		// Check if there is a custom image size
		//TODO: handle MaxImageSize per upload conf
		if sizer != nil || height > MaxImageHeight {
			if sizer != nil {
				height, width = sizer.GetImageSize()
			} else {
				Ratio := float64(MaxImageHeight) / float64(height)
				width = int(float64(width) * Ratio)
				height = int(float64(height) * Ratio)
				if width > MaxImageWidth {
					Ratio = float64(MaxImageWidth) / float64(width)
					width = int(float64(width) * Ratio)
					height = int(float64(height) * Ratio)
				}
			}
			img = resize.Resize(uint(width), uint(height), img, resize.Lanczos3)
		}

		// Store the active file
		fActiveName := strings.Replace(filenameLocation, "_raw", "", -1)
		fActive, err := os.Create(fActiveName)
		if err != nil {
			errorStr := fmt.Sprintf("ProcessForm.Create unable to create file for resized image. %s", err)
			Trail(ERROR, errorStr)
			return "", errors.New(errorStr)
		}
		defer fActive.Close()

		fRaw, err = os.OpenFile(filenameLocation, os.O_WRONLY, 0644)
		if err != nil {
			errorStr := fmt.Sprintf("ProcessForm.Open %s", err)
			Trail(ERROR, errorStr)
			return "", errors.New(errorStr)
		}
		defer fRaw.Close()

		// write new image to file
		if fExt == cJPG || fExt == cJPEG {
			err = jpeg.Encode(fActive, img, nil)
			if err != nil {
				errorStr := fmt.Sprintf("ProcessForm.Encode active jpg. %s", err)
				Trail(ERROR, errorStr)
				return "", errors.New(errorStr)
			}

			err = jpeg.Encode(fRaw, img, nil)
			if err != nil {
				errorStr := fmt.Sprintf("ProcessForm.Encode raw jpg. %s", err)
				Trail(ERROR, errorStr)
				return "", errors.New(errorStr)
			}
		}

		if fExt == cPNG {
			err = png.Encode(fActive, img)
			if err != nil {
				errorStr := fmt.Sprintf("ProcessForm.Encode active png. %s", err)
				Trail(ERROR, errorStr)
				return "", errors.New(errorStr)
			}

			err = png.Encode(fRaw, img)
			if err != nil {
				errorStr := fmt.Sprintf("ProcessForm.Encode raw png. %s", err)
				Trail(ERROR, errorStr)
				return "", errors.New(errorStr)
			}
		}

		if fExt == cGIF {
			o := gif.Options{}
			err = gif.Encode(fActive, img, &o)
			if err != nil {
				errorStr := fmt.Sprintf("ProcessForm.Encode active gif. %s", err)
				Trail(ERROR, errorStr)
				return "", errors.New(errorStr)
			}

			err = gif.Encode(fRaw, img, &o)
			if err != nil {
				errorStr := fmt.Sprintf("ProcessForm.Encode raw gif. %s", err)
				Trail(ERROR, errorStr)
				return "", errors.New(errorStr)
			}
		}
		val = fmt.Sprint(strings.TrimPrefix(fActiveName, "."))
	}

	// Delete old file if it exists and there not required
	if !RetainMediaVersions {
		oldFileName := "." + fmt.Sprint(uploadConf.FieldDef.Value)
		oldFileParts := strings.Split(oldFileName, "/")
		os.RemoveAll(strings.Join(oldFileParts[0:len(oldFileParts)-1], "/"))
	}

	if PostUploadHandler != nil {
		//TODO: error handling better
		val, err = PostUploadHandler(val, uploadConf.FieldDef.ModelName, uploadConf.FieldDef)
	}

	return val, err
}

func fashFile(filenameString string) string {
	h1 := fnv1a.HashString64(filenameString)
	fmt.Printf("FNV-1a hash of '%v': %v", filenameString, h1)
	return fmt.Sprint(h1)
}
