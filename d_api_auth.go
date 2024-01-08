package uadmin

import (
	"net/http"
)

func dAPIAuthHandler(w http.ResponseWriter, r *http.Request, s *Session) {

	if DisableDAPIAuth {
		w.WriteHeader(http.StatusForbidden)
		ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"err_msg": "dAPI auth is disabled",
		})
		return
	}

	modelKV := r.Context().Value(CKey("modelName")).(DApiModelKeyVal)
	command := modelKV.CommandName

	if APIPreAuthHandler != nil {
		e, edata := APIPreAuthHandler(w, r, command)
		if e != nil {
			w.WriteHeader(http.StatusBadRequest)
			// we got data to return
			if len(edata) > 0 {
				ReturnJSON(w, r, edata)
			} else {
				ReturnJSON(w, r, map[string]interface{}{
					"status":  "error",
					"err_msg": e.Error(),
				})
			}
			return
		}
	}

	switch command {
	case "login":
		dAPILoginHandler(w, r, s)
	case "logout":
		dAPILogoutHandler(w, r, s)
	case "signup":
		dAPISignupHandler(w, r, s)
	case "resetpassword":
		dAPIResetPasswordHandler(w, r, s)
	case "changepassword":
		dAPIChangePasswordHandler(w, r, s)
	case "openidlogin":
		dAPIOpenIDLoginHandler(w, r, s)
	case "certs":
		dAPIOpenIDCertHandler(w, r)
	default:
		w.WriteHeader(http.StatusNotFound)
		ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"err_msg": "Unknown auth command: (" + r.URL.Path + ")",
		})
	}
}
