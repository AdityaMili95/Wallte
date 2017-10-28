package simsimi

import (
	"fmt"
	"testing"
)

func TestCreateSimSimiSession(t *testing.T) {
	Debug = true
	session, createErr := CreateSimSimiSession("Noah")
	if createErr != nil {
		t.Fatal(createErr)
	}
	fmt.Println(session)
	var talkErr error
	var responseText string
	responseText, talkErr = session.Talk("Hello!")
	if talkErr != nil {
		t.Fatal(talkErr)
	}
	fmt.Println(responseText)

	responseText, talkErr = session.Talk("天气如何？")
	if talkErr != nil {
		t.Fatal(talkErr)
	}
	fmt.Println(responseText)
}
