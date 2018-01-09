package email

import (
	"net/smtp"
	"strings"
)

type Emailc struct {
	User   string
	Passwd string
	Host   string
	To     string
}

type unencryptedAuth struct {
	smtp.Auth
}

func (a unencryptedAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	s := *server
	s.TLS = true
	return a.Auth.Start(&s)
}

func Newemail(host, user, passwd, to string) (s *Emailc) {
	s = &Emailc{
		User:   user,
		Host:   host,
		Passwd: passwd,
		To:     to,
	}
	return s
}

func (s *Emailc) SendToMail(title, body, mailtype string) error {
	hp := strings.Split(s.Host, ":")
	auth := unencryptedAuth {smtp.PlainAuth("", s.User, s.Passwd, hp[0])}
	var content_type string
	if mailtype == "html" {
		content_type = "Content-Type: text/" + mailtype + "; charset=UTF-8"
	} else {
		content_type = "Content-Type: text/plain" + "; charset=UTF-8"
	}
	//多次发送问题
	msg := []byte("To: " + s.To + "\r\nFrom: " + s.User + ">\r\nSubject: " + title + "\r\n" + content_type + "\r\n\r\n" + body)
	sendTo := strings.Split(s.To, ";")
	err := smtp.SendMail(s.Host, auth, s.User, sendTo, msg)
	if err != nil {
		return err
	}
	return err
}

/*
func main()  {

	email := Newemail("smtp.xxx.com:25", "service@xxx.com", "password", "user@xxx.com")

	email.SendToMail("test","hello this is a test mail","html")
}*/