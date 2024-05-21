package views

/// The base where all the functions for templating will be declared
type BaseTemplateContext struct {
	// tranlation function
	Tf func(path string, lang string, term string, args ...interface{}) string
	//Returns a CSRF value for forms that needs it
	CSRF func() string
	// Return an unique generared value for each template render
	Timestamp func() int64
}
