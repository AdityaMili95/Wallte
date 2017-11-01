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
	stmt, err := tx.Prepare("INSERT INTO wallte_data(user_id,room_id,group_id,JSON) VALUES (?,?,?,?)")
	if err != nil {
		log.Println(err)
	}

	_, err = stmt.Exec(userID, roomID, groupID, json)
	if err != nil {
		log.Println(err)
	}

	tx.Commit()
	stmt.Close()
}

func executeUpdate(json string, userID string, roomID string, groupID string) {
	db, err := connectDB()
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer db.Close()
	tx := db.MustBegin()
	stmt, err := tx.Prepare("update wallte_data set JSON=? where user_id=? and room_id=? and group_id=?")
	if err != nil {
		log.Println(err)
	}

	_, err = stmt.Exec(json, userID, roomID, groupID)
	if err != nil {
		log.Println(err)
	}
	tx.Commit()
	stmt.Close()
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
	if exist {
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
	lenSplitted := len(splitted)
	var template linebot.Template
	altText := ""
	valid := false

	if lenSplitted == 2 {

		template = linebot.NewImageCarouselTemplate(
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

	} else if splitted[2] == "food" {

		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Daily Food", "Food you must be spent everyday to make sure you are alive!",
				linebot.NewPostbackTemplateAction("Breakfast", "/add-expense/food/breakfast", ""),
				linebot.NewPostbackTemplateAction("Lunch", "/add-expense/food/lunch", ""),
				linebot.NewPostbackTemplateAction("Dinner", "/add-expense/food/dinner", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Side Food", "Maybe you eat this food just for fun!",
				linebot.NewPostbackTemplateAction("Snack", "/add-expense/food/snack", ""),
				linebot.NewPostbackTemplateAction("Grocery", "/add-expense/food/grocery", ""),
				linebot.NewPostbackTemplateAction("Beverages", "/add-expense/food/beverages", ""),
			),
		)
		altText = "What type of food did you buy?"
		valid = true

	} else if splitted[2] == "transport" {

		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Public Transportation #1", "Its cheap but you need to put extra effort!",
				linebot.NewPostbackTemplateAction("Bus", "/add-expense/transport/bus", ""),
				linebot.NewPostbackTemplateAction("Train", "/add-expense/transport/train", ""),
				linebot.NewPostbackTemplateAction("Taxi", "/add-expense/transport/taxi", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Public Transportation #2", "There's a lot of public transportation out there!",
				linebot.NewPostbackTemplateAction("Plane ", "/add-expense/transport/plane", ""),
				linebot.NewPostbackTemplateAction("Online Ride", "/add-expense/transport/online", ""),
				linebot.NewPostbackTemplateAction("Ship ", "/add-expense/transport/ship", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "More Personal Ride", "More comfortable and give you extra space",
				linebot.NewPostbackTemplateAction("Car", "/add-expense/transport/car", ""),
				linebot.NewPostbackTemplateAction("MotorCycle", "/add-expense/transport/motorcycle", ""),
				linebot.NewPostbackTemplateAction("Bicycle", "/add-expense/transport/bicycle", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Others", "Hmmm tell me about ride!",
				linebot.NewPostbackTemplateAction("Traffic", "/add-expense/transport/traffic", ""),
				linebot.NewPostbackTemplateAction("Parking", "/add-expense/transport/parking", ""),
				linebot.NewPostbackTemplateAction("Ticket", "/add-expense/transport/ticket", ""),
				//linebot.NewPostbackTemplateAction("Reparation", "/add-expense/transport/reparation", ""),
			),
		)
		altText = "What type of transportation did you ride?"
		valid = true
	} else if splitted[2] == "social" {

		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Fun", "I am happy if you having fun with others!",
				linebot.NewPostbackTemplateAction("Movie", "/add-expense/fun/movie", ""),
				linebot.NewPostbackTemplateAction("Music", "/add-expense/fun/music", ""),
				linebot.NewPostbackTemplateAction("Gift", "/add-expense/fun/gift", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Shopping", "Don't buy uneeded item too much!",
				linebot.NewPostbackTemplateAction("Clothes", "/add-expense/fun/clothes", ""),
				linebot.NewPostbackTemplateAction("Accessories", "/add-expense/fun/accessories", ""),
				linebot.NewPostbackTemplateAction("Cosmetic", "/add-expense/fun/cosmetic", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Gaming", "Do you like game?",
				linebot.NewPostbackTemplateAction("Voucher", "/add-expense/fun/voucher", ""),
				linebot.NewPostbackTemplateAction("Bet", "/add-expense/fun/bet", ""),
				linebot.NewPostbackTemplateAction("Rental", "/add-expense/fun/rental", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Anything Else", "Tell me what you are doing!!",
				linebot.NewPostbackTemplateAction("Club", "/add-expense/fun/club", ""),
				linebot.NewPostbackTemplateAction("Bar", "/add-expense/fun/bar", ""),
				linebot.NewPostbackTemplateAction("Park", "/add-expense/fun/park", ""),
			),
		)
		altText = "Wow you just socialize! What did you do?"
		valid = true
	} else if splitted[2] == "life" {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Health", "Your health is the most important!",
				linebot.NewPostbackTemplateAction("Check Up", "/add-expense/health/checkup", ""),
				linebot.NewPostbackTemplateAction("InPatient", "/add-expense/health/inpatient", ""),
				linebot.NewPostbackTemplateAction("OutPatient", "/add-expense/health/outpatient", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Treatment", "Treat yourself, keep healthy!",
				linebot.NewPostbackTemplateAction("Vitamin", "/add-expense/health/vitamin", ""),
				linebot.NewPostbackTemplateAction("Medicine", "/add-expense/health/medicine", ""),
				linebot.NewPostbackTemplateAction("Ointment", "/add-expense/health/ointment", ""),
			),
		)

		altText = "Please take care of yourself :)"

		valid = true
	} else if splitted[2] == "other" {

		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Payment", "Do you pay for something?",
				linebot.NewPostbackTemplateAction("Tax", "/add-expense/other/tax", ""),
				linebot.NewPostbackTemplateAction("Bill", "/add-expense/other/bill", ""),
				linebot.NewPostbackTemplateAction("Rent", "/add-expense/other/rent", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Other Needs", "You don't know what you need until you need it",
				linebot.NewPostbackTemplateAction("Toiletries", "/add-expense/other/toiletries", ""),
				linebot.NewPostbackTemplateAction("Electronic", "/add-expense/other/electronic", ""),
				linebot.NewPostbackTemplateAction("Tools", "/add-expense/other/tools", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Undescribable", "Describe for me please!",
				linebot.NewPostbackTemplateAction("Tell Me", "/add-expense/other/undescribable", ""),
				linebot.NewPostbackTemplateAction("", "", ""),
				linebot.NewURITemplateAction("Go to Our Shop", "https://tokopedia.com/elefashionshop"),
			),
		)

		altText = "Tell me!! What do you cost for?  -.-"

		valid = true
	}

	if valid {
		if _, err := bot.ReplyMessage(
			event.ReplyToken,
			linebot.NewTemplateMessage(altText, template),
		).Do(); err != nil {
			log.Print(err)
		}
	}
}

func handleTextMessage(event *linebot.Event, message *linebot.TextMessage) {

	userID, roomID, groupID, data, exist, msgType := FetchDataSource(event)
	//fmt.Println(data, exist, userID, groupID, roomID)

	mainType := strings.Split(message.Text, "/")
	lenSplitted := len(mainType)
	valid := false

	msgCategory := ""
	if lenSplitted > 1 {
		msgCategory = mainType[1]
	}

	if msgCategory == ADD_EXPENSE {
		valid = true
		handleAddExpense(mainType, event, exist, userID, roomID, groupID, data, msgType)
	} else if msgCategory == ADD_INCOME {
		valid = true
	} else if msgCategory == PLAN {
		valid = true
	}

	if !exist {
		data = initDataWallet(userID, roomID, groupID, msgType)
	} else if valid {
		data.Data.Last_Action = LastAction{}
	}

	prepareUpdateData(data, exist, userID, roomID, groupID, msgType)
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

func handlePostback(event *linebot.Event) {
	msg := event.Postback.Data
	userID, roomID, groupID, data, exist, msgType := FetchDataSource(event)

	if !exist {
		return
	}

	mainType := strings.Split(msg, "/")
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

	prepareUpdateData(data, exist, userID, roomID, groupID, msgType)
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
			handlePostback(event)
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

	}
}
