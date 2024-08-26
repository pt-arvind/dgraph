package validator

import (
	"github.com/dgraph-io/gqlparser/v2/ast"
	. "github.com/dgraph-io/gqlparser/v2/validator"
)

func init() {
	AddRule("KnownDirectives", func(observers *Events, addError AddErrFunc) {
		observers.OnDirective(func(walker *Walker, directive *ast.Directive) {
			if directive.Definition == nil {
				addError(
					Message(`Unknown directive "%s".`, directive.Name),
					At(directive.Position),
				)
				return
			}

			for _, loc := range directive.Definition.Locations {
				if loc == directive.Location {
					return
				}
			}

			addError(
				Message(`Directive "%s" may not be used on %s.`, directive.Name, directive.Location),
				At(directive.Position),
			)
		})
	})
}
