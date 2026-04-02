package database

import (
	"github.com/samber/do/v2"
)

func Package(i do.Injector) {
	do.Package(do.Lazy(New))(i)
}
