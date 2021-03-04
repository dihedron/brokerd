package openapi

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// AddAPIHandlers adds the API handlers to the given router.
func AddAPIHandlers(router *gin.Engine) {
	for _, route := range routes {
		switch route.Method {
		case http.MethodGet:
			router.GET(route.Pattern, route.HandlerFunc)
		case http.MethodPost:
			router.POST(route.Pattern, route.HandlerFunc)
		case http.MethodPut:
			router.PUT(route.Pattern, route.HandlerFunc)
		case http.MethodDelete:
			router.DELETE(route.Pattern, route.HandlerFunc)
		}
	}
}
