package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/line/line-bot-sdk-go/v8/linebot/messaging_api"
	"github.com/line/line-bot-sdk-go/v8/linebot/webhook"
)

func init() {
	_ = godotenv.Load()
}

func main() {

	handler, err := webhook.NewWebhookHandler(
		os.Getenv("LINE_CHANNEL_SECRET"),
	)
	if err != nil {
		log.Fatal(err)
	}
	bot, err := messaging_api.NewMessagingApiAPI(os.Getenv("LINE_CHANNEL_TOKEN"))
	if err != nil {
		log.Print(err)
		return
	}

	http.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hello, world!"))
	})

	http.HandleFunc("POST /push", func(w http.ResponseWriter, r *http.Request) {
		text := r.URL.Query().Get("text")
		if text == "" {
			http.Error(w, "text not set", http.StatusBadRequest)
			return
		}
		groupID := os.Getenv("LINE_GROUP_ID")
		if groupID == "" {
			http.Error(w, "LINE_GROUP_ID not set", http.StatusInternalServerError)
			return
		}

		req := messaging_api.PushMessageRequest{
			To: groupID,
			Messages: []messaging_api.MessageInterface{
				&messaging_api.TextMessage{
					Message: messaging_api.Message{
						Type: "text",
						QuickReply: messaging_api.QuickReply{
							Items: []messaging_api.QuickReplyItem{{
								ImageUrl: "",
								Type:     "",
							}},
						},
						Sender: messaging_api.Sender{
							Name:    "",
							IconUrl: "",
						},
					},
					Text:       text,
					QuickReply: nil,
					Sender:     nil,
					Emojis:     nil,
					QuoteToken: "",
				},
			},
			NotificationDisabled:   false,
			CustomAggregationUnits: nil,
		}
		if _, err := bot.PushMessage(&req, ""); err != nil {
			log.Println("push error:", err)
			http.Error(w, "push failed", http.StatusInternalServerError)
			return
		}
		fmt.Fprintf(w, "pushed: %s\n", text)
	})

	file, errOpen := os.OpenFile("events.jsonl", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if errOpen != nil {
		log.Fatalf("open file error: %v", errOpen)
	}
	defer file.Close()
	handler.HandleEvents(func(req *webhook.CallbackRequest, r *http.Request) {
		log.Println("Handling events...")
		for _, event := range req.Events {
			if b, err := json.Marshal(event); err == nil {
				if _, err := file.Write(append(b, '\n')); err != nil {
					log.Println("write error:", err)
				}
			} else {
				log.Println("marshal error:", err)
			}
			switch e := event.(type) {
			case webhook.MessageEvent:
				switch message := e.Message.(type) {
				case webhook.TextMessageContent:
					_, err = bot.ReplyMessage(
						&messaging_api.ReplyMessageRequest{
							ReplyToken: e.ReplyToken,
							Messages: []messaging_api.MessageInterface{
								&messaging_api.TextMessage{
									Text: message.Text,
								},
							},
						},
					)
					if err != nil {
						log.Print(err)
					}
				}
			}
		}
	})
	http.Handle("POST /callback", handler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "5000"
	}
	fmt.Println("http://localhost:" + port + "/")
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}
