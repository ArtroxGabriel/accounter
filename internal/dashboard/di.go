package dashboard

import (
	"github.com/samber/do/v2"
)

// Package installs the dashboard dependencies into the injector.
func Package(i do.Injector) {
	do.Package(
		do.Lazy(NewHandler),
	)(i)
}
