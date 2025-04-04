package uadmin

import "net/http"

func dAPISignupHandler(w http.ResponseWriter, r *http.Request, s *Session) {
	// Check if signup API is allowed
	if !AllowDAPISignup {
		w.WriteHeader(http.StatusForbidden)
		ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"err_msg": "Signup API is disabled",
		})
		return
	}

	// get variables from request
	username := r.FormValue("username")
	email := r.FormValue("email")
	firstName := r.FormValue("first_name")
	lastName := r.FormValue("last_name")
	password := r.FormValue("password")
	proof := r.FormValue("proof")

	if proof == "" {
	}
	// set the username to email if there is no username
	if username == "" && email != "" {
		username = email
	}

	// check if password is empty
	if password == "" {
		w.WriteHeader(http.StatusBadRequest)
		ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"err_msg": "password is empty",
		})
		return
	}

	// create user object
	user := User{
		Username:     username,
		FirstName:    firstName,
		LastName:     lastName,
		Password:     password,
		Email:        email,
		Active:       DAPISignupActive,
		Admin:        false,
		RemoteAccess: DAPISignupAllowRemote,
		UserGroupID:  uint(DAPISignupGroupID),
		CreatedByAPI: true,
	}

	if !DAPIAllowDuplicatedEmail {
		err := user.EmailExists()
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			ReturnJSON(w, r, map[string]interface{}{
				"status":  "error",
				"err_msg": err.Error(),
			})
			return
		}

	}

	// run custom validation
	if SignupValidationHandler != nil {
		err := SignupValidationHandler(&user, r)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			ReturnJSON(w, r, map[string]interface{}{
				"status":  "error",
				"err_msg": err.Error(),
			})
			return
		}
	}

	if err, _ := user.Validate(); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"err_msg": err.Error(),
		})
		return
	}

	// Save user record
	user.Save()

	// Check if the record was not saved, that means the username is taken
	//TODO: remove and validate with the badder
	if user.ID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		ReturnJSON(w, r, map[string]interface{}{
			"status":  "error",
			"err_msg": "username taken",
		})
		return
	}

	if CustomDAPISignupHandler != nil {
		//TODO: save errors like this
		e, edata := CustomDAPISignupHandler(r, &user)
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

	// if the user is active, then login in
	if user.Active {
		dAPILoginHandler(w, r, s)
		return
	}

	ReturnJSON(w, r, map[string]interface{}{
		"status": "ok",
	})
}
