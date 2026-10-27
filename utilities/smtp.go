package utilities

import (
	"aunefyren/poenskelisten/config"
	"aunefyren/poenskelisten/logger"
	"aunefyren/poenskelisten/models"
	"html"
	"strings"

	"github.com/go-mail/mail"
)

func SendSMTPVerificationEmail(user models.User) error {
	if strings.ToLower(config.ConfigFile.PoenskelistenEnvironment) == "test" {
		user.Email = &config.ConfigFile.PoenskelistenTestEmail
	}

	logger.Log.Debug("sending e-mail to: " + *user.Email + ".")

	m := mail.NewMessage()
	m.SetAddressHeader("From", config.ConfigFile.SMTPFrom, config.ConfigFile.PoenskelistenName)
	m.SetHeader("To", *user.Email)
	m.SetHeader("Subject", "Please verify your "+config.ConfigFile.PoenskelistenName+" account")
	m.SetBody("text/html", "Hello <b>"+user.FirstName+"</b>!<br><br>Someone created a "+htmlAppName()+" account using your e-mail. If this wasn't you, please ignore this e-mail.<br><br>To verify the new account, visit "+htmlAppName()+" and verify the account using this code: <b>"+*user.VerificationCode+"</b>.")

	d := mail.NewDialer(config.ConfigFile.SMTPHost, config.ConfigFile.SMTPPort, config.ConfigFile.SMTPUsername, config.ConfigFile.SMTPPassword)

	// Send the email
	err := d.DialAndSend(m)
	if err != nil {
		return err
	}

	return nil
}

func SendSMTPResetEmail(user models.User) error {
	if strings.ToLower(config.ConfigFile.PoenskelistenEnvironment) == "test" {
		user.Email = &config.ConfigFile.PoenskelistenTestEmail
	}

	logger.Log.Debug("sending e-mail to: " + *user.Email + ".")

	link := PasswordResetLink(*user.ResetCode)

	m := mail.NewMessage()
	m.SetAddressHeader("From", config.ConfigFile.SMTPFrom, config.ConfigFile.PoenskelistenName)
	m.SetHeader("To", *user.Email)
	m.SetHeader("Subject", config.ConfigFile.PoenskelistenName+" password reset request")
	m.SetBody("text/html", "Hello <b>"+user.FirstName+"</b>!<br><br>Someone attempted a password change on your "+htmlAppName()+" account. If this wasn't you, please ignore this e-mail.<br><br>To reset your password, visit "+htmlAppName()+" using <a href='"+link+"' target='_blank'>this link</a>.")

	d := mail.NewDialer(config.ConfigFile.SMTPHost, config.ConfigFile.SMTPPort, config.ConfigFile.SMTPUsername, config.ConfigFile.SMTPPassword)

	// Send the email
	err := d.DialAndSend(m)
	if err != nil {
		return err
	}

	return nil
}

func SendSMTPDeletedClaimedWish(user models.User, wish models.WishObject, wishlist models.WishlistUser) error {
	if strings.ToLower(config.ConfigFile.PoenskelistenEnvironment) == "test" {
		user.Email = &config.ConfigFile.PoenskelistenTestEmail
	}

	logger.Log.Debug("sending e-mail to: " + *user.Email + ".")

	link := config.ConfigFile.PoenskelistenExternalURL + "/wishlists/" + wishlist.ID.String()

	m := mail.NewMessage()
	m.SetAddressHeader("From", config.ConfigFile.SMTPFrom, config.ConfigFile.PoenskelistenName)
	m.SetHeader("To", *user.Email)
	m.SetHeader("Subject", "The wish you claimed on "+config.ConfigFile.PoenskelistenName+" has been deleted")
	m.SetBody("text/html", "Hello <b>"+user.FirstName+"</b>!<br><br>Someone who manages the wishlist '"+wishlist.Name+"' has deleted the wish '"+wish.Name+"'. We are telling you this because you claimed this gift. It is no longer visible on the wishlist.<br><br>To see the wishlist, visit "+htmlAppName()+" using <a href='"+link+"' target='_blank'>this link</a>.")

	d := mail.NewDialer(config.ConfigFile.SMTPHost, config.ConfigFile.SMTPPort, config.ConfigFile.SMTPUsername, config.ConfigFile.SMTPPassword)

	// Send the email
	err := d.DialAndSend(m)
	if err != nil {
		return err
	}

	return nil
}

// htmlAppName is the configured app name, escaped for an HTML mail body. The
// name is set by the admin and isn't run through ValidateTextCharacters.
func htmlAppName() string {
	return html.EscapeString(config.ConfigFile.PoenskelistenName)
}

// PasswordResetLink is the page a user opens to choose a new password with a
// reset code, whether the code was e-mailed or issued with -resetpassword.
func PasswordResetLink(resetCode string) string {
	return config.ConfigFile.PoenskelistenExternalURL + "/login?reset_code=" + resetCode
}
