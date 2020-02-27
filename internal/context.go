package internal

import (
    "fmt"
    "context"
    "github.com/labstack/echo/v4"
)

type GlobalContext struct {
    echo.Context
    ctx    context.Context
}

func RouterContextToContextMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		ctx := context.WithValue(c.Request().Context(), "RouterContextKey", c)
		c.SetRequest(c.Request().WithContext(ctx))
		return next(&GlobalContext{c, ctx})
	}
}

func RouterContextFromContext(ctx context.Context) (*echo.Context, error) {
	routerContext := ctx.Value("RouterContextKey")
	if routerContext == nil {
		err := fmt.Errorf("could not retrieve router Context")
		return nil, err
	}

	ec, ok := routerContext.(*echo.Context)
	if !ok {
		err := fmt.Errorf("router Context has wrong type")
		return nil, err
	}
	return ec, nil
}

