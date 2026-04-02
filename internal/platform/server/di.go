package server

import (
	"github.com/samber/do/v2"
)

// Package installs the server dependencies into the injector.
func Package(i do.Injector) {
	do.Package(do.Lazy(New))(i)
}
