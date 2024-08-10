package controllers

import (
	"github.com/ankur12345678/models"
)

func HandleTransaction(targetUser *models.ZapPayDB, sourceUser *models.ZapPayDB, amount *int) string {
	if *amount <= 0 {
		return "Amount should be a Value Greater than 0."
	}
	if sourceUser.Amount < *amount {
		return "Insufficient Fund"
	}
	sourceUser.Amount = sourceUser.Amount - *amount
	targetUser.Amount = targetUser.Amount + *amount

	return ""
}
