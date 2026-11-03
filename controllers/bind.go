package controllers

import (
	"errors"
	"net/http"

	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/utilities"

	"github.com/gin-gonic/gin"
)

// bindJSON decodes the request body into request (a pointer to a struct) and
// applies the struct's sanitize tags, so trimming, lowercasing and text
// validation happen in one place instead of in each handler. On failure it
// has already written the response and aborted; the caller just returns.
//
// Every handler binds through here: TestEveryBoundRequestIsSanitized fails on
// a direct ShouldBindJSON call or on a request field without a sanitize tag.
func bindJSON(context *gin.Context, request any) bool {
	if err := context.ShouldBindJSON(request); err != nil {
		logger.Log.Error("Failed to parse request. Error: " + err.Error())
		context.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse request."})
		context.Abort()
		return false
	}

	if err := utilities.SanitizeRequest(request); err != nil {
		var textErr *utilities.TextValidationError
		if errors.As(err, &textErr) {
			logger.Log.Error("Request field " + textErr.Field + " failed text validation.")
			context.JSON(http.StatusBadRequest, gin.H{"error": textErr.Requirements})
			context.Abort()
			return false
		}
		logger.Log.Error("Failed to sanitize request. Error: " + err.Error())
		context.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request."})
		context.Abort()
		return false
	}

	return true
}
