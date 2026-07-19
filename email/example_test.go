package email_test

import (
	"context"
	"fmt"

	"github.com/hallode/golib/email"
)

func ExampleClient_SendContext() {
	client, err := email.New(email.Config{
		Host:     "smtp.example.com",
		Port:     587,
		Username: "app@example.com",
		Password: "secret",
		FromName: "My App",
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	msg := client.NewMessage().
		To("user@example.com").
		Subject("Welcome").
		Text("Plain body").
		HTML("<p>HTML body</p>")

	if err := client.SendContext(context.Background(), msg); err != nil {
		fmt.Println(err)
	}
}
