package category

import (
	"github.com/samber/do/v2"
)

// Package installs the domain dependencies into the injector.
func Package(i do.Injector) {
	do.Package(
		do.Lazy(NewSQLiteRepository),
		do.Lazy(NewService),
		do.Lazy(NewHandler),
	)(i)
}
