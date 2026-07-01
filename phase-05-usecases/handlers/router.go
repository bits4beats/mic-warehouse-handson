package handlers

import (
	"github.com/labstack/echo/v4"
)

// Router registers the Article routes on the Echo instance.
type Router struct {
	articleHandler *ArticleHandler
}

func NewRouter(h *ArticleHandler) *Router {
	return &Router{articleHandler: h}
}

func (r *Router) Register(e *echo.Echo) {
	// GIVEN — the worked-example route.
	e.POST("/articles", r.articleHandler.CreateArticle)

	// TODO Slice 1: expose GetArticle. Uncomment once your handler is done.
	// e.GET("/articles/:id", r.articleHandler.GetArticle)

	// TODO Slice 2: expose ChangeArticlePrice. Uncomment once your handler is done.
	// e.PUT("/articles/:id/price", r.articleHandler.ChangeArticlePrice)
}
