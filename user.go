package uadmin

import (
	"errors"
	"fmt"
	"github.com/rotisserie/eris"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/arbrix/uadmin/helper"
)

// User !
type User struct {
	Model
	Username     string    `uadmin:"required;filter;search"`
	FirstName    string    `uadmin:"filter;search"`
	LastName     string    `uadmin:"filter;search"`
	Password     string    `uadmin:"required;password;help:To reset password, clear the field and type a new password.;list_exclude"`
	Email        string    `uadmin:"email;search"`
	Active       bool      `uadmin:"filter"`
	Admin        bool      `uadmin:"filter"`
	RemoteAccess bool      `uadmin:"filter"`
	UserGroup    UserGroup `uadmin:"filter"`
	UserGroupID  uint
	Photo        string `uadmin:"image"`
	//Language     []Language `gorm:"many2many:user_languages" listExclude:"true"`
	LastLogin     *time.Time `uadmin:"read_only"`
	ExpiresOn     *time.Time
	OTPRequired   bool
	OTPSeed       string `uadmin:"list_exclude;hidden;read_only;password"`
	PasswordReset *time.Time
	CreatedByAPI  bool
}

// String return string
func (u User) String() string {
	return fmt.Sprintf("%s %s", u.FirstName, u.LastName)
}

// Save !
func (u *User) Save() {
	err, _ := u.Validate()
	if err != nil {
		return
	}

	if !strings.HasPrefix(u.Password, "$2a$") || len(u.Password) != 60 {
		u.Password = hashPass(u.Password)
	}
	if u.OTPSeed == "" {
		u.OTPSeed, _ = generateOTPSeed(OTPDigits, OTPAlgorithm, OTPSkew, OTPPeriod, u)
	} else if u.ID != 0 {
		oldUser := User{}
		Get(&oldUser, "id = ?", u.ID)
		if !oldUser.OTPRequired && u.OTPRequired {
			u.OTPSeed, _ = generateOTPSeed(OTPDigits, OTPAlgorithm, OTPSkew, OTPPeriod, u)
		}
	}
	u.Username = strings.ToLower(u.Username)

	Save(u)
	loadSessions()
}

// GetActiveSession !
func (u *User) GetActiveSession() *Session {
	s := Session{}
	Get(&s, "`user_id` = ? AND `active` = ?", u.ID, true)
	if s.ID == 0 {
		return nil
	}
	return &s
}

// Login Logs in user using password and otp. If there is no OTP, just pass an empty string
func (u *User) Login(pass string, otp string) *Session {
	if u == nil {
		return nil
	}

	err := verifyPassword(u.Password, pass)
	if err == nil && u.ID != 0 {
		s := u.GetActiveSession()
		if s == nil {
			s = &Session{}
			s.Active = true
			s.UserID = u.ID
			s.LoginTime = time.Now()
			s.GenerateKey()
		}
		s.LastLogin = time.Now()
		if CookieTimeout > -1 {
			ExpiresOn := s.LastLogin.Add(time.Second * time.Duration(CookieTimeout))
			s.ExpiresOn = &ExpiresOn
		}
		if u.OTPRequired {
			if otp == "" {
				Trail(INFO, "OTP login for: %s is %s", u.Username, u.GetOTP())
				s.PendingOTP = true
			} else {
				s.PendingOTP = !u.VerifyOTP(otp)
			}
		}
		u.LastLogin = &s.LastLogin
		u.Save()
		s.Save()
		return s
	}
	return nil
}

// GetDashboardMenu !
func (u *User) GetDashboardMenu() (menus []DashboardMenu) {
	allItems := []DashboardMenu{}
	All(&allItems)

	userItems := []UserPermission{}
	Filter(&userItems, "user_id = ?", u.ID)

	groupItems := []GroupPermission{}
	Filter(&groupItems, "user_group_id = ?", u.UserGroupID)

	var groupItemIndex int
	var userItemIndex int
	dashboardItems := []DashboardMenu{}
	for _, item := range allItems {
		groupItemIndex = -1
		userItemIndex = -1
		for i, groupItem := range groupItems {
			if groupItem.DashboardMenuID == item.ID {
				groupItemIndex = i
				break
			}
		}
		for i, userItem := range userItems {
			if userItem.DashboardMenuID == item.ID {
				userItemIndex = i
				break
			}
		}
		// Permission exists for group and user: overide group with user
		if groupItemIndex != -1 && userItemIndex != -1 {
			groupItems[groupItemIndex].Read = userItems[userItemIndex].Read
			groupItems[groupItemIndex].Add = userItems[userItemIndex].Add
			groupItems[groupItemIndex].Edit = userItems[userItemIndex].Edit
			groupItems[groupItemIndex].Delete = userItems[userItemIndex].Delete
		}
		// User permission exists but no group, add it to permessions
		if groupItemIndex == -1 && userItemIndex != -1 {
			groupItems = append(groupItems, GroupPermission{
				DashboardMenuID: userItems[userItemIndex].DashboardMenuID,
				Read:            userItems[userItemIndex].Read,
				Add:             userItems[userItemIndex].Add,
				Edit:            userItems[userItemIndex].Edit,
				Delete:          userItems[userItemIndex].Delete,
			})
			groupItemIndex = len(groupItems) - 1
		}
		// Reconstruct the dashboard list
		if u.Admin || groupItemIndex != -1 || userItemIndex != -1 {
			if u.Admin || groupItems[groupItemIndex].Read {
				dashboardItems = append(dashboardItems, item)
			}
		}
	}
	return dashboardItems
}

// HasAccess returns the user level permission to a model. The modelName
// the the URL of the model
func (u *User) HasAccess(modelName string) UserPermission {
	Trail(WARNING, "User.HasAccess was deprecated in version 0.6.0. Use User.GetAccess instead.")
	return u.hasAccess(modelName)
}

// hasAccess returns the user level permission to a model. The modelName
// the the URL of the model
func (u *User) hasAccess(modelName string) UserPermission {
	up := UserPermission{}
	dm := DashboardMenu{}
	if CachePermissions {
		modelID := uint(0)
		for _, m := range cachedModels {
			if m.URL == modelName {
				modelID = m.ID
				break
			}
		}
		for _, p := range cacheUserPerms {
			if p.UserID == u.ID && p.DashboardMenuID == modelID {
				up = p
				break
			}
		}
	} else {
		Get(&dm, "url = ?", modelName)
		Get(&up, "user_id = ? and dashboard_menu_id = ?", u.ID, dm.ID)
	}
	return up
}

// GetAccess returns the user's permission to a dashboard menu based on
// their admin status, group and user permissions
func (u *User) GetAccess(modelName string) UserPermission {
	// Check if the user has permission to a model
	if u.UserGroup.ID != u.UserGroupID {
		Preload(u)
	}
	uPerm := u.hasAccess(modelName)
	gPerm := u.UserGroup.hasAccess(modelName)
	perm := UserPermission{}

	if gPerm.ID != 0 {
		perm.Read = gPerm.Read
		perm.Edit = gPerm.Edit
		perm.Add = gPerm.Add
		perm.Delete = gPerm.Delete
		perm.Approval = gPerm.Approval
	}
	if uPerm.ID != 0 {
		perm.Read = uPerm.Read
		perm.Edit = uPerm.Edit
		perm.Add = uPerm.Add
		perm.Delete = uPerm.Delete
		perm.Approval = uPerm.Approval
	}
	if u.Admin {
		perm.Read = true
		perm.Edit = true
		perm.Add = true
		perm.Delete = true
		perm.Approval = true
	}
	return perm
}

// Validate user when saving from uadmin
func (u User) Validate() (err error, ret map[string]string) {
	ret = map[string]string{}

	userNameErrMess := u.validateUserName()
	if userNameErrMess != nil {
		ret["Username"] = userNameErrMess.Error()
		err = userNameErrMess
	}

	passErrMess := u.validatePass()
	if passErrMess != nil {
		ret["Password"] = passErrMess.Error()
		err = passErrMess
	}

	return
}

func (u User) validateUserName() error {
	if u.ID == 0 {
		Get(&u, "username=?", strings.ToLower(u.Username))
		if u.ID > 0 {
			return E_UsernameTaken
		}
	}
	return nil
}

func (u User) validatePass() error {
	validator := helper.NewPasswordValidator()
	err := validator.Validate(u.Password)
	if err != nil {
		return eris.Wrapf(E_InvalidPassword, "invalid password: %s", err)
	}

	// check if new password same as previous
	if u.ID != 0 {
		var prevUser User
		Get(&prevUser, "id=?", u.ID)
		compareErr := verifyPassword(prevUser.Password, u.Password)
		if compareErr == nil {
			return eris.Wrapf(E_InvalidPassword, "sorry, you can't use the same password")
		}
	}

	return nil
}

// Validate user when saving from uadmin
func (u User) EmailExists() (e error) {
	var tmp User
	Get(&tmp, "email=?", strings.ToLower(u.Email))
	if tmp.ID > 0 {
		e = errors.New("Email is already Taken.")
	}
	return
}

// GetOTP !
func (u *User) GetOTP() string {
	return getOTP(u.OTPSeed, OTPDigits, OTPAlgorithm, OTPSkew, OTPPeriod)
}

// VerifyOTP !
func (u *User) VerifyOTP(pass string) bool {
	return verifyOTP(pass, u.OTPSeed, OTPDigits, OTPAlgorithm, OTPSkew, OTPPeriod)
}

func (u *User) VerifyOTPAtPasswordReset(pass string) bool {
	// Password reset link is valid for 24 hours
	if u.PasswordReset == nil || u.PasswordReset.Before(time.Now().AddDate(0, 0, -1)) {
		return false
	}
	return verifyOTPAt(pass, u.OTPSeed, OTPDigits, OTPAlgorithm, OTPSkew, OTPPeriod, *u.PasswordReset)
}

func (u *User) GeneratePasswordResetLink(r *http.Request, link string) (string, error) {
	// Check if the host name is in the allowed hosts list
	allowed := false
	var host string
	var allowedHost string
	var err error
	if host, _, err = net.SplitHostPort(GetHostName(r)); err != nil {
		host = r.Host
	}
	for _, v := range strings.Split(AllowedHosts, ",") {
		if allowedHost, _, err = net.SplitHostPort(v); err != nil {
			allowedHost = v
		}
		if allowedHost == host {
			allowed = true
			break
		}
	}
	host = GetHostName(r)
	if !allowed {
		Trail(CRITICAL, "Reset password request for host: (%s) which is not in AllowedHosts settings", host)
		return "", fmt.Errorf("Reset password request for host: (%s) which is not in AllowedHosts settings", host)
	}

	schema := GetSchema(r)
	if link == "" {
		link = "{SCHEMA}://{HOST}" + RootURL + "resetpassword?u={USER_ID}&key={OTP}"
	}
	link = strings.ReplaceAll(link, "{SCHEMA}", schema)
	link = strings.ReplaceAll(link, "{HOST}", host)
	link = strings.ReplaceAll(link, "{USER_ID}", fmt.Sprint(u.ID))
	link = strings.ReplaceAll(link, "{EMAIL}", fmt.Sprint(u.Email))
	link = strings.ReplaceAll(link, "{OTP}", u.GeneratePasswordResetOTP())

	return link, nil
}

func (u *User) GeneratePasswordResetOTP() string {
	// Set the date time for the password reset
	now := time.Now()
	u.PasswordReset = &now
	Save(u)

	return u.GetOTP()
}
