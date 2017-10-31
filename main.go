// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"database/sql"

	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/NoahShen/go-simsimi"
	 _ "github.com/go-sql-driver/mysql"
)

var bot *linebot.Client
var session *simsimi.SimSimiSession
func main() {
	var err error
	session, _ = simsimi.CreateSimSimiSession("Wallte")
	bot, err = linebot.New(os.Getenv("CHANNEL_SECRET"), os.Getenv("CHANNEL_TOKEN"))
	log.Println("Bot:", bot, " err:", err)
	http.HandleFunc("/callback", callbackHandler)
	port := os.Getenv("PORT")
	addr := fmt.Sprintf(":%s", port)
	http.ListenAndServe(addr, nil)
}

func connect() (*sql.DB, error) {
	db, err := sql.Open("mysql", os.Getenv("DB_CONNECT"))
	if err != nil {
		fmt.Println(err.Error())
		return nil, err
	}
	return db, nil
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {
	events, err := bot.ParseRequest(r)

	if err != nil {
		if err == linebot.ErrInvalidSignature {
			w.WriteHeader(400)
		} else {
			w.WriteHeader(500)
		}
		return
	}

	for _, event := range events {
		if event.Type == linebot.EventTypeMessage {
			switch message := event.Message.(type) {
			case *linebot.TextMessage:
				/*if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message.ID+":"+message.Text+" OK!")).Do(); err != nil {
					log.Print(err)
				}*/
				imageURL := "https://drive.google.com/file/d/0Bx6cTEFypiiNaHVTcXV5VkFpbFE/view?usp=sharing"
				template := linebot.NewButtonsTemplate(
					imageURL, "My button sample"+message.Text, "Hello, my button",
					linebot.NewURITemplateAction("Go to line.me", "https://line.me"),
					linebot.NewPostbackTemplateAction("Say hello1", "hello こんにちは", ""),
					linebot.NewPostbackTemplateAction("言 hello2", "hello こんにちは", "hello こんにちは"),
					linebot.NewMessageTemplateAction("Say message", "Rice=米"),
				)
				if _, err := bot.ReplyMessage(
					event.ReplyToken,
					linebot.NewTemplateMessage("Buttons alt text", template),
				).Do(); err != nil {
					return
				}
				
				responseText, _ := session.Talk(message.Text)
				if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(responseText)).Do(); err != nil {
					log.Print(err)
				}

				
			}
		}else if event.Type == linebot.EventTypePostback{
			/*if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("iniPostback")).Do(); err != nil {
					log.Print(err)
				}*/
			
			imageURL := "https://github.com/AdityaMili95/Wallte/blob/master/README/qI5Ujdy9n1.png"
			/*template := linebot.NewCarouselTemplate(
				linebot.NewCarouselColumn(
					imageURL, "hoge", "fuga",
					linebot.NewURITemplateAction("Go to line.me", "https://line.me"),
					linebot.NewPostbackTemplateAction("Say hello1", "hello こんにちは", ""),
				),
				linebot.NewCarouselColumn(
					imageURL, "hoge", "fuga",
					linebot.NewPostbackTemplateAction("言 hello2", "hello こんにちは", "hello こんにちは"),
					linebot.NewMessageTemplateAction("Say message", "Rice=米"),
				),
			)
			if _, err := bot.ReplyMessage(
				event.ReplyToken,
				linebot.NewTemplateMessage("Carousel alt text", template),
			).Do(); err != nil {
				log.Print(err)
			}*/

			profile, err := bot.GetProfile(event.Source.UserID).Do()
			if(err!=nil){
				log.Print(err)
				return;
			}
			template := linebot.NewImageCarouselTemplate(
				linebot.NewImageCarouselColumn(
					imageURL,
					linebot.NewURITemplateAction("Go to LINE", "https://line.me"),
				),
				linebot.NewImageCarouselColumn(
					imageURL,
					linebot.NewPostbackTemplateAction("Say hello1", "hello こんにちは", "Hello"+profile.UserID),
				),
				linebot.NewImageCarouselColumn(
					imageURL,
					linebot.NewMessageTemplateAction("Say message", "Rice=米"),
				),
				linebot.NewImageCarouselColumn(
					imageURL,
					linebot.NewDatetimePickerTemplateAction("datetime", "DATETIME", "datetime", "", "", ""),
				),
			)
			if _, err := bot.ReplyMessage(
				event.ReplyToken,
				linebot.NewTemplateMessage("Image carousel alt text", template),
			).Do(); err != nil {
				log.Print(err)
			}
			
			/*template := linebot.NewButtonsTemplate(
				"", "", "Select date / time !",
				linebot.NewDatetimePickerTemplateAction("date", "DATE", "date", "", "", ""),
				linebot.NewDatetimePickerTemplateAction("time", "TIME", "time", "", "", ""),
				linebot.NewDatetimePickerTemplateAction("datetime", "DATETIME", "datetime", "", "", ""),
			)
			if _, err := bot.ReplyMessage(
				event.ReplyToken,
				linebot.NewTemplateMessage("Datetime pickers alt text", template),
			).Do(); err != nil {
				log.Print(err)
			}*/

			
		}
		
	}
}
