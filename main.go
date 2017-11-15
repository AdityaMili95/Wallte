package main

import (
	//"context"
	"encoding/json"
	"fmt"
	//"html/template"
	"log"
	"math/rand"
	"net/http"
	"os"
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
			Currency:  "Rp.",
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

func handleAddExpense(splitted []string, event *linebot.Event, exist bool, userID string, roomID string, groupID string, data *DataWallet, msgType int, isPostback bool) (bool, bool) {
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
		altText = "What type of food did you buy  \U00100055"
		valid = true

		remove_last_action = false
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

		remove_last_action = false
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
		replyTextMessage(event, "How much did you cost ? \U0010008C\n\nChat me the number please:")

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
			Action: "/add-expense/confirm/yes/" + key,
		}

		two := Option{
			Label:  "NO",
			Action: "/add-expense/confirm/no/" + key,
		}

		title := fmt.Sprintf("Add This Expense?\U00100087\nCategory : %s\nType : %s\nCost : %s %d\nDescription : %s\nDate : %s", trans.Category, trans.SpentType, data.Data.Currency, data.Data.Last_Action.Price, data.Data.Last_Action.Description, convertedDate)
		confirmationMessage(event, title, one, two, "Confirm Your Expense!! \U00100080")

		data.Data.Last_Action.Created_date = date
		prepareUpdateData(data, true, userID, roomID, groupID, msgType)

		remove_last_action = false
		must_update = false

	} else if exist && lenSplitted == 5 && splitted[2] == "confirm" && isPostback {

		if data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" || data.Data.Last_Action.Key != splitted[4] {
			replyTextMessage(event, "Oops your confirmation is outdated \U00100088")
			remove_last_action = false
			must_update = false
		} else if splitted[3] == "yes" {

			created_date := data.Data.Last_Action.Created_date
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
		//NGAPAIN
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

func handleAddIncome(splitted []string, event *linebot.Event, exist bool, userID string, roomID string, groupID string, data *DataWallet, msgType int, isPostback bool) (bool, bool) {

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
		replyTextMessage(event, "Woww How much?!!\n\nChat me the number please \U0010007A : ")

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
			Action: "/add-income/confirm/yes/" + key,
		}

		two := Option{
			Label:  "NO",
			Action: "/add-income/confirm/no/" + key,
		}

		title := fmt.Sprintf("Add This Income?\U00100087\nCategory : %s\nType : %s\nCost : %s %d\nDescription : %s\nDate : %s", trans.Category, trans.SpentType, data.Data.Currency, data.Data.Last_Action.Price, data.Data.Last_Action.Description, convertedDate)
		confirmationMessage(event, title, one, two, "Confirm Your Income!! \U00100097")

		data.Data.Last_Action.Created_date = date
		prepareUpdateData(data, true, userID, roomID, groupID, msgType)

		/*if !exist || data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" {
			return false, false
		}*/

		return false, false

	} else if exist && lenSplitted == 5 && splitted[2] == "confirm" && isPostback {

		if data.Data.Last_Action == nil || data.Data.Last_Action.Keyword == "" || data.Data.Last_Action.Key != splitted[4] {
			replyTextMessage(event, "Oops your confirmation is outdated \U00100088")
			return false, false
		} else if splitted[3] == "yes" {

			created_date := data.Data.Last_Action.Created_date
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
	}

	return true, true
}

func replyTextMessage(event *linebot.Event, text string) {
	if _, err := bot.ReplyMessage(event.ReplyToken, linebot.NewTextMessage(text)).Do(); err != nil {
		log.Print(err)
	}
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

func getChartData(splitted []string, event *linebot.Event, exist bool, userID string, roomID string, groupID string, data *DataWallet, msgType int, isPostback bool) {

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
				imageURL, "Daily", "What do you spent for this day?",
				linebot.NewPostbackTemplateAction("Select", "/report/detail/daily", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Monthly", "Full detail of your financial in a month",
				linebot.NewPostbackTemplateAction("Select", "/report/detail/monthly", ""),
			),
		)
		altText = "Choose report's period! \U00100024"

	} else if lenSplitted == 3 && splitted[2] == "pie" {

		template = linebot.NewCarouselTemplate(
			linebot.NewCarouselColumn(
				imageURL, "Daily", "You can report a PIE diagram in daily basis",
				linebot.NewPostbackTemplateAction("Select", "/report/pie/daily", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Monthly", "Yay! Summarize your expense and income report monthly",
				linebot.NewPostbackTemplateAction("Select", "/report/pie/monthly", ""),
			),
			linebot.NewCarouselColumn(
				imageURL, "Yearly", "Maybe its new year? Compare your report yearly!",
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

			reportText := "\U0010007D Expense:\n\n"

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

			reportText := fmt.Sprintf("%d-%s-%d Monthly Report\n\n", day, monthName, year)
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

				if expense && data.Data.Expense[year][month][day] != nil && len(data.Data.Expense[year][month][day].All_Transactions) > 0 {
					reportText += fmt.Sprintf("%d %s : %s %d\n", i, trimmedMonthName, data.Data.Currency, data.Data.Expense[year][month][day].Total)
					totalExpense += data.Data.Expense[year][month][day].Total
				} else {
					reportText += fmt.Sprintf("%d %s : %s 0\n", i, trimmedMonthName, data.Data.Currency)
				}

			}

			reportText += "\n\nIncome:\n\n"

			for i := 1; i <= lastDay; i++ {

				if income && data.Data.Income[year][month][day] != nil && len(data.Data.Income[year][month][day].All_Transactions) > 0 {
					reportText += fmt.Sprintf("%d %s : %s %d\n", i, trimmedMonthName, data.Data.Currency, data.Data.Income[year][month][day].Total)
					totalIncome += data.Data.Income[year][month][day].Total
				} else {
					reportText += fmt.Sprintf("%d %s : %s 0\n", i, trimmedMonthName, data.Data.Currency)
				}

			}

			reportText += "\n===================="
			reportText += fmt.Sprintf("\nTotal Expense: %s %d", data.Data.Currency, totalExpense)
			reportText += fmt.Sprintf("\nTotal Income: %s %d", data.Data.Currency, totalIncome)

			if len(reportText) > 2000 {
				reportText = reportText[0:1997]
				reportText += "..."
			}

			replyTextMessage(event, reportText)

			return
		}

		linkChart = fmt.Sprintf("%s&day=%d&month=%d&year=%d&period=%s&chartType=%s", linkChart, day, month, year, splitted[3], splitted[2])

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
		//NGAPAIN
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
		remove_last_action, _ = handleAddExpense(mainType, event, exist, userID, roomID, groupID, data, msgType, false)
	} else if msgCategory == ADD_INCOME {
		remove_last_action, _ = handleAddIncome(mainType, event, exist, userID, roomID, groupID, data, msgType, false)
	} else if msgCategory == REPORT {

		getChartData(mainType, event, exist, userID, roomID, groupID, data, msgType, false)

		if exist && data.Data.Last_Action != nil && data.Data.Last_Action.Keyword != "" {
			remove_last_action = true
		}

	} else if msgCategory == GET_REPORT {
		sendChartImage(mainType, event, exist, userID, roomID, groupID, data, msgType)

		if exist && data.Data.Last_Action != nil && data.Data.Last_Action.Keyword != "" {
			remove_last_action = true
		}

	} else if lenSplitted == 3 && msgCategory == DRAW {

		getChartData(mainType, event, exist, userID, roomID, groupID, data, msgType, false)

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
		}

	} else {
		// ga ada last action
	}

	if remove_last_action {
		data.Data.Last_Action = &LastAction{}
		must_update = true
	}

	if must_update {
		prepareUpdateData(data, exist, userID, roomID, groupID, msgType)
	}

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
		remove_last_action, must_update = handleAddExpense(mainType, event, exist, userID, roomID, groupID, data, msgType, true)
	} else if msgCategory == ADD_INCOME {
		remove_last_action, must_update = handleAddIncome(mainType, event, exist, userID, roomID, groupID, data, msgType, true)
	} else if msgCategory == REPORT {
		getChartData(mainType, event, exist, userID, roomID, groupID, data, msgType, true)
		remove_last_action = true
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
