package main

import (
	//"context"
	"encoding/json"
	"fmt"
	//"html/template"
	"bytes"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
	//"database/sql"

	"github.com/NoahShen/go-simsimi"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	//cdp "github.com/knq/chromedp"
	"github.com/line/line-bot-sdk-go/linebot"
	"github.com/xuyu/goredis"
	"github.com/zabawaba99/firego"
)

var bot *linebot.Client
var session *simsimi.SimSimiSession
var Redis *goredis.Redis
var db *sqlx.DB

//var cumaTest int

const (
	ADD_EXPENSE = "add-expense"
	ADD_INCOME  = "add-income"
	REPORT      = "report"
	OTHER       = "other"
	GET_REPORT  = "get-report"
	DRAW        = "draw"
	USER        = 1
	ROOM        = 2
	GROUP       = 3
)

type DetailMessage struct {
	Desc_text       string
	Cost_Not_Number string
	Cost_Zero       string
}

type ChatBot struct {
	Complete          bool            `json:"complete"`
	CurrentNode       string          `json:"currentNode"`
	Input             string          `json:"input"`
	SpeechResponse    string          `json:"speechResponse"`
	Intent            IntentData      `json:"intent"`
	Parameters        []ParameterInfo `json:"parameters"`
	MissingParameters []string        `json:"missingParameters"`
	Context           interface{}     `json:"context"`
}

type IntentData struct {
	Name    string `json:"name"`
	StoryId string `json:"storyId"`
}

type ParameterInfo struct {
	Required bool   `json:"required"`
	Type     string `json:"type"`
	Name     string `json:"name"`
}

var monthToInt = map[string]int{
	"January":   1,
	"February":  2,
	"March":     3,
	"April":     4,
	"May":       5,
	"June":      6,
	"July":      7,
	"August":    8,
	"September": 9,
	"October":   10,
	"November":  11,
	"December":  12,
}

var continent = map[string]string{
	"africa":       "Africa",
	"antartica":    "Antartica",
	"asia":         "Asia",
	"europe":       "Europe",
	"northamerica": "North America",
	"oceania":      "Oceania",
	"southamerica": "South America",
}

var keyToInfo = map[string]map[string]TransactionInfo{
	"food": map[string]TransactionInfo{
		"breakfast": TransactionInfo{SpentType: "Breakfast", Category: "Food", SubCategory: "Daily Food"},
		"lunch":     TransactionInfo{SpentType: "Lunch", Category: "Food", SubCategory: "Daily Food"},
		"dinner":    TransactionInfo{SpentType: "Dinner", Category: "Food", SubCategory: "Daily Food"},
		"snack":     TransactionInfo{SpentType: "Snack", Category: "Food", SubCategory: "Side Food"},
		"grocery":   TransactionInfo{SpentType: "Grocery", Category: "Food", SubCategory: "Side Food"},
		"beverages": TransactionInfo{SpentType: "Beverages", Category: "Food", SubCategory: "Side Food"},
	},
	"transport": map[string]TransactionInfo{
		"bus":        TransactionInfo{SpentType: "Bus", Category: "Transport", SubCategory: "Public Transportation #1"},
		"train":      TransactionInfo{SpentType: "Train", Category: "Transport", SubCategory: "Public Transportation #1"},
		"taxi":       TransactionInfo{SpentType: "Taxi", Category: "Transport", SubCategory: "Public Transportation #1"},
		"plane":      TransactionInfo{SpentType: "Plane", Category: "Transport", SubCategory: "Public Transportation #2"},
		"online":     TransactionInfo{SpentType: "Online Ride", Category: "Transport", SubCategory: "Public Transportation #2"},
		"ship":       TransactionInfo{SpentType: "Ship", Category: "Transport", SubCategory: "Public Transportation #2"},
		"car":        TransactionInfo{SpentType: "Car", Category: "Transport", SubCategory: "More Personal Ride"},
		"motorcycle": TransactionInfo{SpentType: "Motorcycle", Category: "Transport", SubCategory: "More Personal Ride"},
		"bicycle":    TransactionInfo{SpentType: "Bicycle", Category: "Transport", SubCategory: "More Personal Ride"},
		"traffic":    TransactionInfo{SpentType: "Traffic", Category: "Transport", SubCategory: "Others"},
		"parking":    TransactionInfo{SpentType: "Parking", Category: "Transport", SubCategory: "Others"},
		"ticket":     TransactionInfo{SpentType: "Ticket", Category: "Transport", SubCategory: "Others"},
	},
	"social": map[string]TransactionInfo{
		"movie":       TransactionInfo{SpentType: "Movie", Category: "Social", SubCategory: "Fun"},
		"music":       TransactionInfo{SpentType: "Music", Category: "Social", SubCategory: "Fun"},
		"gift":        TransactionInfo{SpentType: "Gift", Category: "Social", SubCategory: "Fun"},
		"clothes":     TransactionInfo{SpentType: "Clothes", Category: "Social", SubCategory: "Shopping"},
		"accessories": TransactionInfo{SpentType: "Accessories", Category: "Social", SubCategory: "Shopping"},
		"cosmetic":    TransactionInfo{SpentType: "Cosmetic", Category: "Social", SubCategory: "Shopping"},
		"voucher":     TransactionInfo{SpentType: "Voucher", Category: "Social", SubCategory: "Gaming"},
		"bet":         TransactionInfo{SpentType: "Bet", Category: "Social", SubCategory: "Gaming"},
		"rental":      TransactionInfo{SpentType: "Rental", Category: "Social", SubCategory: "Gaming"},
		"club":        TransactionInfo{SpentType: "Club", Category: "Social", SubCategory: "Anything Else"},
		"bar":         TransactionInfo{SpentType: "Bar", Category: "Social", SubCategory: "Anything Else"},
		"park":        TransactionInfo{SpentType: "Park", Category: "Social", SubCategory: "Anything Else"},
	},
	"life": map[string]TransactionInfo{
		"checkup":    TransactionInfo{SpentType: "Checkup", Category: "Life", SubCategory: "Health"},
		"inpatient":  TransactionInfo{SpentType: "Inpatient", Category: "Life", SubCategory: "Health"},
		"outpatient": TransactionInfo{SpentType: "Outpatient", Category: "Life", SubCategory: "Health"},
		"vitamin":    TransactionInfo{SpentType: "Vitamin", Category: "Life", SubCategory: "Treatment"},
		"medicine":   TransactionInfo{SpentType: "Medicine", Category: "Life", SubCategory: "Treatment"},
		"ointment":   TransactionInfo{SpentType: "Ointment", Category: "Life", SubCategory: "Treatment"},
	},
	"other": map[string]TransactionInfo{
		"tax":            TransactionInfo{SpentType: "Tax", Category: "Other", SubCategory: "Payment"},
		"bill":           TransactionInfo{SpentType: "Bill", Category: "Other", SubCategory: "Payment"},
		"rent":           TransactionInfo{SpentType: "Rent", Category: "Other", SubCategory: "Payment"},
		"toiletries":     TransactionInfo{SpentType: "Toiletries", Category: "Other", SubCategory: "Other Needs"},
		"electronic":     TransactionInfo{SpentType: "Electronic", Category: "Other", SubCategory: "Other Needs"},
		"tools":          TransactionInfo{SpentType: "Tools", Category: "Other", SubCategory: "Other Needs"},
		"undescribeable": TransactionInfo{SpentType: "Undescribeable", Category: "Other", SubCategory: "Undescribeable"},
	},
	"income": map[string]TransactionInfo{
		"business":   TransactionInfo{SpentType: "Business", Category: "Income", SubCategory: "Business"},
		"investment": TransactionInfo{SpentType: "Investment", Category: "Income", SubCategory: "Investment"},
		"transfer":   TransactionInfo{SpentType: "Transfer", Category: "Income", SubCategory: "Transfer"},
		"other":      TransactionInfo{SpentType: "Other", Category: "Income", SubCategory: "Other"},
	},
}

type DataWallet struct {
	Data Wallet
}

type Wallet struct {
	UserInfo    Info
	RoomInfo    Info
	GroupInfo   Info
	Currency    string
	Money       int
	Income      map[int]map[int]map[int]*DayTransaction
	Expense     map[int]map[int]map[int]*DayTransaction
	Last_Action *LastAction
	Chart       *MyChart
	Silent      bool
	LastTalk    *ChatBot `json:"last_talk"`
}

type MyChart struct {
	MainImage    string
	PreviewImage string
}

type LastAction struct {
	Status       bool
	Keyword      string
	Description  string
	Price        int
	Key          string
	Category     string
	SubCategory  string
	SpentType    string
	Created_date string
}

type Info struct {
	ID string
}

type TransactionInfo struct {
	Created_by   string
	Price        int
	Created_date string
	Planned_date string
	Category     string
	SubCategory  string
	SpentType    string
	Description  string
}

type DayTransaction struct {
	All_Transactions []TransactionInfo
	Total            int
}

type Option struct {
	Label  string
	Action string
}

func main() {
	var err error
	//cumaTest = 0
	connectDB()
	connectRedis()
	session, _ = simsimi.CreateSimSimiSession("Wallte")
	bot, err = linebot.New(os.Getenv("CHANNEL_SECRET"), os.Getenv("CHANNEL_TOKEN"))
	log.Println("Bot:", bot, " err:", err)
	http.HandleFunc("/callback", callbackHandler)
	http.HandleFunc("/replyImage", replyImage)
	port := os.Getenv("PORT")
	addr := fmt.Sprintf(":%s", port)
	http.ListenAndServe(addr, nil)
}

func GenerateKey(strlen int) string {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, strlen)
	for i := range result {
		result[i] = chars[r.Intn(len(chars))]
	}
	return string(result)
}

func connectRedis() {
	var err error
	Redis, err = goredis.DialURL(os.Getenv("REDIS_CONNECT"))
	if err != nil {
		log.Println(err.Error())
		return
	}
}

func connectDB() error {
	var err error
	db, err = sqlx.Open("mysql", os.Getenv("DB_CONNECT"))
	if err != nil {
		log.Println(err.Error())
		return err
	}
	return nil
}

func GetFirebase(key string) string {
	fb := firego.New(os.Getenv("FIREBASE_CONNECT")+key, nil)

	var v interface{}
	if err := fb.Value(&v); err != nil {
		log.Fatal(err)
	}

	if v == nil {
		return ""
	}

	return v.(map[string]interface{})["json"].(string)
}

func GetRedis(key string) string {
	data, err := Redis.Get(key)

	if err != nil || data == nil {
		log.Println(err)
		return ""
	}

	return string(data)
}

func GetTimeInfo(t time.Time) (int, int, int, int, int, string) {
	year, month, day := t.Date()
	hour := t.Hour()
	minute := t.Minute()
	monthString := month.String()

	return year, monthToInt[monthString], day, hour, minute, monthString
}

func GetCurrentTime() (int, int, int, int, int, string) {
	t := time.Now()
	return GetTimeInfo(t)
}

func ParseTime(date string) (int, int, int, int, int, string) {
	t, _ := time.Parse("2006-01-02T15:04", date)
	return GetTimeInfo(t)
}

func LastDayOfMonth(t time.Time) int {
	firstDay := time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	lastDay := firstDay.AddDate(0, 1, 0).Add(-time.Nanosecond)
	return lastDay.Day()
}

func SetRedis(key string, value string) {

	err := Redis.Set(key, value, 0, 0, false, false)
	if err != nil {
		log.Printf("%#v\n", err)
		return
	}

}

func SetFirebase(key string, value string) {
	fb := firego.New(os.Getenv("FIREBASE_CONNECT")+key, nil)

	val := map[string]string{
		"json": value,
	}
	err := fb.Set(val)
	if err != nil {
		log.Printf("%#v\n", err)
	}
}

func executeQuery(query string) *sqlx.Rows {
	/*db, err := connectDB()
	if err != nil {
		log.Println(err.Error())
		return nil
	}
	defer db.Close()*/

	if db == nil {
		log.Println("Database connection failed")
		return nil
	}
	rows, err := db.Queryx(query)
	if err != nil {
		log.Println(err.Error())
		return nil
	}
	return rows
}

func executeInsert(json string, userID string, roomID string, groupID string) {
	/*db, err := connectDB()

	if err != nil {
		log.Println(err.Error())
		return
	}
	defer db.Close()*/

	if db == nil {
		log.Println("Database connection failed")
		return
	}

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
	/*db, err := connectDB()
	if err != nil {
		log.Println(err.Error())
		return
	}
	defer db.Close()*/

	if db == nil {
		log.Println("Database connection failed")
		return
	}

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
	SetFirebase(redisKey, res)
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

	res = GetFirebase(ID)
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
			Currency:  "IDR",
			Silent:    false,
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

func handleAddExpense(splitted []string, event *linebot.Event, exist bool, userID string, roomID string, groupID string, data *DataWallet, msgType int, isPostback bool, message string) (bool, bool) {
	imageURL := "https://github.com/AdityaMili95/Wallte/raw/master/README/qI5Ujdy9n1.png"
	lenSplitted := len(splitted)
	var template linebot.Template
	altText := ""
	valid := false
	okay := false
	keyword := strings.Join(splitted, "/")
	var info TransactionInfo
	must_update := true
	remove_last_action := true

	if lenSplitted == 4 {
		info, okay = keyToInfo[splitted[2]][splitted[3]]
	}

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
			linebot.NewTemplateMessage("Select Expense Category \U00100058", template),
		).Do(); err != nil {
			log.Print(err)
		}

		if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
			return false, false
		}

	} else if lenSplitted == 3 && splitted[2] == "food" && isPostback {

		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Daily Food", "Food you spend everyday to keep you alive!",
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
		altText = "What type of food did you buy  \U00100055"
		valid = true

		remove_last_action = true
		must_update = false
		if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
			must_update = false
			remove_last_action = false
		}

	} else if lenSplitted == 3 && splitted[2] == "transport" && isPostback {

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
		altText = "What type of transportation did you ride?  \U00100049"
		valid = true

		remove_last_action = true
		must_update = false
		if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
			must_update = false
			remove_last_action = false
		}

	} else if lenSplitted == 3 && splitted[2] == "social" && isPostback {

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
				imageURL, "Anything Else", "Tell me what are you doing!!",
				linebot.NewPostbackTemplateAction("Club", "/add-expense/fun/club", ""),
				linebot.NewPostbackTemplateAction("Bar", "/add-expense/fun/bar", ""),
				linebot.NewPostbackTemplateAction("Park", "/add-expense/fun/park", ""),
			),
		)
		altText = "Wow you just socialize! What did you do  \U0010006A"
		valid = true

		if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
			must_update = false
			remove_last_action = false
		}
	} else if lenSplitted == 3 && splitted[2] == "life" && isPostback {
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

		altText = "Please take care of yourself  \U001000B2"

		valid = true

		if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
			must_update = false
			remove_last_action = false
		}

	} else if lenSplitted == 3 && splitted[2] == "other" && isPostback {

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
				imageURL, "Undescribeable", "Describe for me please!",
				linebot.NewPostbackTemplateAction("Tell Me", "/add-expense/other/undescribeable", ""),
				linebot.NewURITemplateAction("Visit Author", "http://adityamili.com"),
				linebot.NewURITemplateAction("Go to Our Shop", "https://tokopedia.com/elefashionshop"),
			),
		)

		altText = "Tell me!! What do you cost for? \U0010009A"
		valid = true

		if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
			must_update = false
			remove_last_action = false
		}

	} else if lenSplitted == 4 && okay && isPostback {

		textAsk := fmt.Sprintf("How much did you cost ? \U0010008C\n\nChat me the number please: %s.", data.Data.Currency)
		replyTextMessage(event, textAsk)

		data.Data.Last_Action = &LastAction{Keyword: keyword, Status: true, Key: GenerateKey(100), SpentType: info.SpentType, Category: info.Category, SubCategory: info.SubCategory}

		remove_last_action = false
		must_update = true

	} else if exist && lenSplitted == 6 && splitted[4] == "datepick" && isPostback {

		if data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" || data.Data.Last_Action.Key != splitted[5] {
			replyTextMessage(event, "Oops this data is outdated \U0010009B")
			return false, false
		}
		mainType := strings.Split(data.Data.Last_Action.Keyword, "/")
		trans := keyToInfo[mainType[2]][mainType[3]]
		key := data.Data.Last_Action.Key
		date := event.Postback.Params.Datetime
		convertedDate := strings.Replace(date, "T", " ", -1)
		one := Option{
			Label:  "YES",
			Action: "/add-expense/confirm/yes/" + key + "/" + date,
		}

		two := Option{
			Label:  "NO",
			Action: "/add-expense/confirm/no/" + key + "/" + date,
		}

		title := fmt.Sprintf("Add This Expense?\U00100087\nCategory : %s\nType : %s\nCost : %s %d\nDescription : %s\nDate : %s", trans.Category, trans.SpentType, data.Data.Currency, data.Data.Last_Action.Price, data.Data.Last_Action.Description, convertedDate)
		confirmationMessage(event, title, one, two, "Confirm Your Expense!! \U00100080")

		//data.Data.Last_Action.Created_date = date
		//prepareUpdateData(data, true, userID, roomID, groupID, msgType)

		remove_last_action = false
		must_update = false

	} else if exist && lenSplitted == 6 && splitted[2] == "confirm" && isPostback {

		if data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" || data.Data.Last_Action.Key != splitted[4] {
			replyTextMessage(event, "Oops your confirmation is outdated \U00100088")
			remove_last_action = false
			must_update = false
		} else if splitted[3] == "yes" {

			created_date := splitted[5]
			year, month, day, _, _, _ := ParseTime(created_date)
			name := "-"
			profile, err := bot.GetProfile(event.Source.UserID).Do()
			if err != nil {
				name = profile.DisplayName
			} else {
				log.Println(err)
			}

			if data.Data.Expense == nil {
				data.Data.Expense = map[int]map[int]map[int]*DayTransaction{}
			}

			if data.Data.Expense[year] == nil {
				data.Data.Expense[year] = map[int]map[int]*DayTransaction{}
			}

			if data.Data.Expense[year][month] == nil {
				data.Data.Expense[year][month] = map[int]*DayTransaction{}
			}

			if data.Data.Expense[year][month][day] == nil {
				atr := []TransactionInfo{}
				data.Data.Expense[year][month][day] = &DayTransaction{Total: 0, All_Transactions: atr}
			}

			data.Data.Expense[year][month][day].All_Transactions = append(data.Data.Expense[year][month][day].All_Transactions, TransactionInfo{
				Created_by:   name,
				Price:        data.Data.Last_Action.Price,
				Description:  data.Data.Last_Action.Description,
				Created_date: created_date,
				Planned_date: "-",
				Category:     data.Data.Last_Action.Category,
				SubCategory:  data.Data.Last_Action.SubCategory,
				SpentType:    data.Data.Last_Action.SpentType,
			})

			data.Data.Expense[year][month][day].Total += data.Data.Last_Action.Price

			replyTextMessage(event, "Expense Recorded! \U00100097")
		} else {
			replyTextMessage(event, "Cancelled! \U0010007E")
		}

	} else {

		success := false
		data, success = talk(event, message, data)

		remove_last_action = false
		must_update = false

		if exist && data.Data.Last_Action != nil {
			remove_last_action = true
		}

		if success {
			remove_last_action = true
			must_update = true
		}

	}

	if valid {
		if _, err := bot.ReplyMessage(
			event.ReplyToken,
			linebot.NewTemplateMessage(altText, template),
		).Do(); err != nil {
			log.Print(err)
		}
	}

	return remove_last_action, must_update
}

func handleAddIncome(splitted []string, event *linebot.Event, exist bool, userID string, roomID string, groupID string, data *DataWallet, msgType int, isPostback bool, message string) (bool, bool) {

	imageURL := "https://github.com/AdityaMili95/Wallte/raw/master/README/qI5Ujdy9n1.png"
	lenSplitted := len(splitted)
	var template linebot.Template
	keyword := strings.Join(splitted, "/")
	okay := false
	var info TransactionInfo

	if lenSplitted == 3 {
		info, okay = keyToInfo["income"][splitted[2]]
	}

	if lenSplitted == 2 {
		template = linebot.NewImageCarouselTemplate(
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("Business", "/add-income/business", ""),
			),
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("Investment", "/add-income/investment", ""),
			),
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("Transfer", "/add-income/transfer", ""),
			),
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("Other", "/add-income/other", ""),
			),
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewURITemplateAction("Our Shop", "https://tokopedia.com/elefashionshop"),
			),
		)
		if _, err := bot.ReplyMessage(
			event.ReplyToken,
			linebot.NewTemplateMessage("Select Income Type \U00100095", template),
		).Do(); err != nil {
			log.Print(err)
		}

		if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
			return false, false
		}

	} else if lenSplitted == 3 && okay && isPostback {

		textAsk := fmt.Sprintf("Woww How much?!!\n\nChat me the number please \U0010007A : %s.", data.Data.Currency)
		replyTextMessage(event, textAsk)

		data.Data.Last_Action = &LastAction{Keyword: keyword, Status: true, Key: GenerateKey(100), SpentType: info.SpentType, Category: info.Category, SubCategory: info.SubCategory}

		return false, true

	} else if exist && lenSplitted == 5 && splitted[3] == "datepick" && isPostback {

		if data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" || data.Data.Last_Action.Key != splitted[4] {
			replyTextMessage(event, "Oops this data is outdated \U0010009B")
			return false, false
		}
		mainType := strings.Split(data.Data.Last_Action.Keyword, "/")
		trans := keyToInfo["income"][mainType[2]]
		key := data.Data.Last_Action.Key
		date := event.Postback.Params.Datetime
		convertedDate := strings.Replace(date, "T", " ", -1)
		one := Option{
			Label:  "YES",
			Action: "/add-income/confirm/yes/" + key + "/" + date,
		}

		two := Option{
			Label:  "NO",
			Action: "/add-income/confirm/no/" + key + "/" + date,
		}

		title := fmt.Sprintf("Add This Income?\U00100087\nCategory : %s\nType : %s\nCost : %s %d\nDescription : %s\nDate : %s", trans.Category, trans.SpentType, data.Data.Currency, data.Data.Last_Action.Price, data.Data.Last_Action.Description, convertedDate)
		confirmationMessage(event, title, one, two, "Confirm Your Income!! \U00100097")

		//data.Data.Last_Action.Created_date = date
		//prepareUpdateData(data, true, userID, roomID, groupID, msgType)

		/*if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
			return false, false
		}*/

		return false, false

	} else if exist && lenSplitted == 6 && splitted[2] == "confirm" && isPostback {

		if data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" || data.Data.Last_Action.Key != splitted[4] {
			replyTextMessage(event, "Oops your confirmation is outdated \U00100088")
			return false, false
		} else if splitted[3] == "yes" {

			created_date := splitted[5]
			year, month, day, _, _, _ := ParseTime(created_date)
			name := "-"
			profile, err := bot.GetProfile(event.Source.UserID).Do()
			if err != nil {
				name = profile.DisplayName
			} else {
				log.Println(err)
			}

			if data.Data.Income == nil {
				data.Data.Income = map[int]map[int]map[int]*DayTransaction{}
			}

			if data.Data.Income[year] == nil {
				data.Data.Income[year] = map[int]map[int]*DayTransaction{}
			}

			if data.Data.Income[year][month] == nil {
				data.Data.Income[year][month] = map[int]*DayTransaction{}
			}

			if data.Data.Income[year][month][day] == nil {
				atr := []TransactionInfo{}
				data.Data.Income[year][month][day] = &DayTransaction{Total: 0, All_Transactions: atr}
			}

			data.Data.Income[year][month][day].All_Transactions = append(data.Data.Income[year][month][day].All_Transactions, TransactionInfo{
				Created_by:   name,
				Price:        data.Data.Last_Action.Price,
				Description:  data.Data.Last_Action.Description,
				Created_date: created_date,
				Planned_date: "-",
				Category:     data.Data.Last_Action.Category,
				SubCategory:  data.Data.Last_Action.SubCategory,
				SpentType:    data.Data.Last_Action.SpentType,
			})

			data.Data.Income[year][month][day].Total += data.Data.Last_Action.Price

			replyTextMessage(event, "Income Recorded! \U00100097")
		} else {
			replyTextMessage(event, "Cancelled! \U0010007E")
		}

	} else {
		//NGAPAIN
		success := false
		data, success = talk(event, message, data)

		remove_last_action := false
		must_update := false

		if exist && data.Data.Last_Action != nil {
			remove_last_action = true
		}

		if success {
			remove_last_action = true
			must_update = true
		}

		return remove_last_action, must_update
	}

	return true, true
}

func HandleAdditionalOptions(splitted []string, event *linebot.Event, exist bool, userID string, roomID string, groupID string, data *DataWallet, msgType int, isPostback bool, message string) (bool, bool) {

	imageURL := "https://github.com/AdityaMili95/Wallte/raw/master/README/qI5Ujdy9n1.png"
	lenSplitted := len(splitted)
	var template linebot.Template

	okay := false

	if lenSplitted == 4 || lenSplitted == 5 || lenSplitted == 6 {
		_, okay = continent[splitted[3]]
	}

	if lenSplitted == 2 {

		silentText := "Silent"
		altText := "You don't want to chat me huh :("
		dataSilent := "/other/silent"
		btnSilenText := "SILENT"

		if exist && data.Data.Silent {
			silentText = "Talk to me"
			altText = "Lets make a talk!"
			dataSilent = "/other/talk"
			btnSilenText = "TALK"
		}

		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Currency", "Set currency to use",
				linebot.NewPostbackTemplateAction("SET", "/other/currency", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, silentText, altText,
				linebot.NewPostbackTemplateAction(btnSilenText, dataSilent, ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Wipe", "Delete all your saved data",
				linebot.NewPostbackTemplateAction("WIPE", "/other/wipe", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "About", "Get to Know Us",
				linebot.NewPostbackTemplateAction("HELLO", "/about-us", "/about-us"),
			),
		)
		if _, err := bot.ReplyMessage(
			event.ReplyToken,
			linebot.NewTemplateMessage("WHat do you want to do? \U00100009", template),
		).Do(); err != nil {
			log.Print(err)
		}

		if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
			return false, false
		}
	} else if lenSplitted == 3 && splitted[2] == "wipe" && isPostback {

		one := Option{
			Label:  "WIPE",
			Action: "/other/wipe/yes",
		}

		two := Option{
			Label:  "CANCEL",
			Action: "/other/wipe/no",
		}
		confirmationMessage(event, "All Saved Data Will Be Removed \U00100085\nAre you sure?", one, two, "Wipe your data? \U00100085")
		if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
			return false, false
		}
	} else if lenSplitted == 4 && splitted[2] == "wipe" && splitted[3] == "yes" && isPostback {

		replyTextMessage(event, "Data wiped \U0010007C\nYour data already reset")
		if !exist {
			return false, false
		}

		data = initDataWallet(userID, roomID, groupID, msgType)
		prepareUpdateData(data, true, userID, roomID, groupID, msgType)
		return false, false

	} else if lenSplitted == 4 && splitted[2] == "wipe" && splitted[3] == "no" && isPostback {

		replyTextMessage(event, "Yayy wipe cancelled \U0010007A")
		if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
			return false, false
		}
	} else if lenSplitted == 3 && splitted[2] == "silent" && isPostback {

		replyTextMessage(event, "Okay I will not chat you \U00100098\nMaybe I am too noisy for you")

		if data.Data.Silent {

			if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
				return false, false
			}
			return true, true
		}
		data.Data.Silent = true

	} else if lenSplitted == 3 && splitted[2] == "talk" && isPostback {

		replyTextMessage(event, "You want to chat me again? How can \U0010007A\nI am so happy to have someone to converse with, sometimes I feel really lonely you know \U00100092")

		if !data.Data.Silent {
			if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
				return false, false
			}
			return true, true
		}

		data.Data.Silent = false
	} else if lenSplitted == 3 && splitted[2] == "about" && isPostback {

		if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
			return false, false
		}
	} else if lenSplitted == 3 && splitted[2] == "currency" && isPostback {
		template := linebot.NewImageCarouselTemplate(
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("Africa", "/other/currency/africa", ""),
			),
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("Antartica", "/other/currency/antartica", ""),
			),
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("Asia", "/other/currency/asia", ""),
			),
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("Europe", "/other/currency/europe", ""),
			),
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("North Amrca", "/other/currency/northamerica", ""),
			),
			linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("Oceania", "/other/currency/oceania", ""),
			), linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("South Amrca", "/other/currency/southamerica", ""),
			), linebot.NewImageCarouselColumn(
				imageURL,
				linebot.NewPostbackTemplateAction("Tell Me", "/other/currency/my-currency", ""),
			),
		)
		if _, err := bot.ReplyMessage(
			event.ReplyToken,
			linebot.NewTemplateMessage("Select Income Type \U00100095", template),
		).Do(); err != nil {
			log.Print(err)
		}

		if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
			return false, false
		}
	} else if (lenSplitted == 4 || lenSplitted == 5) && splitted[2] == "currency" && okay && isPostback {
		//keyword := strings.Join(splitted, "/")
		replyContinentCurrency(event, splitted)

		if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
			return false, false
		}
	} else if (lenSplitted == 4) && splitted[2] == "currency" && splitted[3] == "my-currency" && isPostback {
		keyword := strings.Join(splitted, "/")

		replyTextMessage(event, "Tell me your currency \U0010007F in 3 characters!")
		data.Data.Last_Action = &LastAction{Keyword: keyword, Status: true}

		return false, true
	} else if lenSplitted == 6 && splitted[2] == "currency" && okay && isPostback {
		data.Data.Currency = splitted[5]
		replyTextMessage(event, "Yay currency changed! \U00100090\nYour current currency changed to: "+splitted[5])
	} else {

		success := false
		data, success = talk(event, message, data)

		remove_last_action := false
		must_update := false

		if exist && data.Data.Last_Action != nil {
			remove_last_action = true
		}

		if success {
			remove_last_action = true
			must_update = true
		}

		return remove_last_action, must_update
	}

	return true, true

}

func replyAfricaContinent(splitted []string) linebot.Template {
	var template linebot.Template
	imageURL := "https://github.com/AdityaMili95/Wallte/raw/master/README/qI5Ujdy9n1.png"
	base := "/other/currency/africa/"
	prefix := base + "select/"
	var lenSplitted = len(splitted)

	if lenSplitted == 4 {

		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Algerian", "Fractional Unit: Santeem\nد.ج :Symbol",
				linebot.NewPostbackTemplateAction("DZD", prefix+"DZD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Burundi", "Fractional Unit: Centime\nSymbol: Fr",
				linebot.NewPostbackTemplateAction("BIF", prefix+"BIF", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Cape Verde", "Fractional Unit: Centavo\nSymbol: Esc",
				linebot.NewPostbackTemplateAction("CVE", prefix+"CVE", ""),
			), linebot.NewCarouselColumn(
				imageURL, "CFA BCEAO", "Fractional Unit: Centime\nSymbol: Fr",
				linebot.NewPostbackTemplateAction("XOF", prefix+"XOF", ""),
			), linebot.NewCarouselColumn(
				imageURL, "CFA BEAC", "Fractional Unit: Centime\nSymbol: Fr",
				linebot.NewPostbackTemplateAction("XAF", prefix+"XAF", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Comoro", "Fractional Unit: Centime\nSymbol: Fr",
				linebot.NewPostbackTemplateAction("KMF", prefix+"KMF", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Congolese", "Fractional Unit: Centime\nSymbol: Fr",
				linebot.NewPostbackTemplateAction("CDF", prefix+"CDF", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Dalasi", "Fractional Unit: Butut\nSymbol: D",
				linebot.NewPostbackTemplateAction("GMD", prefix+"GMD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Djibouti", "Fractional Unit: Centime\nSymbol: Fr",
				linebot.NewPostbackTemplateAction("DJF", prefix+"DJF", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Other", "Maybe you can find your preferred currency here",
				linebot.NewPostbackTemplateAction("NEXT", base+"next-one", ""),
			),
		)

	} else if splitted[4] == "next-one" {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Dobra", "Fractional Unit: Cêntimo\nSymbol: Db",
				linebot.NewPostbackTemplateAction("STD", prefix+"STD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Egypt", "Fractional Unit: Piastre\nSymbol: £",
				linebot.NewPostbackTemplateAction("EGP", prefix+"EGP", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Eithiopia", "Fractional Unit: Santim\nSymbol: Br",
				linebot.NewPostbackTemplateAction("ETB", prefix+"ETB", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Euro", "Fractional Unit: Cent\nSymbol: €",
				linebot.NewPostbackTemplateAction("EUR", prefix+"EUR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Ghana", "Fractional Unit: Pesewa\nSymbol: ₵",
				linebot.NewPostbackTemplateAction("GHS", prefix+"GHS", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Guinea", "Fractional Unit: Centime\nSymbol: Fr",
				linebot.NewPostbackTemplateAction("GNF", prefix+"GNF", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Kenya", "Fractional Unit: Cent\nSymbol: Sh",
				linebot.NewPostbackTemplateAction("KES", prefix+"KES", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Kwacha", "Fractional Unit: Tambala\nSymbol: MK",
				linebot.NewPostbackTemplateAction("WMK", prefix+"WMK", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Kwanza", "Fractional Unit: Cêntimo\nSymbol: Kz",
				linebot.NewPostbackTemplateAction("AOA", prefix+"AOA", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Other", "Maybe you can find your preferred currency here",
				linebot.NewPostbackTemplateAction("NEXT", base+"next-two", ""),
			),
		)
	} else if splitted[4] == "next-two" {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Leone", "Fractional Unit: Cent\nSymbol: Le",
				linebot.NewPostbackTemplateAction("SLL", prefix+"SLL", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Liberia", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("LRD", prefix+"LRD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Libya", "Fractional Unit: Dirham\nل.د :Symbol",
				linebot.NewPostbackTemplateAction("LYD", prefix+"LYD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Lilangeni", "Fractional Unit: Cent\nSymbol: L",
				linebot.NewPostbackTemplateAction("SZL", prefix+"SZL", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Loti", "Fractional Unit: Sente\nSymbol: L",
				linebot.NewPostbackTemplateAction("LSL", prefix+"LSL", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Malagasy", "Fractional Unit: Iraimbilanja\nSymbol: Ar",
				linebot.NewPostbackTemplateAction("MGA", prefix+"MGA", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Mauritius", "Fractional Unit: Cent\nSymbol: ₨",
				linebot.NewPostbackTemplateAction("MUR", prefix+"MUR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Moroccan", "Fractional Unit: Centime\nد. م. :Symbol",
				linebot.NewPostbackTemplateAction("MAD", prefix+"MAD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Mozambique", "Fractional Unit: Centavo\nSymbol: MT",
				linebot.NewPostbackTemplateAction("MZN", prefix+"MZN", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Other", "Maybe you can find your preferred currency here",
				linebot.NewPostbackTemplateAction("NEXT", base+"next-three", ""),
			),
		)
	} else if splitted[4] == "next-three" {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Naira", "Fractional Unit: Kobo\nSymbol: ₦",
				linebot.NewPostbackTemplateAction("NGN", prefix+"NGN", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Nakfa", "Fractional Unit: Cent\nSymbol: Nfk",
				linebot.NewPostbackTemplateAction("ERN", prefix+"ERN", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Namibia", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("NAD", prefix+"NAD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Ouguiya", "Fractional Unit: Khoums\nSymbol: UM",
				linebot.NewPostbackTemplateAction("MRO", prefix+"MRO", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Pula", "Fractional Unit: Thebe\nSymbol: P",
				linebot.NewPostbackTemplateAction("BWP", prefix+"BWP", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Rand", "Fractional Unit: Cent\nSymbol: R",
				linebot.NewPostbackTemplateAction("ZAR", prefix+"ZAR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Rwanda", "Fractional Unit: Centime\nSymbol: Fr",
				linebot.NewPostbackTemplateAction("RWF", prefix+"RWF", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Saint Helena", "Fractional Unit: Penny\nSymbol: £",
				linebot.NewPostbackTemplateAction("SHP", prefix+"SHP", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Seychelles", "Fractional Unit: Cent\nSymbol: ₨",
				linebot.NewPostbackTemplateAction("SCR", prefix+"SCR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Other", "Maybe you can find your preferred currency here",
				linebot.NewPostbackTemplateAction("NEXT", base+"next-four", ""),
			),
		)
	} else if splitted[4] == "next-four" {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Somali", "Fractional Unit: Cent\nSymbol: Sh",
				linebot.NewPostbackTemplateAction("SOS", prefix+"SOS", ""),
			), linebot.NewCarouselColumn(
				imageURL, "South Sudan", "Fractional Unit: Piastre\nSymbol: £",
				linebot.NewPostbackTemplateAction("SSP", prefix+"SSP", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Sudan", "Fractional Unit: Piastre\nSymbol: £",
				linebot.NewPostbackTemplateAction("SDG", prefix+"SDG", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Tanzania", "Fractional Unit: Cent\nSymbol: Sh",
				linebot.NewPostbackTemplateAction("TZS", prefix+"TZS", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Tunisia", "Fractional Unit: Millime\nد.ت :Symbol",
				linebot.NewPostbackTemplateAction("TND", prefix+"TND", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Uganda", "Fractional Unit: Cent\nSymbol: Sh",
				linebot.NewPostbackTemplateAction("UGX", prefix+"UGX", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Zambia", "Fractional Unit: Ngwee\nSymbol: ZK",
				linebot.NewPostbackTemplateAction("ZMW", prefix+"ZMW", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Zimbabwe", "Fractional Unit: -\nSymbol: -",
				linebot.NewPostbackTemplateAction("ZWL", prefix+"ZWL", ""),
			),
		)
	}

	return template

}

func replyAntarticaContinent(splitted []string) linebot.Template {
	var template linebot.Template
	imageURL := "https://github.com/AdityaMili95/Wallte/raw/master/README/qI5Ujdy9n1.png"
	base := "/other/currency/antartica/"
	prefix := base + "select/"

	template = linebot.NewCarouselTemplate(
		linebot.NewCarouselColumn(
			imageURL, "Australia", "Fractional Unit: Cent\nSymbol: $",
			linebot.NewPostbackTemplateAction("AUD", prefix+"AUD", ""),
		), linebot.NewCarouselColumn(
			imageURL, "Euro", "Fractional Unit: Cent\nSymbol: €",
			linebot.NewPostbackTemplateAction("EUR", prefix+"EUR", ""),
		), linebot.NewCarouselColumn(
			imageURL, "Norwegia", "Fractional Unit: Øre\nSymbol: kr",
			linebot.NewPostbackTemplateAction("NOK", prefix+"NOK", ""),
		),
	)

	return template
}

func replyAsiaContinent(splitted []string) linebot.Template {
	var template linebot.Template
	imageURL := "https://github.com/AdityaMili95/Wallte/raw/master/README/qI5Ujdy9n1.png"
	base := "/other/currency/asia/"
	prefix := base + "select/"
	var lenSplitted = len(splitted)

	if lenSplitted == 4 {

		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Afghani", "Fractional Unit: Pul\nSymbol: ؋",
				linebot.NewPostbackTemplateAction("AFN", prefix+"AFN", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Armenian", "Fractional Unit: Luma\nSymbol: դր",
				linebot.NewPostbackTemplateAction("AMD", prefix+"AMD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Azerbaijan", "Fractional Unit: Qəpik\nSymbol: -",
				linebot.NewPostbackTemplateAction("AZN", prefix+"AZN", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Bahrain", "Fractional Unit: Fils\n.د.ب :Symbol",
				linebot.NewPostbackTemplateAction("BHD", prefix+"BHD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Baht", "Fractional Unit: Satang\nSymbol: ฿",
				linebot.NewPostbackTemplateAction("THB", prefix+"THB", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Brunei", "Fractional Unit: Sen\nSymbol: $",
				linebot.NewPostbackTemplateAction("BND", prefix+"BND", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Dong", "Fractional Unit: Hào\nSymbol: ₫",
				linebot.NewPostbackTemplateAction("VND", prefix+"VND", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Hong Kong", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("HKD", prefix+"HKD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "India", "Fractional Unit: Paisa\nSymbol: -",
				linebot.NewPostbackTemplateAction("INR", prefix+"INR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Other", "Maybe you can find your preferred currency here",
				linebot.NewPostbackTemplateAction("NEXT", base+"next-one", ""),
			),
		)

	} else if splitted[4] == "next-one" {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Iran", "Fractional Unit: Dinar\n﷼ :Symbol",
				linebot.NewPostbackTemplateAction("IRR", prefix+"IRR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Iraq", "Fractional Unit: Fils\nع.د :Symbol",
				linebot.NewPostbackTemplateAction("IQD", prefix+"IQD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Jordania", "Fractional Unit: Piastre\nد.ا :Symbol",
				linebot.NewPostbackTemplateAction("JOD", prefix+"JOD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Kip", "Fractional Unit: Att\nSymbol: ₭",
				linebot.NewPostbackTemplateAction("LAK", prefix+"LAK", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Kuwait", "Fractional Unit: Fils\nد.ك :Symbol",
				linebot.NewPostbackTemplateAction("KWD", prefix+"KWD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Kyat", "Fractional Unit: Pya\nSymbol: Ks",
				linebot.NewPostbackTemplateAction("MMK", prefix+"MMK", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Lari", "Fractional Unit: Tetri\nSymbol: ლ",
				linebot.NewPostbackTemplateAction("GEL", prefix+"GEL", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Lebanon", "Fractional Unit: Piastre\nل.ل :Symbol",
				linebot.NewPostbackTemplateAction("LBP", prefix+"LBP", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Ringgit", "Fractional Unit: Sen\nSymbol: RM",
				linebot.NewPostbackTemplateAction("MYR", prefix+"MYR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Other", "Maybe you can find your preferred currency here",
				linebot.NewPostbackTemplateAction("NEXT", base+"next-two", ""),
			),
		)
	} else if splitted[4] == "next-two" {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Nepal", "Fractional Unit: Paisa\nSymbol: ₨",
				linebot.NewPostbackTemplateAction("NPR", prefix+"NPR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Israel", "Fractional Unit: Agora\nSymbol: ₪",
				linebot.NewPostbackTemplateAction("ILS", prefix+"ILS", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Taiwan", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("TWD", prefix+"TWD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Ngultrum", "Fractional Unit: Chetrum\nSymbol: Nu.",
				linebot.NewPostbackTemplateAction("BTN", prefix+"BTN", ""),
			), linebot.NewCarouselColumn(
				imageURL, "North Korea", "Fractional Unit: Chon\nSymbol: ₩",
				linebot.NewPostbackTemplateAction("KPW", prefix+"KPW", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Pakistan", "Fractional Unit: Paisa\nSymbol: ₨",
				linebot.NewPostbackTemplateAction("PKR", prefix+"PKR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Pataca", "Fractional Unit: Avo\nSymbol: P",
				linebot.NewPostbackTemplateAction("MOP", prefix+"MOP", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Peso", "Fractional Unit: Centavo\nSymbol: ₱",
				linebot.NewPostbackTemplateAction("PHP", prefix+"PHP", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Qatar", "Fractional Unit: Dirham\nر.ق :Symbol",
				linebot.NewPostbackTemplateAction("QAR", prefix+"QAR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Other", "Maybe you can find your preferred currency here",
				linebot.NewPostbackTemplateAction("NEXT", base+"next-three", ""),
			),
		)
	} else if splitted[4] == "next-three" {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Oman", "Fractional Unit: Baisa\nر.ع. :Symbol",
				linebot.NewPostbackTemplateAction("OMR", prefix+"OMR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Riel", "Fractional Unit: Sen\nSymbol: ៛",
				linebot.NewPostbackTemplateAction("KHR", prefix+"KHR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Rufiyaa", "Fractional Unit: Laari\n.ރ :Symbol",
				linebot.NewPostbackTemplateAction("MVR", prefix+"MVR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Rupiah", "Fractional Unit: Sen\nSymbol: Rp",
				linebot.NewPostbackTemplateAction("IDR", prefix+"IDR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Arab Saudi", "Fractional Unit: Halala\nر.س :Symbol",
				linebot.NewPostbackTemplateAction("SAR", prefix+"SAR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Singapore", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("SGD", prefix+"SGD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Som", "Fractional Unit: Tyiyn\nSymbol: -",
				linebot.NewPostbackTemplateAction("KGS", prefix+"KGS", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Somoni", "Fractional Unit: Diram\nSymbol: ЅМ",
				linebot.NewPostbackTemplateAction("TJS", prefix+"TJS", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Sri Lanka", "Fractional Unit: Cent\nSymbol: Rs",
				linebot.NewPostbackTemplateAction("LKR", prefix+"LKR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Other", "Maybe you can find your preferred currency here",
				linebot.NewPostbackTemplateAction("NEXT", base+"next-four", ""),
			),
		)
	} else if splitted[4] == "next-four" {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Syria", "Fractional Unit: Piastre\nSymbol: £",
				linebot.NewPostbackTemplateAction("SYP", prefix+"SYP", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Taka", "Fractional Unit: Paisa\nSymbol: ৳",
				linebot.NewPostbackTemplateAction("BDT", prefix+"BDT", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Tenge", "Fractional Unit: Tïın\nSymbol: ₸",
				linebot.NewPostbackTemplateAction("KZT", prefix+"KZT", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Tugrik", "Fractional Unit: Möngö\nSymbol: ₮",
				linebot.NewPostbackTemplateAction("MNT", prefix+"MNT", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Turky", "Fractional Unit: Kuruş\nSymbol: -",
				linebot.NewPostbackTemplateAction("TRY", prefix+"TRY", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Turkmenistan", "Fractional Unit: Tennesi\nSymbol: m",
				linebot.NewPostbackTemplateAction("TMT", prefix+"TMT", ""),
			), linebot.NewCarouselColumn(
				imageURL, "UAE", "Fractional Unit: Fils\nد.إ :Symbol",
				linebot.NewPostbackTemplateAction("AED", prefix+"AED", ""),
			), linebot.NewCarouselColumn(
				imageURL, "US Dollar", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("USD", prefix+"USD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Uzbekistan", "Fractional Unit: Tiyin\nSymbol: -",
				linebot.NewPostbackTemplateAction("UZS", prefix+"UZS", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Other", "Maybe you can find your preferred currency here",
				linebot.NewPostbackTemplateAction("NEXT", base+"next-five", ""),
			),
		)
	} else if splitted[4] == "next-five" {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Won", "Fractional Unit: Jeon\nSymbol: ₩",
				linebot.NewPostbackTemplateAction("KRW", prefix+"KRW", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Yemen", "Fractional Unit: Fils\n﷼ :Symbol",
				linebot.NewPostbackTemplateAction("YER", prefix+"YER", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Yen", "Fractional Unit: Sen\nSymbol: ¥",
				linebot.NewPostbackTemplateAction("JPY", prefix+"JPY", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Yuan", "Fractional Unit: Fen\nSymbol: ¥",
				linebot.NewPostbackTemplateAction("CNY", prefix+"CNY", ""),
			),
		)
	}

	return template

}

func replyEuropeContinent(splitted []string) linebot.Template {
	var template linebot.Template
	imageURL := "https://github.com/AdityaMili95/Wallte/raw/master/README/qI5Ujdy9n1.png"
	base := "/other/currency/europe/"
	prefix := base + "select/"
	var lenSplitted = len(splitted)

	if lenSplitted == 4 {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Belarussia", "Fractional Unit: Kapyeyka\nSymbol: Br",
				linebot.NewPostbackTemplateAction("BYR", prefix+"BYR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Bulgaria", "Fractional Unit: Stotinka\nSymbol: лв",
				linebot.NewPostbackTemplateAction("BGN", prefix+"BGN", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Convertible", "Fractional Unit: Fening\nSymbol: KM",
				linebot.NewPostbackTemplateAction("BAM", prefix+"BAM", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Croatia", "Fractional Unit: Lipa\nSymbol: kn",
				linebot.NewPostbackTemplateAction("HRK", prefix+"HRK", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Czech", "Fractional Unit: Haléř\nSymbol: Kč",
				linebot.NewPostbackTemplateAction("CZK", prefix+"CZK", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Danish", "Fractional Unit: Øre\nSymbol: kr",
				linebot.NewPostbackTemplateAction("DKK", prefix+"DKK", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Denar", "Fractional Unit: Deni\nSymbol: ден",
				linebot.NewPostbackTemplateAction("MKD", prefix+"MKD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Euro", "Fractional Unit: Cent\nSymbol: €",
				linebot.NewPostbackTemplateAction("EUR", prefix+"EUR", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Forint", "Fractional Unit: Fillér\nSymbol: Ft",
				linebot.NewPostbackTemplateAction("HUF", prefix+"HUF", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Other", "Maybe you can find your preferred currency here",
				linebot.NewPostbackTemplateAction("NEXT", base+"next-one", ""),
			),
		)
	} else if splitted[4] == "next-one" {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Gibraltar", "Fractional Unit: Penny\nSymbol: £",
				linebot.NewPostbackTemplateAction("GIP", prefix+"GIP", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Hryvnia", "Fractional Unit: Kopiyka\nSymbol: ₴",
				linebot.NewPostbackTemplateAction("UAH", prefix+"UAH", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Iceland", "Fractional Unit: Eyrir\nSymbol: kr",
				linebot.NewPostbackTemplateAction("ISK", prefix+"ISK", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Latvia", "Fractional Unit: Santīms\nSymbol: Ls",
				linebot.NewPostbackTemplateAction("LVL", prefix+"LVL", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Lek", "Fractional Unit: Qindarkë\nSymbol: L",
				linebot.NewPostbackTemplateAction("ALL", prefix+"ALL", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Lithuania", "Fractional Unit: Centas\nSymbol: Lt",
				linebot.NewPostbackTemplateAction("LTL", prefix+"LTL", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Moldova", "Fractional Unit: Ban\nSymbol: L",
				linebot.NewPostbackTemplateAction("MDL", prefix+"MDL", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Romania", "Fractional Unit: Ban\nSymbol: L",
				linebot.NewPostbackTemplateAction("RON", prefix+"RON", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Norwegia", "Fractional Unit: Øre\nSymbol: kr",
				linebot.NewPostbackTemplateAction("NOK", prefix+"NOK", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Other", "Maybe you can find your preferred currency here",
				linebot.NewPostbackTemplateAction("NEXT", base+"next-two", ""),
			),
		)
	} else if splitted[4] == "next-two" {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Pound Sterling", "Fractional Unit: Penny\nSymbol: £",
				linebot.NewPostbackTemplateAction("GBP", prefix+"GBP", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Russia", "Fractional Unit: Kopek\nSymbol: р",
				linebot.NewPostbackTemplateAction("RUB", prefix+"RUB", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Serbia", "Fractional Unit: Para\nSymbol: дин",
				linebot.NewPostbackTemplateAction("RSD", prefix+"RSD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Swedish", "Fractional Unit: Öre\nSymbol: kr",
				linebot.NewPostbackTemplateAction("SEK", prefix+"SEK", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Swish", "Fractional Unit: Rappen\nSymbol: Fr",
				linebot.NewPostbackTemplateAction("CHF", prefix+"CHF", ""),
			), linebot.NewCarouselColumn(
				imageURL, "WIR Euro", "Fractional Unit: -\nSymbol: -",
				linebot.NewPostbackTemplateAction("CHE", prefix+"CHE", ""),
			), linebot.NewCarouselColumn(
				imageURL, "WIR Franc", "Fractional Unit: -\nSymbol: -",
				linebot.NewPostbackTemplateAction("CHW", prefix+"CHW", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Zloty", "Fractional Unit: Grosz\nSymbol: zł",
				linebot.NewPostbackTemplateAction("PLN", prefix+"PLN", ""),
			),
		)
	}

	return template
}

func replyNorthAmericaContinent(splitted []string) linebot.Template {
	var template linebot.Template
	imageURL := "https://github.com/AdityaMili95/Wallte/raw/master/README/qI5Ujdy9n1.png"
	base := "/other/currency/northamerica/"
	prefix := base + "select/"
	var lenSplitted = len(splitted)

	if lenSplitted == 4 {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Aruban", "Fractional Unit: Cent\nSymbol: ƒ",
				linebot.NewPostbackTemplateAction("AWG", prefix+"AWG", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Bahamia", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("BSD", prefix+"BSD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Balboa", "Fractional Unit: Centésimo\nSymbol: B/",
				linebot.NewPostbackTemplateAction("PAB", prefix+"PAB", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Barbados", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("BBD", prefix+"BBD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Belize", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("BZD", prefix+"BZD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Bermuda", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("BMD", prefix+"BMD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Canada", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("CAD", prefix+"CAD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Cayman", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("KYD", prefix+"KYD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Cordoba", "Fractional Unit: Centavo\nSymbol: C$",
				linebot.NewPostbackTemplateAction("NIO", prefix+"NIO", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Other", "Maybe you can find your preferred currency here",
				linebot.NewPostbackTemplateAction("NEXT", base+"next-one", ""),
			),
		)
	} else if splitted[4] == "next-one" {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Costa Rica", "Fractional Unit: Céntimo\nSymbol: ₡",
				linebot.NewPostbackTemplateAction("CRC", prefix+"CRC", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Cuban", "Fractional Unit: Centavo\nSymbol: $",
				linebot.NewPostbackTemplateAction("CUP", prefix+"CUP", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Danish", "Fractional Unit: Øre\nSymbol: kr",
				linebot.NewPostbackTemplateAction("DKK", prefix+"DKK", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Dominica", "Fractional Unit: Centavo\nSymbol: $",
				linebot.NewPostbackTemplateAction("DOP", prefix+"DOP", ""),
			), linebot.NewCarouselColumn(
				imageURL, "East Carib", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("XCD", prefix+"XCD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "El Salvador", "Fractional Unit: -\nSymbol: -",
				linebot.NewPostbackTemplateAction("SVC", prefix+"SVC", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Gourde", "Fractional Unit: Centime\nSymbol: G",
				linebot.NewPostbackTemplateAction("HTG", prefix+"HTG", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Jamaica", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("JMD", prefix+"JMD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Lempira", "Fractional Unit: Centavo\nSymbol: L",
				linebot.NewPostbackTemplateAction("HNL", prefix+"HNL", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Other", "Maybe you can find your preferred currency here",
				linebot.NewPostbackTemplateAction("NEXT", base+"next-two", ""),
			),
		)
	} else if splitted[4] == "next-two" {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Mexico", "Fractional Unit: Centavo\nSymbol: $",
				linebot.NewPostbackTemplateAction("MXN", prefix+"MXN", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Mexico", "Fractional Unit: -\nSymbol: -",
				linebot.NewPostbackTemplateAction("MXV", prefix+"MXV", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Peso", "Fractional Unit: Centavo\nSymbol: $",
				linebot.NewPostbackTemplateAction("CUC", prefix+"CUC", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Quetzal", "Fractional Unit: Centavo\nSymbol: Q",
				linebot.NewPostbackTemplateAction("GTQ", prefix+"GTQ", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Trinidad", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("TTD", prefix+"TTD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "US Dollar", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("USD", prefix+"USD", ""),
			),
		)
	}

	return template

}

func replyOceaniaContinent(splitted []string) linebot.Template {
	var template linebot.Template
	imageURL := "https://github.com/AdityaMili95/Wallte/raw/master/README/qI5Ujdy9n1.png"
	base := "/other/currency/oceania/"
	prefix := base + "select/"
	var lenSplitted = len(splitted)

	if lenSplitted == 4 {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Australia", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("AUD", prefix+"AUD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "CFP Franc", "Fractional Unit: Centime\nSymbol: Fr",
				linebot.NewPostbackTemplateAction("XPF", prefix+"XPF", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Fiji", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("FJD", prefix+"FJD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Kina", "Fractional Unit: Toea\nSymbol: K",
				linebot.NewPostbackTemplateAction("PGK", prefix+"PGK", ""),
			), linebot.NewCarouselColumn(
				imageURL, "New Zealand", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("NZD", prefix+"NZD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Pa'ranga", "Fractional Unit: Seniti\nSymbol: T$",
				linebot.NewPostbackTemplateAction("TOP", prefix+"TOP", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Salomon Isl.", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("SBD", prefix+"SBD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Tala", "Fractional Unit: Sene\nSymbol: T",
				linebot.NewPostbackTemplateAction("WST", prefix+"WST", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Vatu", "Fractional Unit: None\nSymbol: Vt",
				linebot.NewPostbackTemplateAction("VUV", prefix+"VUV", ""),
			),
		)
	}

	return template

}

func replySouthAmericaContinent(splitted []string) linebot.Template {
	var template linebot.Template
	imageURL := "https://github.com/AdityaMili95/Wallte/raw/master/README/qI5Ujdy9n1.png"
	base := "/other/currency/southamerica/"
	prefix := base + "select/"
	var lenSplitted = len(splitted)

	if lenSplitted == 4 {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Argentina", "Fractional Unit: Centavo\nSymbol: $",
				linebot.NewPostbackTemplateAction("ARS", prefix+"ARS", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Bolivar", "Fractional Unit: Céntimo\nSymbol: Bs F",
				linebot.NewPostbackTemplateAction("VEF", prefix+"VEF", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Bolivia", "Fractional Unit: Centavo\nSymbol: Bs",
				linebot.NewPostbackTemplateAction("BOB", prefix+"BOB", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Brazil", "Fractional Unit: Centavo\nSymbol: R$",
				linebot.NewPostbackTemplateAction("BRL", prefix+"BRL", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Chile", "Fractional Unit: Centavo\nSymbol: $",
				linebot.NewPostbackTemplateAction("CLP", prefix+"CLP", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Colombia", "Fractional Unit: Centavo\nSymbol: $",
				linebot.NewPostbackTemplateAction("COP", prefix+"COP", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Falkland", "Fractional Unit: Penny\nSymbol: £",
				linebot.NewPostbackTemplateAction("FKP", prefix+"FKP", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Guarani", "Fractional Unit: Céntimo\nSymbol: ₲",
				linebot.NewPostbackTemplateAction("PYG", prefix+"PYG", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Guyan", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("GYD", prefix+"GYD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Other", "Maybe you can find your preferred currency here",
				linebot.NewPostbackTemplateAction("NEXT", base+"next-one", ""),
			),
		)
	} else if splitted[4] == "next-one" {
		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Mvdol", "Fractional Unit: -\nSymbol: -",
				linebot.NewPostbackTemplateAction("BOV", prefix+"BOV", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Nuevo SOl", "Fractional Unit: Céntimo\nSymbol: S/",
				linebot.NewPostbackTemplateAction("PEN", prefix+"PEN", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Uruguay", "Fractional Unit: Centésimo\nSymbol: $",
				linebot.NewPostbackTemplateAction("UYU", prefix+"UYU", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Surinam", "Fractional Unit: Cent\nSymbol: $",
				linebot.NewPostbackTemplateAction("SRD", prefix+"SRD", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Unidad", "Fractional Unit: -\nSymbol: -",
				linebot.NewPostbackTemplateAction("COU", prefix+"COU", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Unidades", "Fractional Unit: -\nSymbol: -",
				linebot.NewPostbackTemplateAction("CLF", prefix+"CLF", ""),
			), linebot.NewCarouselColumn(
				imageURL, "Uruguay", "Fractional Unit: -\nSymbol: -",
				linebot.NewPostbackTemplateAction("UYI", prefix+"UYI", ""),
			),
		)

	}

	return template
}

func replyContinentCurrency(event *linebot.Event, splitted []string) {

	var template linebot.Template

	if splitted[3] == "africa" {
		template = replyAfricaContinent(splitted)
	} else if splitted[3] == "antartica" {
		template = replyAntarticaContinent(splitted)
	} else if splitted[3] == "asia" {
		template = replyAsiaContinent(splitted)
	} else if splitted[3] == "europe" {
		template = replyEuropeContinent(splitted)
	} else if splitted[3] == "northamerica" {
		template = replyNorthAmericaContinent(splitted)
	} else if splitted[3] == "oceania" {
		template = replyOceaniaContinent(splitted)
	} else if splitted[3] == "southamerica" {
		template = replySouthAmericaContinent(splitted)
	}

	if _, err := bot.ReplyMessage(
		event.ReplyToken,
		linebot.NewTemplateMessage("What is your currency huh? \U00100009", template),
	).Do(); err != nil {
		log.Print(err)
	}

}

func replyTextMessage(event *linebot.Event, text string) {
	if _, err := bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(text)).Do(); err != nil {
		log.Print(err)
	}
}

func talk(event *linebot.Event, message string, data *DataWallet) (*DataWallet, bool) {

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	text := "Huh budum..."

	var lastTalk *ChatBot

	if data.Data.LastTalk == nil {

		lastTalk = &ChatBot{
			Complete:          true,
			CurrentNode:       "",
			Input:             message,
			SpeechResponse:    "",
			MissingParameters: make([]string, 0),
			Parameters:        make([]ParameterInfo, 0),
			Intent: IntentData{
				Name:    "Welcome message",
				StoryId: "59aae7bd26f6f60007b06fb7",
			},
		}

	} else {
		lastTalk = data.Data.LastTalk
	}

	lastTalk.Input = message

	reqData, err := json.Marshal(lastTalk)

	if err != nil {
		log.Println("ERROR MARSHAL", err)
		replyTextMessage(event, text)
		return data, false
	}

	var request *http.Request

	request, _ = http.NewRequest(http.MethodPost, "https://wallte-mongodb-chatbot.herokuapp.com/api/v1", bytes.NewBuffer([]byte(reqData)))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("datatype", "json")

	resp, err := client.Do(request)

	if err != nil {
		log.Println("ERROR REQUEST", err)
		replyTextMessage(event, text)
		return data, false
	}
	defer resp.Body.Close()

	var result ChatBot
	body, err := ioutil.ReadAll(resp.Body)

	if err != nil {
		log.Println("ERROR READ", err)
		replyTextMessage(event, text)
		return data, false
	}

	err = json.Unmarshal(body, &result)

	if err != nil {
		log.Println("ERROR UNMARSHAL", err)
		replyTextMessage(event, text)
		return data, false
	}

	text = result.SpeechResponse
	replyTextMessage(event, text)

	data.Data.LastTalk = lastTalk
	return data, true
}

func CancelAction(data *DataWallet) *DataWallet {
	data.Data.Last_Action = &LastAction{}
	return data
}

func confirmationMessage(event *linebot.Event, title string, one Option, two Option, tmplMessage string) {
	template := linebot.NewConfirmTemplate(
		title,
		linebot.NewPostbackTemplateAction(one.Label, one.Action, ""),
		linebot.NewPostbackTemplateAction(two.Label, two.Action, ""),
	)
	if _, err := bot.ReplyMessage(
		event.ReplyToken,
		linebot.NewTemplateMessage(tmplMessage, template),
	).Do(); err != nil {
		log.Println(err)
	}
}

func handleAskDetail(event *linebot.Event, message *linebot.TextMessage, userID string, roomID string, groupID string, data *DataWallet, msgType int, d DetailMessage) {

	text := message.Text
	if data.Data.Last_Action.Price == 0 {
		val, err := strconv.Atoi(text)
		if err == nil && val > 0 {

			if val > 1000000000 {
				replyTextMessage(event, "Too much money \U00100083\nAdd multiple time if it is more than 1 Billion!")
				return
			}
			data.Data.Last_Action.Price = val
			replyTextMessage(event, d.Desc_text)
		} else if err != nil {
			replyTextMessage(event, d.Cost_Not_Number)
			data = CancelAction(data)
		} else if val < 1 {
			replyTextMessage(event, d.Cost_Zero)
			data = CancelAction(data)
		}

		prepareUpdateData(data, true, userID, roomID, groupID, msgType)
		return
	}

	if data.Data.Last_Action.Description == "" {

		if len(text) > 100 {
			text = text[0:97]
			text += "..."
		}

		data.Data.Last_Action.Description = text
		prepareUpdateData(data, true, userID, roomID, groupID, msgType)

		lastWeek := time.Now().AddDate(0, 0, -7)
		now := time.Now()
		curr := time.Now()
		curr.Format("2006-01-02T15:04")

		month := fmt.Sprintf("%d", curr.Month())
		if curr.Month() < 10 {
			month = "0" + month
		}

		day := fmt.Sprintf("%d", curr.Day())
		if curr.Day() < 10 {
			day = "0" + day
		}

		max := fmt.Sprintf("%d-%s-%sT23:59", curr.Year(), month, day)

		template := linebot.NewImageCarouselTemplate(

			linebot.NewImageCarouselColumn(
				"https://github.com/AdityaMili95/Wallte/raw/master/README/qI5Ujdy9n1.png",
				linebot.NewDatetimePickerTemplateAction("Select Date", data.Data.Last_Action.Keyword+"/datepick/"+data.Data.Last_Action.Key, "datetime", now.Format("2006-01-02T15:04"), max, lastWeek.Format("2006-01-02T00:00")),
			),
		)
		if _, err := bot.ReplyMessage(
			event.ReplyToken,
			linebot.NewTemplateMessage("Select Expense Date! \U00100084", template),
		).Do(); err != nil {
			log.Println(err)
		}
	}

}

func sendChartImage(splitted []string, event *linebot.Event, exist bool, userID string, roomID string, groupID string, data *DataWallet, msgType int) {

	//mainImg := "https://firebasestorage.googleapis.com/v0/b/wallte-2df83.appspot.com/o/1%2Fimg?alt=media&token=99f30f70-da12-4096-8dc7-887a4b9aa81a"
	//previewImg := "https://firebasestorage.googleapis.com/v0/b/wallte-2df83.appspot.com/o/1%2Fimg-preview?alt=media&token=e4e468c9-1f30-48ef-b6c8-f71d2c2a378a"

	if !exist || data.Data.Chart == nil {
		replyTextMessage(event, "You haven't rendered any chat! \U00100085\nPlease select a chart from report menu")
		return
	}

	if _, err := bot.ReplyMessage(
		event.ReplyToken,
		linebot.NewImageMessage(data.Data.Chart.MainImage, data.Data.Chart.PreviewImage),
	).Do(); err != nil {
		return
	}
}

func replyImage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", r.Header.Get("Origin"))
	w.Header().Set("Access-Control-Allow-Methods", "POST")

	err := r.ParseForm()
	if err != nil {
		return
	}

	mainImg := r.PostFormValue("imageURL")
	previewImg := r.PostFormValue("previewURL")
	msgType := r.PostFormValue("msgType")
	//token := r.PostFormValue("token")
	ID := r.PostFormValue("ID")

	//log.Println("||||||||||||||||||||||||||||||||", msgType, ID)

	data, exist := getUserData(ID)

	if !exist {
		return
	}

	data.Data.Chart = &MyChart{
		MainImage:    mainImg,
		PreviewImage: previewImg,
	}

	if msgType == "User" {
		prepareUpdateData(data, exist, ID, "", "", USER)
	} else if msgType == "Room" {
		prepareUpdateData(data, exist, "", ID, "", ROOM)
	} else if msgType == "Group" {
		prepareUpdateData(data, exist, "", "", ID, GROUP)
	}

	/*if _, err := bot.ReplyMessage(
		token,
		linebot.NewImageMessage(mainImg, previewImg),
	).Do(); err != nil {
		return
	}*/

	return

}

func aliasJSON(jsonText string) string {
	jsonText = strings.Replace(jsonText, "{", "Q", -1)
	jsonText = strings.Replace(jsonText, "}", "Z", -1)
	jsonText = strings.Replace(jsonText, "[", "W", -1)
	jsonText = strings.Replace(jsonText, "]", "P", -1)
	jsonText = strings.Replace(jsonText, ",", "U", -1)
	jsonText = strings.Replace(jsonText, ":", "B", -1)
	jsonText = strings.Replace(jsonText, "\"", "K", -1)
	return jsonText
}

func getJSONforChart(period string, day int, month int, year int, data *DataWallet) string {

	type dataDailyChart struct {
		Expense int
		Income  int
	}

	jsonText := ""

	if period == "daily" {

		tempData := dataDailyChart{Expense: 0, Income: 0}
		if data.Data.Income != nil && data.Data.Income[year] != nil && data.Data.Income[year][month] != nil && data.Data.Income[year][month][day] != nil && len(data.Data.Income[year][month][day].All_Transactions) > 0 {
			tempData.Income = data.Data.Income[year][month][day].Total
		}
		if data.Data.Expense != nil && data.Data.Expense[year] != nil && data.Data.Expense[year][month] != nil && data.Data.Expense[year][month][day] != nil && len(data.Data.Expense[year][month][day].All_Transactions) > 0 {
			tempData.Expense = data.Data.Expense[year][month][day].Total
		}
		jsonText, _ = Marshal(tempData)
		jsonText = aliasJSON(jsonText)

	} else if period == "monthly" {

		monthlyData := map[int]dataDailyChart{}
		for i := 1; i <= 12; i++ {
			totalIncome := 0
			totalExpense := 0

			for j := 1; j <= 31; j++ {
				if data.Data.Income != nil && data.Data.Income[year] != nil && data.Data.Income[year][i] != nil && data.Data.Income[year][i][j] != nil && len(data.Data.Income[year][i][j].All_Transactions) > 0 {
					totalIncome += data.Data.Income[year][i][j].Total
				}

				if data.Data.Expense != nil && data.Data.Expense[year] != nil && data.Data.Expense[year][i] != nil && data.Data.Expense[year][i][j] != nil && len(data.Data.Expense[year][i][j].All_Transactions) > 0 {
					totalExpense += data.Data.Expense[year][i][j].Total
				}
			}
			monthlyData[i] = dataDailyChart{Expense: totalExpense, Income: totalIncome}
		}
		jsonText, _ = Marshal(monthlyData)
		jsonText = aliasJSON(jsonText)

	} else if period == "yearly" {
		yearlyData := map[int]dataDailyChart{}
		for k := year - 6; k <= year; k++ {
			totalIncome := 0
			totalExpense := 0
			for i := 1; i <= 12; i++ {
				for j := 1; j <= 31; j++ {
					if data.Data.Income != nil && data.Data.Income[k] != nil && data.Data.Income[k][i] != nil && data.Data.Income[k][i][j] != nil && len(data.Data.Income[k][i][j].All_Transactions) > 0 {
						totalIncome += data.Data.Income[k][i][j].Total
					}

					if data.Data.Expense != nil && data.Data.Expense[k] != nil && data.Data.Expense[k][i] != nil && data.Data.Expense[k][i][j] != nil && len(data.Data.Expense[k][i][j].All_Transactions) > 0 {
						totalExpense += data.Data.Expense[k][i][j].Total
					}
				}
			}
			yearlyData[k] = dataDailyChart{Expense: totalExpense, Income: totalIncome}
		}
		jsonText, _ = Marshal(yearlyData)
		jsonText = aliasJSON(jsonText)
	}

	return jsonText

}

func getChartData(splitted []string, event *linebot.Event, exist bool, userID string, roomID string, groupID string, data *DataWallet, msgType int, isPostback bool, message string) {

	lenSplitted := len(splitted)
	var template linebot.Template
	imageURL := "https://github.com/AdityaMili95/Wallte/raw/master/README/qI5Ujdy9n1.png"
	altText := ""
	linkChart := "https://adityamiliapp.herokuapp.com/render_chart?"
	if msgType == USER {
		linkChart += "xyz=" + userID
	} else if msgType == ROOM {
		linkChart += "yyz=" + userID
	} else if msgType == GROUP {
		linkChart += "zyz=" + userID
	}

	if lenSplitted == 2 {

		imgTemplate := linebot.NewImagemapMessage(
			"https://github.com/AdityaMili95/Wallte/raw/master/README/chart/",
			"What chart do you like? I like pie one \U001000B6",
			linebot.ImagemapBaseSize{1040, 1040},
			linebot.NewMessageImagemapAction("/draw/pie", linebot.ImagemapArea{0, 0, 520, 520}),
			linebot.NewMessageImagemapAction("/draw/line", linebot.ImagemapArea{520, 0, 520, 520}),
			linebot.NewMessageImagemapAction("/draw/bar", linebot.ImagemapArea{0, 520, 520, 520}),
			linebot.NewMessageImagemapAction("/report/detail", linebot.ImagemapArea{520, 520, 520, 520}),
		)

		if _, err := bot.ReplyMessage(
			event.ReplyToken,
			imgTemplate,
		).Do(); err != nil {
			log.Print(err)
		}

		return

	} else if lenSplitted == 3 && splitted[2] == "detail" {

		template = linebot.NewCarouselTemplate(

			linebot.NewCarouselColumn(
				imageURL, "Per Day", "What do you spent for this day?",
				linebot.NewPostbackTemplateAction("Select", "/report/detail/daily", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Per Month", "Full detail of your financial in a month",
				linebot.NewPostbackTemplateAction("Select", "/report/detail/monthly", ""),
			),
		)
		altText = "Choose report's period! \U00100024"

	} else if lenSplitted == 3 && splitted[2] == "pie" {

		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Per Day", "You can report a PIE diagram in daily basis",
				linebot.NewPostbackTemplateAction("Select", "/report/pie/daily", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Per Month", "Yay! Summarize your expense and income report monthly",
				linebot.NewPostbackTemplateAction("Select", "/report/pie/monthly", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Per Year", "Maybe its new year? How is this year?",
				linebot.NewPostbackTemplateAction("Select", "/report/pie/yearly", ""),
			),
		)

		altText = "Choose report's period! \U00100024"
	} else if lenSplitted == 4 && (splitted[2] == "pie" || splitted[2] == "bar" || splitted[2] == "line" || splitted[2] == "detail") && (splitted[3] == "daily" || splitted[3] == "monthly" || splitted[3] == "yearly") && isPostback {

		title := "Select Day"
		postMsg := "/report/" + splitted[2] + "/daily"

		if splitted[3] == "monthly" {
			title = "Select Month"
			postMsg = "/report/" + splitted[2] + "/monthly"
		} else if splitted[3] == "yearly" {
			title = "Select Year"
			postMsg = "/report/" + splitted[2] + "/yearly"
		}

		now := time.Now()
		nowString := now.Format("2006-01-02")

		template = linebot.NewImageCarouselTemplate(

			linebot.NewImageCarouselColumn(
				"https://github.com/AdityaMili95/Wallte/raw/master/README/qI5Ujdy9n1.png",
				linebot.NewDatetimePickerTemplateAction(title, postMsg+"/datepick", "date", nowString, nowString, ""),
			),
		)

		altText = title + "! " + "\U0010007A"

	} else if lenSplitted == 5 && exist && splitted[4] == "datepick" && (splitted[2] == "pie" || splitted[2] == "bar" || splitted[2] == "line" || splitted[2] == "detail") && (splitted[3] == "daily" || splitted[3] == "monthly" || splitted[3] == "yearly") && isPostback {

		date := event.Postback.Params.Date
		date += "T00:00"
		year, month, day, _, _, monthName := ParseTime(date)
		//res, err := Marshal(data)
		/*if err != nil || res == "" {
			replyTextMessage(event, "Upss something happened \U00100088\nRender Cancelled!")
			return
		}*/

		if splitted[2] == "detail" && splitted[3] == "daily" {

			if (data.Data.Income == nil || data.Data.Income[year] == nil || data.Data.Income[year][month] == nil || data.Data.Income[year][month][day] == nil || len(data.Data.Income[year][month][day].All_Transactions) == 0) && (data.Data.Expense == nil || data.Data.Expense[year] == nil || data.Data.Expense[year][month] == nil || data.Data.Expense[year][month][day] == nil || len(data.Data.Expense[year][month][day].All_Transactions) == 0) {
				replyTextMessage(event, "There is no data here \U0010009C")
				return
			}

			reportText := fmt.Sprintf("%d-%s-%d\nPer Day Report\n\n", day, monthName, year)
			reportText += "\U0010007D Expense:\n\n"

			if data.Data.Expense != nil && data.Data.Expense[year] != nil && data.Data.Expense[year][month] != nil && data.Data.Expense[year][month][day] != nil && len(data.Data.Expense[year][month][day].All_Transactions) != 0 {

				for idx, tran := range data.Data.Expense[year][month][day].All_Transactions {

					tempText := fmt.Sprintf("%d. %s:%s:%s  # %s \n\n%s %d\n\n", (idx + 1), tran.Category, tran.SubCategory, tran.SpentType, tran.Description, data.Data.Currency, tran.Price)
					reportText += tempText
				}

			} else {
				reportText += "You have no Expense \U00100095\n\n"
			}

			reportText += "\U00100080 Income:\n\n"

			if data.Data.Income != nil && data.Data.Income[year] != nil && data.Data.Income[year][month] != nil && data.Data.Income[year][month][day] != nil && len(data.Data.Income[year][month][day].All_Transactions) != 0 {

				for idx, tran := range data.Data.Income[year][month][day].All_Transactions {

					tempText := fmt.Sprintf("%d. %s:%s  # %s \n\n%s %d\n\n", (idx + 1), tran.Category, tran.SpentType, tran.Description, data.Data.Currency, tran.Price)
					reportText += tempText

				}

			} else {
				reportText += "You have no Income \U00100094"
			}

			if len(reportText) > 2000 {
				reportText = reportText[0:1997]
				reportText += "..."
			}

			replyTextMessage(event, reportText)
			return

		} else if splitted[2] == "detail" && splitted[3] == "monthly" {

			expense := true
			income := true

			if data.Data.Expense == nil || data.Data.Expense[year] == nil || data.Data.Expense[year][month] == nil {
				expense = false
			}

			if data.Data.Income == nil && data.Data.Income[year] == nil || data.Data.Income[year][month] == nil {
				income = false
			}

			if !expense && !income {
				replyTextMessage(event, "There is no data in this month \U00100082")
				return
			}

			reportText := fmt.Sprintf("%s-%d\nPer Month Report\n\n", monthName, year)
			trimmedMonthName := monthName[0:3]

			t, err := time.Parse("2006-01-02T15:04", date)
			lastDay := 31

			if err == nil {
				lastDay = LastDayOfMonth(t)
			}

			reportText += "Expense:\n\n"
			totalExpense := 0
			totalIncome := 0

			for i := 1; i <= lastDay; i++ {
				space := ""
				if i < 10 {
					space = " "
				}
				if expense && data.Data.Expense[year][month][i] != nil && len(data.Data.Expense[year][month][i].All_Transactions) > 0 {
					reportText += fmt.Sprintf("%d %s %s: %s %d\n", i, trimmedMonthName, space, data.Data.Currency, data.Data.Expense[year][month][i].Total)
					totalExpense += data.Data.Expense[year][month][i].Total
				} else {
					reportText += fmt.Sprintf("%d %s %s: %s 0\n", i, trimmedMonthName, space, data.Data.Currency)
				}

			}

			reportText += "\n\nIncome:\n\n"

			for i := 1; i <= lastDay; i++ {
				space := ""
				if i < 10 {
					space = " "
				}
				if income && data.Data.Income[year][month][i] != nil && len(data.Data.Income[year][month][i].All_Transactions) > 0 {
					reportText += fmt.Sprintf("%d %s %s: %s %d\n", i, trimmedMonthName, space, data.Data.Currency, data.Data.Income[year][month][i].Total)
					totalIncome += data.Data.Income[year][month][i].Total
				} else {
					reportText += fmt.Sprintf("%d %s %s: %s 0\n", i, trimmedMonthName, space, data.Data.Currency)
				}

			}

			reportText += "\n==============="
			reportText += fmt.Sprintf("\nTotal Expense: %s %d", data.Data.Currency, totalExpense)
			reportText += fmt.Sprintf("\nTotal Income: %s %d", data.Data.Currency, totalIncome)

			if len(reportText) > 2000 {
				reportText = reportText[0:1997]
				reportText += "..."
			}

			replyTextMessage(event, reportText)

			return
		}

		jsonText := getJSONforChart(splitted[3], day, month, year, data)
		linkChart = fmt.Sprintf("%s&day=%d&month=%d&year=%d&period=%s&chartType=%s&tok=%s", linkChart, day, month, year, splitted[3], splitted[2], jsonText)

		log.Println("INI LOHHHHHHH ", linkChart)

		template = linebot.NewButtonsTemplate(
			imageURL, "Should I?", "Just to make sure you are ready \U0010000B",
			linebot.NewURITemplateAction("Render Now", linkChart),
		)

		altText = "Render your report now! \U00100091"
	} else if lenSplitted == 3 && splitted[2] == "line" {

		template = linebot.NewCarouselTemplate(

			linebot.NewCarouselColumn(
				imageURL, "Monthly", "Compare your financial to another month!",
				linebot.NewPostbackTemplateAction("Select", "/report/line/monthly", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Yearly", "Its time to see how your year going",
				linebot.NewPostbackTemplateAction("Select", "/report/line/yearly", ""),
			),
		)
		altText = "Choose report's period! \U00100024"
	} else if lenSplitted == 3 && splitted[2] == "bar" {

		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Monthly", "Bar like chocolate bar every month! nyam nyam",
				linebot.NewPostbackTemplateAction("Select", "/report/bar/monthly", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Yearly", "I hope next year I get more chocolates bar",
				linebot.NewPostbackTemplateAction("Select", "/report/bar/yearly", ""),
			),
		)
		altText = "Choose report's period! \U00100024"
	} else {
		data, _ = talk(event, message, data)
	}

	if _, err := bot.ReplyMessage(
		event.ReplyToken,
		linebot.NewTemplateMessage(altText, template),
	).Do(); err != nil {
		log.Print(err)
	}

	return

	if _, err := bot.ReplyMessage(
		event.ReplyToken,
		linebot.NewTemplateMessage("What chart do you like? I like pie one", template),
	).Do(); err != nil {
		log.Print(err)
	}

}

func handleTextMessage(event *linebot.Event, message *linebot.TextMessage) {

	userID, roomID, groupID, data, exist, msgType := FetchDataSource(event)
	//fmt.Println(data, exist, userID, groupID, roomID)

	mainType := strings.Split(message.Text, "/")
	lenSplitted := len(mainType)
	msgCategory := ""
	remove_last_action := false
	must_update := false

	if !exist {
		data = initDataWallet(userID, roomID, groupID, msgType)
	}

	if lenSplitted > 1 {
		msgCategory = mainType[1]
	}

	if msgCategory == ADD_EXPENSE {
		remove_last_action, _ = handleAddExpense(mainType, event, exist, userID, roomID, groupID, data, msgType, false, message.Text)
	} else if msgCategory == ADD_INCOME {
		remove_last_action, _ = handleAddIncome(mainType, event, exist, userID, roomID, groupID, data, msgType, false, message.Text)
	} else if msgCategory == REPORT {

		getChartData(mainType, event, exist, userID, roomID, groupID, data, msgType, false, message.Text)

		if exist && data.Data.Last_Action != nil && data.Data.Last_Action.Keyword != "" {
			remove_last_action = true
		}

	} else if msgCategory == GET_REPORT {
		sendChartImage(mainType, event, exist, userID, roomID, groupID, data, msgType)

		if exist && data.Data.Last_Action != nil && data.Data.Last_Action.Keyword != "" {
			remove_last_action = true
		}

	} else if lenSplitted == 3 && msgCategory == DRAW {

		getChartData(mainType, event, exist, userID, roomID, groupID, data, msgType, false, message.Text)

		if exist && data.Data.Last_Action != nil && data.Data.Last_Action.Keyword != "" {
			remove_last_action = true
		}

	} else if msgCategory == OTHER {

		remove_last_action, _ = HandleAdditionalOptions(mainType, event, exist, userID, roomID, groupID, data, msgType, false, message.Text)

	} else if lenSplitted == 2 && msgCategory == "about-us" {

		imageURL := "https://github.com/AdityaMili95/Wallte/raw/master/README/qI5Ujdy9n1.png"
		template := linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "My Website", "See all of my works in my lifetime",
				linebot.NewURITemplateAction("Visit Me", "http://www.adityamili.com"),
			),
			linebot.NewCarouselColumn(
				imageURL, "Shop", "I opened a shop that sells apparel for men",
				linebot.NewURITemplateAction("Visit Shop", "https://www.tokopedia.com/elefashionshop"),
			),
		)
		if _, err := bot.ReplyMessage(
			event.ReplyToken,
			linebot.NewTemplateMessage("WHat do you want to do? \U00100009", template),
		).Do(); err != nil {
			log.Print(err)
		}

		if exist && data.Data.Last_Action != nil && data.Data.Last_Action.Keyword != "" {
			remove_last_action = true
		}

	} else if exist && data.Data.Last_Action != nil && data.Data.Last_Action.Keyword != "" {
		detailType := strings.Split(data.Data.Last_Action.Keyword, "/")
		var d DetailMessage
		if detailType[1] == ADD_EXPENSE {
			d = DetailMessage{
				Desc_text:       "Give the description below! \U0010009D",
				Cost_Not_Number: "Ouchh! \U00100085 Cost is about how much which means it must be a number!!\n\nCancelled",
				Cost_Zero:       "Awww! \U00100083 if the number is less than 1 that mean there is no income!!\n\nCancelled",
			}
			handleAskDetail(event, message, userID, roomID, groupID, data, msgType, d)
		} else if detailType[1] == ADD_INCOME {
			d = DetailMessage{
				Desc_text:       "Tell me the description please \U00100078",
				Cost_Not_Number: "Ouchh! \U0010007C You have no income, do you?\n\nCancelled",
				Cost_Zero:       "Awww! \U0010009E if the cost is less than 1 that mean there is no cost!!\n\nCancelled",
			}
			handleAskDetail(event, message, userID, roomID, groupID, data, msgType, d)
		} else if detailType[1] == OTHER && detailType[2] == "currency" {
			inputText := message.Text

			if len(inputText) < 3 {
				replyTextMessage(event, "Maybe you don't know that currency code is always 3 character!! \U00100083\n\nCancelled!")
				remove_last_action = true
			} else {
				if len(inputText) > 3 {
					inputText = inputText[0:3]
				}

				if !checkAllAlpha(inputText) {
					replyTextMessage(event, "How funny is it...\nYou accidentally input number in currency code! \U00100079\n\nCancelled!")
				} else {
					inputText = strings.ToUpper(inputText)
					data.Data.Currency = inputText
					replyTextMessage(event, "Yay currency changed! \U00100090\nYour current currency changed to: "+inputText)

				}

				remove_last_action = true
			}

		} else {
			// ga valid
			success := false
			data, success = talk(event, message.Text, data)

			if success {
				remove_last_action = true
			}
		}

	} else {
		// ga ada last action
		success := false
		data, success = talk(event, message.Text, data)

		if success {
			must_update = true
		}
	}

	if remove_last_action {
		data.Data.Last_Action = &LastAction{}
		must_update = true
	}

	if must_update {
		prepareUpdateData(data, exist, userID, roomID, groupID, msgType)
	}

}

func checkAllAlpha(text string) bool {
	isAlpha := regexp.MustCompile(`^[A-Za-z]+$`).MatchString

	if !isAlpha(text) {
		return false
	}

	return true
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
		data = initDataWallet(userID, roomID, groupID, msgType)
	}

	mainType := strings.Split(msg, "/")
	lenSplitted := len(mainType)
	remove_last_action := false
	must_update := true

	msgCategory := ""
	if lenSplitted >= 3 {
		if mainType[1] == ADD_EXPENSE && lenSplitted > 3 {
			msgCategory = mainType[1]
		} else if lenSplitted >= 3 {
			msgCategory = mainType[1]
		}

	}

	if msgCategory == ADD_EXPENSE {
		remove_last_action, must_update = handleAddExpense(mainType, event, exist, userID, roomID, groupID, data, msgType, true, msg)
	} else if msgCategory == ADD_INCOME {
		remove_last_action, must_update = handleAddIncome(mainType, event, exist, userID, roomID, groupID, data, msgType, true, msg)
	} else if msgCategory == REPORT {
		getChartData(mainType, event, exist, userID, roomID, groupID, data, msgType, true, msg)

		if exist && data.Data.Last_Action != nil && data.Data.Last_Action.Keyword != "" {
			remove_last_action = true
		}

	} else if msgCategory == OTHER {
		remove_last_action, must_update = HandleAdditionalOptions(mainType, event, exist, userID, roomID, groupID, data, msgType, true, msg)
	}

	if remove_last_action {
		must_update = true
		data.Data.Last_Action = &LastAction{}
	}

	if must_update {
		prepareUpdateData(data, exist, userID, roomID, groupID, msgType)
	}
}

func callbackHandler(w http.ResponseWriter, r *http.Request) {

	//[!!!]cumaTest += 1
	//log.Println("|||||||||||||||||||||||||||||||||||||||||||||", cumaTest)

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
