package controllers

import (
	"net/http"
	"poenskelisten/database"
	"poenskelisten/models"

	"github.com/gin-gonic/gin"
)

func RegisterInvite(context *gin.Context) {
	var invite models.Invite
	if err := context.ShouldBindJSON(&invite); err != nil {
		context.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		context.Abort()
		return
	}

	record := database.Instance.Create(&invite)

	if record.Error != nil {
		context.JSON(http.StatusInternalServerError, gin.H{"error": record.Error.Error()})
		context.Abort()
		return
	}

	context.JSON(http.StatusCreated, gin.H{"invite_code": invite.InviteCode})
}
