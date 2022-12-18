package utilities

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/models"
	"log"
	"net/smtp"
	"strconv"
)

func SendSMTPVerificationEmail(user models.User) error {

	// Get configuration
	config, err := config.GetConfig()
	if err != nil {
		return err
	}

	log.Println("Sending e-mail to user " + user.FirstName + " " + user.LastName + ".")

	toEmailAddress := user.Email
	to := []string{toEmailAddress}

	auth := smtp.PlainAuth("", config.SMTPUsername, config.SMTPPassword, config.SMTPHost)

	subject := "Subject: This is the subject of the mail\n"
	body := "Verify your Pønskeliste account"
	message := []byte(subject + body)

	smt_port_int := strconv.Itoa(config.SMTPPort)
	host := "smtp.gmail.com"
	address := host + ":" + smt_port_int

	err = smtp.SendMail(address, auth, config.SMTPFrom, to, message)
	if err != nil {
		return err
	}

	return nil
}
