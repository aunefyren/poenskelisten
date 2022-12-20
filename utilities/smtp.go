package utilities

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/models"
	"log"

	"github.com/go-mail/mail"
)

func SendSMTPVerificationEmail(user models.User) error {

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		return err
	}

	log.Println("Sending e-mail to: " + user.Email + ".")

	m := mail.NewMessage()
	m.SetAddressHeader("From", config.SMTPFrom, config.PoenskelistenName)
	m.SetHeader("To", user.Email)
	m.SetHeader("Subject", "Please verify your account")
	m.SetBody("text/html", "Hello <b>"+user.FirstName+"</b>!<br><br>Someone created a Pønskeliste account using your e-mail. If this wasn't you, please ignore this e-mail.<br><br>To verify the new account, visit Pønskelisten and verify the account using this code: <b>"+user.VerificationCode+"</b>.")

	d := mail.NewDialer(config.SMTPHost, config.SMTPPort, config.SMTPUsername, config.SMTPPassword)

	// Send the email
	err = d.DialAndSend(m)
	if err != nil {
		return err
	}

	return nil

}
