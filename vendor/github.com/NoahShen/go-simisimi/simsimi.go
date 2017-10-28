package simsimi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
)

var Debug = false
var Language = "ch"
var MaxTimes = 3 // The max time of retrieve response if the response is Ad
var AdKeywords = "http|qq|www|.cn|.com|.net|.org|.cc"
var AdKeywordMap = map[string]string{
	"ch": "|微信|扣扣|Unauthorized|关注|加q|加扣|打炮|贱鸡|加Q|约炮|微博|陌陌|QQ|电话|手机",
}

const (
	SimsimiUrl     = "http://www.simsimi.com/talk.htm?lc=%s"
	SimsimiTalkUrl = "http://www.simsimi.com/func/req?msg=%s&lc=%s"
)

type SimSimiSession struct {
	Id      string
	Name    string
	cookies []*http.Cookie
}

func CreateSimSimiSession(name string) (*SimSimiSession, error) {
	url := fmt.Sprintf(SimsimiUrl, Language)
	req := createHttpRequest(url)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	var sessionId string
	cookies := resp.Cookies()
	for _, cookie := range cookies {
		if "JSESSIONID" == strings.ToUpper(cookie.Name) {
			sessionId = cookie.Value
		}
	}
	if len(sessionId) == 0 {
		return nil, errors.New("Didn't get session id!")
	}
	return &SimSimiSession{sessionId, name, cookies}, nil
}

//{"id":25179158,"uid":7,"aid":5,"domain":"simsimi_ch","regtime":1368615465000,"request":"hello!","response":"U\n","loadTime":2}
type talkResponse struct {
	Id       int    `json:"id,omitempty"`
	Uid      int    `json:"uid,omitempty"`
	Aid      int    `json:"aid,omitempty"`
	Domain   string `json:"domain,omitempty"`
	RegTime  int64  `json:"regtime,omitempty"`
	Request  string `json:"request,omitempty"`
	Response string `json:"response,omitempty"`
	LoadTime int64  `json:"loadTime,omitempty"`
}

func (self *SimSimiSession) Talk(message string) (string, error) {
	t := MaxTimes
	var responseText string
	adRegex := AdKeywords + AdKeywordMap[Language]
	if Debug {
		log.Printf("[simsimi] Ad regex :%s \n", adRegex)
	}
	for t > 0 {
		var err error
		responseText, err := self.getResponseText(message)
		if err != nil {
			return "", err
		}
		// filter Ad
		matched, regexpErr := regexp.MatchString(adRegex, responseText)
		if !matched || regexpErr != nil {
			if Debug && regexpErr != nil {
				log.Printf("[simsimi] MatchString error:%v contains\n", regexpErr)
			}
			if Debug && matched {
				log.Printf("[simsimi] The response [%s] contains Ad\n", responseText)
			}
			return responseText, nil
		}
		t--
	}

	return responseText, nil
}

func (self *SimSimiSession) getResponseText(msg string) (string, error) {
	url := fmt.Sprintf(SimsimiTalkUrl, msg, Language)
	req := createHttpRequest(url)
	for _, cookie := range self.cookies {
		req.AddCookie(cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	bytes, readErr := ioutil.ReadAll(resp.Body)
	if readErr != nil {
		return "", readErr
	}
	if Debug {
		log.Printf("[simsimi] The response of [%s] : %s\n", msg, strings.TrimSpace(string(bytes)))
	}
	var talkResponse talkResponse
	unmarshalErr := json.Unmarshal(bytes, &talkResponse)
	return talkResponse.Response, unmarshalErr
}

func createHttpRequest(url string) *http.Request {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "*/*")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.31 (KHTML, like Gecko) Chrome/26.0.1410.64 Safari/537.31")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", fmt.Sprintf(SimsimiUrl, Language))
	return req
}
