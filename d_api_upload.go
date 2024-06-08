package uadmin

import (
	"net/http"
	"strings"
)

func dAPIUpload(w http.ResponseWriter, r *http.Request, schema *ModelSchema) (map[string]string, error) {
	fileList := map[string]string{}

	if r.MultipartForm == nil {
		return fileList, nil
	}

	// make a list of files
	kList := []string{}
	for k := range r.MultipartForm.File {
		kList = append(kList, k)
	}

	for _, k := range kList {
		// Process File
		var field *FieldDefinition = schema.FieldByColumnName(k[1:])
		if field == nil {
			Trail(WARNING, "dAPIUpload received a file that has no field: %s", k)
			continue
		}

		r.MultipartForm.File[k[1:]] = r.MultipartForm.File[k]

		s := r.Context().Value(CKey("session"))
		var session *Session
		if s != nil {
			session = s.(*Session)
		}

		fileName := processUpload(r, field, schema.ModelName, session, schema)
		if fileName != "" {
			fileList[field.ColumnName] = fileName
		}
	}
	return fileList, nil
}

func DAPIUploadWithModel(r *http.Request, schema *ModelSchema, session *Session) (map[string]string, error) {
	fileList := map[string]string{}

	// Parse the Form
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		r.ParseForm()
	}

	if r.MultipartForm == nil {
		return fileList, nil
	}

	// make a list of files
	kFileList := []string{}
	for k := range r.MultipartForm.File {
		kFileList = append(kFileList, k)
	}

	for _, kFile := range kFileList {
		// Process File
		cname := kFile[1:]
		var field *FieldDefinition = schema.FieldByColumnName(cname)
		if field == nil {
			Trail(WARNING, "dAPIUpload received a file that has no field: %s", kFile)
			continue
		}

		r.MultipartForm.File[kFile[1:]] = r.MultipartForm.File[kFile]

		fileName := processUpload(r, field, schema.ModelName, session, schema)
		if fileName != "" {
			fileList[field.ColumnName] = fileName
		}
	}
	return fileList, nil
}

// Only upload the file for this field.
func DAPIUploadWithField(r *http.Request, fieldDefinition *FieldDefinition, session *Session) (map[string]string, error) {
	fileList := map[string]string{}

	// Parse the Form
	err := r.ParseMultipartForm(32 << 20)
	if err != nil {
		r.ParseForm()
	}

	if r.MultipartForm == nil {
		return fileList, nil
	}

	// make a list of files
	kFileList := []string{}
	for k := range r.MultipartForm.File {
		kFileList = append(kFileList, k)
	}

	for _, kFile := range kFileList {
		// Process File
		//TODO: document the lower dash in column names
		cname := kFile[1:] // remove the _ from the filename _imagefile -> imagefile
		if !strings.EqualFold(cname, fieldDefinition.ColumnName) {
			Trail(WARNING, "dAPIUpload received a file that has no field: %s", kFile)
			continue
		}

		r.MultipartForm.File[kFile[1:]] = r.MultipartForm.File[kFile]

		upConf := UploadConf{
			FieldDef:           fieldDefinition,
			ImageSizeFromModel: true,
		}

		fileName, procErr := ProcessUpload(r, upConf, session)
		if fileName != "" {
			fileList[fieldDefinition.ColumnName] = fileName
		} else {
			Trail(WARNING, "ProcessUpload failed proccing upload: %s", procErr)
		}
	}
	return fileList, nil
}
