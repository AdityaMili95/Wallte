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
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
	//"database/sql"

	"github.com/NoahShen/go-simsimi"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/xuyu/goredis"
)

var bot *linebot.Client
var session *simsimi.SimSimiSession
var Redis *goredis.Redis

const (
	ADD_EXPENSE = "add-expense"
	ADD_INCOME  = "add-income"
	PLAN        = "plan"
	USER        = 1
	ROOM        = 2
	GROUP       = 3
)

type DataWallet struct {
	Data Wallet
}

type Wallet struct {
	UserInfo     Info
	RoomInfo     Info
	GroupInfo    Info
	Money        int
	Income       map[int]map[int][]TransactionInfo
	Expense      map[int]map[int][]TransactionInfo
	Plan_Income  map[int]map[int][]TransactionInfo
	Plan_Expense map[int]map[int][]TransactionInfo
	Last_Action  LastAction
}

type LastAction struct {
	Status      bool
	Keyword     string
	Description string
	Price       int
	Category    string
}

type Info struct {
	ID string
}

type TransactionInfo struct {
	Created_by   string
	Price        int
	Created_date time.Time
	Planned_date time.Time
}

func main() {
	var err error

	connectRedis()
	session, _ = simsimi.CreateSimSimiSession("Wallte")
	bot, err = linebot.New(os.Getenv("CHANNEL_SECRET"), os.Getenv("CHANNEL_TOKEN"))
	log.Println("Bot:", bot, " err:", err)
	http.HandleFunc("/callback", callbackHandler)
	port := os.Getenv("PORT")
	addr := fmt.Sprintf(":%s", port)
	http.ListenAndServe(addr, nil)
}

func connectRedis() {
	var err error
	Redis, err = goredis.DialURL(os.Getenv("REDIS_CONNECT"))
	if err != nil {
		log.Println(err.Error())
		return
	}
}

func connectDB() (*sqlx.DB, error) {
	db, err := sqlx.Open("mysql", os.Getenv("DB_CONNECT"))
	if err != nil {
		log.Println(err.Error())
		return nil, err
	}
	return db, nil
}

func GetRedis(key string) string {
	data, err := Redis.Get(key)

	if err != nil {
		log.Println(err)
		return ""
	}

	return string(data)
}

func SetRedis(key string, value string) {

	err := Redis.Set(key, value, 0, 0, false, false)
	if err != nil {
		log.Printf("%#v\n", err)
		return
	}

}

func executeQuery(query string) *sqlx.Rows {
	db, err := connectDB()
	if err != nil {
		log.Println(err.Error())
		return nil
	}
	defer db.Close()
	rows, err := db.Queryx(query)
	if err != nil {
		log.Println(err.Error())
		return nil
	}
	return rows
}

func executeInsert(json string, userID string, roomID string, groupID string) {
	db, err := connectDB()

	if err != nil {
		log.Println(err.Error())
		return
	}
	defer db.Close()
	tx := db.MustBegin()
	tx.MustExec("INSERT INTO wallte_data(user_id,room_id,group_id,JSON) VALUES('$1','$2','$3','$4')", userID, roomID, groupID, json)
	tx.Commit()
}

func executeUpdate(json string, userID string, roomID string, groupID string) {
	db, err := connectDB()
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer db.Close()
	tx := db.MustBegin()
	tx.MustExec("update wallte_data set JSON='$1' where user_id='$2' and room_id='$3' and group_id='$4'", json, userID, roomID, groupID)
	tx.Commit()
}

func Marshal(data interface{}) (string, error) {

	res, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

func updateData(data *DataWallet, exist bool, userID string, roomID string, groupID string) {
	res, err := Marshal(data)
	if err != nil || res == "" {
		return
	}

	redisKey := userID
	if roomID != "" {
		redisKey = roomID
	}

	if groupID != "" {
		redisKey = groupID
	}

	SetRedis(redisKey, res)
	if !exist {
		executeUpdate(res, userID, roomID, groupID)
		return
	}

	executeInsert(res, userID, roomID, groupID)
}

func prepareUpdateData(data *DataWallet, exist bool, userID string, roomID string, groupID string, msgType int) {
	if msgType == USER {
		updateData(data, exist, userID, "", "")
	} else if msgType == ROOM {
		updateData(data, exist, "", roomID, "")
	} else if msgType == GROUP {
		updateData(data, exist, "", "", groupID)
	}
}

func getUserData(ID string) (*DataWallet, bool) {
	var data *DataWallet
	res := GetRedis(ID)
	if res != "" && res != "nil" {
		err := json.Unmarshal([]byte(res), &data)
		if err == nil {
			return data, true
		}
	}

	query := fmt.Sprintf("SELECT JSON FROM wallte_data WHERE user_id='%s' OR group_id='%s' OR room_id='%s' LIMIT 1", ID, ID, ID)
	rows := executeQuery(query)

	if rows == nil {
		return nil, false
	}

	defer rows.Close()

	if rows.Next() {
		var jsonString string
		err := rows.Scan(&jsonString)
		if err != nil {
			return nil, false
		}

		err = json.Unmarshal([]byte(jsonString), &data)
		if err == nil {
			return data, true
		}
	}

	return nil, false
}

func initDataWallet(userID string, roomID string, groupID string, msgType int) *DataWallet {

	var userInfo, roomInfo, groupInfo Info

	if msgType == USER {
		userInfo = Info{
			ID: userID,
		}
	} else if msgType == ROOM {
		roomInfo = Info{
			ID: userID,
		}
	} else if msgType == GROUP {
		groupInfo = Info{
			ID: userID,
		}
	}

	return &DataWallet{
		Data: Wallet{
			UserInfo:  userInfo,
			RoomInfo:  roomInfo,
			GroupInfo: groupInfo,
		},
	}
}

func FetchDataSource(event *linebot.Event) (string, string, string, *DataWallet, bool, int) {
	userID := ""
	roomID := ""
	groupID := ""

	var data *DataWallet
	var exist bool
	var msgType int

	source := event.Source
	switch source.Type {
	case linebot.EventSourceTypeUser:
		userID = source.UserID
		data, exist = getUserData(userID)
		msgType = USER
	case linebot.EventSourceTypeGroup:
		roomID = source.RoomID
		userID = source.UserID
		data, exist = getUserData(roomID)
		msgType = ROOM
	case linebot.EventSourceTypeRoom:
		groupID = source.GroupID
		userID = source.UserID
		data, exist = getUserData(groupID)
		msgType = GROUP
	}

	return userID, roomID, groupID, data, exist, msgType
}

func handleAddExpense(splitted []string, event *linebot.Event, exist bool, userID string, roomID string, groupID string, data *DataWallet, msgType int) {
	imageURL := "https://github.com/AdityaMili95/Wallte/raw/master/README/qI5Ujdy9n1.png"
	/*if _, err := bot.ReplyMessage(
		event.ReplyToken,
		linebot.NewTextMessage(message.ID+":"+message.Text+" OK!"),
	).Do(); err != nil {
		return
	}*/

	lenSplitted := len(splitted)

	if !exist {
		data = initDataWallet(userID, roomID, groupID, msgType)
		prepareUpdateData(data, exist, userID, roomID, groupID, msgType)

	}

	if lenSplitted == 2 {

		template := linebot.NewImageCarouselTemplate(
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("Food", "/add-expense/food", ""),
			),
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("Transport", "/add-expense/transport", ""),
			),
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("Social", "/add-expense/social", ""),
			),
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("Life", "/add-expense/life", ""),
			),
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("Other", "/add-expense/other", ""),
			),
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewURITemplateAction("Shop Now", "https://tokopedia.com/elefashionshop"),
			),
		)
		if _, err := bot.ReplyMessage(
			event.ReplyToken,
			linebot.NewTemplateMessage("Select Expense Category!!", template),
		).Do(); err != nil {
			log.Print(err)
		}

	} /* else {
		if _, err := bot.ReplyMessage(
			event.ReplyToken,
			linebot.NewTextMessage(" OK!"),
		).Do(); err != nil {
			return
		}
	}*/
}

func handleTextMessage(event *linebot.Event, message *linebot.TextMessage) {

	userID, roomID, groupID, data, exist, msgType := FetchDataSource(event)
	//fmt.Println(data, exist, userID, groupID, roomID)

	mainType := strings.Split(message.Text, "/")
	lenSplitted := len(mainType)

	msgCategory := ""
	if lenSplitted > 1 {
		msgCategory = mainType[1]
	}

	if msgCategory == ADD_EXPENSE {
		handleAddExpense(mainType, event, exist, userID, roomID, groupID, data, msgType)
	} else if msgCategory == ADD_INCOME {

	} else if msgCategory == PLAN {

	}

	if !exist {
		fmt.Println("||||||||||||||||||||||||||||||||||||||||||||||||||||||||||||")
		data = initDataWallet(userID, roomID, groupID, msgType)
		prepareUpdateData(data, exist, userID, roomID, groupID, msgType)

	}
	/*if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(message.ID+":"+message.Text+" OK!")).Do(); err != nil {
		log.Print(err)
	}*/
	/*imageURL := "https://drive.google.com/file/d/0Bx6cTEFypiiNaHVTcXV5VkFpbFE/view?usp=sharing"
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
	}*/
}

func handleSticker(event *linebot.Event, message *linebot.StickerMessage) {
	if _, err := bot.ReplyMessage(
		event.ReplyToken,
		linebot.NewStickerMessage(message.PackageID, message.StickerID),
	).Do(); err != nil {
		log.Println(err)
	}
}

func handleMessage(event *linebot.Event) {
	switch message := event.Message.(type) {
	case *linebot.TextMessage:
		handleTextMessage(event, message)
	case *linebot.StickerMessage:
		handleSticker(event, message)
	}
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

			handleMessage(event)

		} else if event.Type == linebot.EventTypePostback {
			/*if _, err = bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage("iniPostback")).Do(); err != nil {
				log.Print(err)
			}*/

			//imageURL := "https://github.com/AdityaMili95/Wallte/blob/master/README/qI5Ujdy9n1.png"
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

			/*profile, err := bot.GetProfile(event.Source.UserID).Do()
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
			}*/

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
