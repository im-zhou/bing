package main

import (
	"crypto/tls"
	"fmt"
	"mime"
	"net"
	"net/mail"
	"net/smtp"
	"strings"
	"time"
)

func notify(cfg Config, subject, body string) error {
	c := cfg.SMTP
	if c.Address == "" {
		return nil
	}
	host, port, err := net.SplitHostPort(c.Address)
	if err != nil {
		return err
	}
	from, err := mail.ParseAddress(c.From)
	if err != nil {
		return err
	}
	to, err := mail.ParseAddress(c.To)
	if err != nil {
		return err
	}
	dialer := &net.Dialer{Timeout: 10 * time.Second}
	var conn net.Conn
	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}
	if port == "465" {
		conn, err = tls.DialWithDialer(dialer, "tcp", c.Address, tlsCfg)
	} else {
		conn, err = dialer.Dial("tcp", c.Address)
	}
	if err != nil {
		return err
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(15 * time.Second))
	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return err
	}
	defer client.Close()
	if port != "465" {
		if ok, _ := client.Extension("STARTTLS"); !ok {
			return fmt.Errorf("SMTP server must support STARTTLS")
		}
		if err = client.StartTLS(tlsCfg); err != nil {
			return err
		}
	}
	if c.Username != "" {
		if err = client.Auth(smtp.PlainAuth("", c.Username, c.Password, host)); err != nil {
			return err
		}
	}
	if err = client.Mail(from.Address); err != nil {
		return err
	}
	if err = client.Rcpt(to.Address); err != nil {
		return err
	}
	w, err := client.Data()
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(w, "From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=utf-8\r\n\r\n%s\r\n", from.String(), to.String(), mime.QEncoding.Encode("utf-8", subject), strings.ReplaceAll(body, "\n", "\r\n"))
	if err != nil {
		return err
	}
	if err = w.Close(); err != nil {
		return err
	}
	return client.Quit()
}
