package expense

import (
	"github.com/ArtroxGabriel/accounter/internal/category"
	"github.com/samber/do/v2"
)

// Package installs the domain dependencies into the injector.
func Package(i do.Injector) {
	do.Package(
		do.Lazy(NewSQLiteRepository),
		do.Lazy(NewService),
		do.Lazy(NewHandler),
		do.Lazy(func(i do.Injector) (CategoryChecker, error) {
			return do.MustInvoke[category.Service](i), nil
		}),
	)(i)
}
