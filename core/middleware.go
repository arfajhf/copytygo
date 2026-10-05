package core

type Middleware func(next Handler) Handler

func applyMiddleware(
	handler Handler,
	middleware []Middleware,
) Handler {
	for i := len(middleware) - 1; i >= 0; i-- {
		handler = middleware[i](handler)
	}
	return handler

}
