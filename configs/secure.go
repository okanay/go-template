package configs

import "github.com/gin-contrib/secure"

var SecureConfig = secure.New(secure.Config{
	FrameDeny:             true,
	ContentTypeNosniff:    true,
	BrowserXssFilter:      true,
	ContentSecurityPolicy: "default-src 'self'",
})
