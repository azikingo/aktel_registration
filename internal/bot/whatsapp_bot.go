package bot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
)

type WPBot struct {
	token string
}

func NewWPBot() (*WPBot, error) {
	token := os.Getenv("WHATSAPP_TOKEN")
	if token == "" {
		return nil, errors.New("missing WHATSAPP_TOKEN in environment")
	}

	return &WPBot{token: token}, nil
}

type MessagePayload struct {
	MessagingProduct string `json:"messaging_product"`
	To               string `json:"to"`
	Type             string `json:"type"`
	Text             struct {
		Body string `json:"body"`
	} `json:"text"`
}

func (b *WPBot) SendMessage(phoneNumber, message string) error {
	payload := MessagePayload{
		MessagingProduct: "whatsapp",
		To:               phoneNumber,
		Type:             "text",
	}
	payload.Text.Body = message

	jsonPayload, _ := json.Marshal(payload)

	req, err := http.NewRequest("POST", "https://graph.facebook.com/v23.0/701322406398011/messages", bytes.NewBuffer(jsonPayload))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+b.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 300 {
		return fmt.Errorf("failed: %s", resp.Status)
	}

	return nil
}
