package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/ankur12345678/constants"
	"github.com/ankur12345678/controllers"
	"github.com/ankur12345678/models"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Payload struct {
	Email  string `json:"email"`
	Amount int    `json:"amount"`
}

func main() {
	////////////////////////////////////////get the db instance/////////////////////////////////////////////////////////
	dsn := "host=localhost user=ankur password=ankur dbname=userwallet port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Print("error occured")

	}
	db.AutoMigrate(&models.ZapPayDB{}, &models.Transaction{})
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////////

	///////////////////////////////////////////	ALL ROUTING /////////////////////////////////////////////////////////////
	r := gin.Default()
	r.POST("/login", func(c *gin.Context) {
		var receivedData Payload
		c.BindJSON(&receivedData)

		if receivedData.Email == "" {
			c.JSON(200, gin.H{
				"message": "Please provide an email id",
			})
			return
		}

		var user models.ZapPayDB
		result := db.Unscoped().Where("email = ?", receivedData.Email).First(&user)

		if result.Error != nil && result.Error != gorm.ErrRecordNotFound {
			c.JSON(500, gin.H{
				"message": "Error checking user record",
			})
			return
		}

		if result.RowsAffected > 0 {
			// User exists, including soft-deleted records
			if user.DeletedAt.Valid {
				// Recover the soft-deleted user

				if err := db.Unscoped().Model(&models.ZapPayDB{}).Where("email = ?", receivedData.Email).Update("deleted_at", nil).Error; err != nil {
					c.JSON(500, gin.H{
						"message": "Error recovering user",
					})
					return
				}
			}
		} else {
			// User does not exist, create a new user
			newUser := models.ZapPayDB{
				Email:  receivedData.Email,
				Amount: 200,
			}
			if err := db.Create(&newUser).Error; err != nil {
				c.JSON(500, gin.H{
					"message": "Error creating new user",
				})
				return
			}
		}

		jwt_token, err := GenerateJWT(constants.SECRET_KEY, receivedData.Email)
		if err == nil {
			c.JSON(200, gin.H{
				"access_token": jwt_token,
				"instruction":  "Put it in Authorization Header at the FetchBalance page to see Balance",
			})
		} else {
			c.JSON(200, gin.H{
				"error": "error generating jwt",
			})
		}

	})

	r.POST("/fetchBalance", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(200, gin.H{
				"error": "No auth token found",
			})
			return
		}
		splitAuthHeader := strings.Split(authHeader, " ")
		authToken := splitAuthHeader[1]
		if authToken == "" {
			c.JSON(200, gin.H{
				"error": "No auth token found",
			})
			return
		}

		//verify jwt
		claims := jwt.MapClaims{}
		parsedToken, _ := jwt.ParseWithClaims(authToken, &claims, func(token *jwt.Token) (interface{}, error) {

			return []byte(constants.SECRET_KEY), nil // Use the same secret key used for signing
		})
		//error handling
		if !parsedToken.Valid {
			c.JSON(200, gin.H{
				"error": "Inavlid Auth token",
			})
			return

		}
		targetEmail := claims["email"]
		//find wallet balance for targetEmail from db
		var targetUser models.ZapPayDB
		result := db.Where("email = ?", targetEmail).First(&targetUser)
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(200, gin.H{
				"Message": "No such user exists in record, Please visit /login",
			})
			return
		}
		c.JSON(200, gin.H{
			"Email":  targetEmail,
			"Amount": targetUser.Amount,
		})

	})

	r.POST("/Transaction", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(200, gin.H{
				"error": "No auth token found",
			})
			return
		}
		splitAuthHeader := strings.Split(authHeader, " ")
		authToken := splitAuthHeader[1]
		if authToken == "" {
			c.JSON(200, gin.H{
				"error": "No auth token found",
			})
			return
		}

		//verify jwt
		claims := jwt.MapClaims{}
		parsedToken, _ := jwt.ParseWithClaims(authToken, &claims, func(token *jwt.Token) (interface{}, error) {

			return []byte(constants.SECRET_KEY), nil // Use the same secret key used for signing
		})
		//error handling
		if !parsedToken.Valid {
			c.JSON(200, gin.H{
				"error": "Inavlid Auth token",
			})
			return

		}
		sourceEmail := claims["email"]
		//find wallet balance for targetEmail from db
		var receivedData Payload
		c.BindJSON(&receivedData)
		targetEmail := receivedData.Email
		amountOfTransaction := receivedData.Amount

		//create a go routine to handle transaction
		var targetUser models.ZapPayDB
		resultTarget := db.Where("email = ?", targetEmail).First(&targetUser)
		if resultTarget.Error == gorm.ErrRecordNotFound {
			c.JSON(200, gin.H{
				"message": "No such recipient exists!",
			})
			return
		}
		var sourceUser models.ZapPayDB
		resultSource := db.Where("email = ?", sourceEmail).First(&sourceUser)
		//check if source user is there or not(might be deleted)
		if resultSource.Error == gorm.ErrRecordNotFound {
			c.JSON(200, gin.H{
				"message": "No such sender exists!",
			})
			return
		}

		if sourceEmail == targetEmail {
			c.JSON(200, gin.H{
				"message": "Sender and Recipient Cannot be same!",
			})
			return
		}

		resp := controllers.HandleTransaction(&targetUser, &sourceUser, &amountOfTransaction)
		//update the db

		db.Model(&models.ZapPayDB{}).Where("email = ?", sourceEmail).Updates(sourceUser)
		db.Model(&models.ZapPayDB{}).Where("email = ?", targetEmail).Updates(targetUser)

		//create a uuid for the transaction and add to transaction table
		tid := uuid.New()

		if resp == "" {
			newTransactionRecord := models.Transaction{
				Tid:           tid.String(),
				SenderEmail:   sourceUser.Email,
				ReceiverEmail: targetUser.Email,
				Amount:        amountOfTransaction,
				State:         "Success",
			}
			result := db.Create(&newTransactionRecord)
			fmt.Print(result.Error)
			c.JSON(200, gin.H{
				"message":      "Transaction Success",
				"tid":          tid.String(),
				"your balance": sourceUser.Amount,
			})
		} else {
			newTransactionRecord := models.Transaction{
				Tid:           tid.String(),
				SenderEmail:   sourceUser.Email,
				ReceiverEmail: targetUser.Email,
				Amount:        amountOfTransaction,
				State:         "Failed",
			}
			db.Create(&newTransactionRecord)

			c.JSON(200, gin.H{
				"message":      "Transaction failed",
				"your balance": resp,
			})
		}

	})

	r.DELETE("/deleteAccount", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(200, gin.H{
				"error": "No auth token found",
			})
			return
		}
		splitAuthHeader := strings.Split(authHeader, " ")
		authToken := splitAuthHeader[1]
		if authToken == "" {
			c.JSON(200, gin.H{
				"error": "No auth token found",
			})
			return
		}

		//verify jwt
		claims := jwt.MapClaims{}
		parsedToken, _ := jwt.ParseWithClaims(authToken, &claims, func(token *jwt.Token) (interface{}, error) {

			return []byte(constants.SECRET_KEY), nil // Use the same secret key used for signing
		})
		//error handling
		if !parsedToken.Valid {
			c.JSON(200, gin.H{
				"error": "Inavlid Auth token",
			})
			return

		}
		targetEmail := claims["email"]
		//delete entry for targetEmail from db

		result := db.Where("email = ?", targetEmail).Delete(&models.ZapPayDB{})
		if result.RowsAffected == 0 {
			c.JSON(200, gin.H{
				"Message": "No such User's Record present",
			})
			return
		} else if result.Error == nil {
			c.JSON(200, gin.H{
				"Message": "User Successfully Deleted!",
			})
			return
		}
		c.JSON(200, gin.H{
			"Message": "Error Deleting User",
		})

	})

	r.GET("/MyTransactions", func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(200, gin.H{
				"error": "No auth token found",
			})
			return
		}
		splitAuthHeader := strings.Split(authHeader, " ")
		authToken := splitAuthHeader[1]
		if authToken == "" {
			c.JSON(200, gin.H{
				"error": "No auth token found",
			})
			return
		}

		//verify jwt
		claims := jwt.MapClaims{}
		parsedToken, _ := jwt.ParseWithClaims(authToken, &claims, func(token *jwt.Token) (interface{}, error) {

			return []byte(constants.SECRET_KEY), nil // Use the same secret key used for signing
		})
		//error handling
		if !parsedToken.Valid {
			c.JSON(200, gin.H{
				"error": "Inavlid Auth token",
			})
			return

		}
		targetEmail := claims["email"]
		//find wallet balance for targetEmail from db
		var targetUser models.ZapPayDB
		result := db.Where("email = ?", targetEmail).First(&targetUser)
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(200, gin.H{
				"Message": "No such user exists in record, Please visit /login endpoint",
			})
			return
		}

		//two type of transactions
		//send transactions --> where we are the sender
		//received transactions --> where we are the receiver

		var sendTransactions []models.Transaction
		var receivedTransactions []models.Transaction

		db.Where("sender_email = ?", targetEmail).Find(&sendTransactions)
		db.Where("receiver_email = ?", targetEmail).Find(&receivedTransactions)

		c.JSON(200, gin.H{
			"Message":               "Transaction Details",
			"Send Transactions":     sendTransactions,
			"Received Transactions": receivedTransactions,
		})

	})

	r.Run(":8000")
	///////////////////////////////////////////////////////////////////////////////////////////////////////////////////

}

func GenerateJWT(secret string, email string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256,
		jwt.MapClaims{
			"email": email,
			"exp":   time.Now().Add(time.Second * 500).Unix(),
		})
	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", err
	}

	return tokenString, nil
}
