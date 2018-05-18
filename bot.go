package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"os"
	"strconv"
	"strings"
	"time"
	"errors"

	"crypto/tls"
	"net/http"

	"github.com/boltdb/bolt"
	tb "gopkg.in/tucnak/telebot.v2"
)

const APP_ENV_DB_STORAGE = "BLUE_BOT_DB_STORAGE"
const APP_ENV_BOT_TOKEN = "SIGNIN_BOT_TOKEN"


type Step int

const (
	stepToConfirmFulName Step = iota
	stepToAskFullName
	stepToAskTos
	stepToAskSubscription
	stepToCreateAcount
	stepDone
)

type userInfo struct {
	DisplayName       string
	TosAgreed         bool
	Subscription      bool
	RegistrationStep  Step
	LastSigninRequest time.Time
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var userMap = make(map[int]*userInfo)

var defaultDbStorage = "my.db"
var defaultDbFileMode os.FileMode = 0600
var defaultBucket = "userinfo"


func test(){
	fmt.Printf("hello, world\n")
    var user userInfo
    user.DisplayName="test name"
    user.TosAgreed=false
    user.Subscription=true
    user.RegistrationStep=stepToAskTos

    log.Printf("User info step=%d displayName=%s tosAgreed=%s", user.RegistrationStep, user.DisplayName, user.TosAgreed)
        
    userBytes, err := json.Marshal(user)
    if err == nil{
    	os.Stdout.Write(userBytes)
	}

}

func appInit(){
	//Setup log
	log.SetFlags(log.LstdFlags | log.Llongfile)

	//Set env value
	tempValue := os.Getenv(APP_ENV_DB_STORAGE)
	if tempValue == ""{
		os.Setenv(APP_ENV_DB_STORAGE, defaultDbStorage)
	}
	err := initDb()
	if err != nil {
		log.Fatal(err)
	}
	err = initDbBucket(defaultBucket)
	if err != nil {
		log.Fatal(err)
	}

	tempValue = os.Getenv(APP_ENV_BOT_TOKEN)
	if tempValue == ""{
		log.Fatal("I dont have the KEY to open the door! :(")
	}
}

func initDb() (error) {
	storage := os.Getenv(APP_ENV_DB_STORAGE)
	log.Printf("InitDB: Initialize boltdb with storage %s!", storage)
	db, err := bolt.Open(storage, defaultDbFileMode, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal(err)
	} else {
		log.Printf("InitDB: DB is initialize successful to %s.", storage)
	}
	defer db.Close()
	return err
}

func initDbBucket(bucket string) (error) {
	storage := os.Getenv(APP_ENV_DB_STORAGE)
	db, err := bolt.Open(storage, defaultDbFileMode, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(bucket))
		if err != nil {
			errStr := fmt.Errorf("Could not create bucket %s error: %s", bucket, err)
			log.Print(errStr)
			return errStr
		}
		return nil
	})
	defer db.Close()
	return err
}
func updateUserInfo(user *userInfo, id int, bucket string) error {
	storage := os.Getenv(APP_ENV_DB_STORAGE)
	db, err := bolt.Open(storage, defaultDbFileMode, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal(err)
	}

	err = db.Update(func(tx *bolt.Tx) error {
		bk := tx.Bucket([]byte(bucket))
		log.Printf("User info step=%d displayName=%s tosAgreed=%s", user.RegistrationStep, user.DisplayName, user.TosAgreed)
		userBytes, err := json.Marshal(user)
		if err == nil {
			err = bk.Put([]byte(strconv.Itoa(id)), []byte(userBytes))
			if err == nil {
				log.Printf("updateUserInfo: Insert user id=%d value=%s to db successfully!", id, userBytes)
			}else{
				log.Printf("updateUserInfo: Failure insert user id=%d value=%s to db!", id, userBytes)
			}
		}else{
			log.Printf("updateUserInfo: Failure to jsonize user data with id=%d", id)
		}
		return nil
	})

	defer db.Close()
	return err
}

func getUserInfo(id int, bucket string) (userInfo, error) {
	storage := os.Getenv(APP_ENV_DB_STORAGE)
	db, err := bolt.Open(storage, defaultDbFileMode, &bolt.Options{Timeout: 1 * time.Second})
	if err != nil {
		log.Fatal(err)
	}
	var user userInfo
	err = db.View(func(tx *bolt.Tx) error {
		bk := tx.Bucket([]byte(bucket))
		userBytes := bk.Get([]byte(strconv.Itoa(id)))

		var users []userInfo
		json.Unmarshal(userBytes, &users)
		if len(users) > 1{
			user = users[0]
			return nil
		}
		errStr := "Not found info of user id=" + strconv.Itoa(id)
		return errors.New(errStr)
	})
	return user, err
}


func main() {

	appInit()

	test()

	//Create http client
	transCfg := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, //disable verify
	}
	client := &http.Client{Transport: transCfg}
	b, err := tb.NewBot(tb.Settings{
		Token:  os.Getenv(APP_ENV_BOT_TOKEN),
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
		Client: client,
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	b.Handle(tb.OnText, func(m *tb.Message) {
		log.Printf("Handle onText=%s", m.Text)
		handleReply(b, m)
	})

	b.Handle("/start", func(m *tb.Message) {
		log.Printf("Handle /start command=%s", m.Text)
		next(b, m)
	})

	b.Handle("/signin", func(m *tb.Message) {
		log.Printf("Handle /signin command=%s", m.Text)
		next(b, m)
	})

	b.Start()
}

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func getFullName(m *tb.Message) string {
	log.Printf("getFullName: from user id=%d", m.Sender.ID)
	if user, ok := userMap[m.Sender.ID]; ok {
		if user.DisplayName != "" {
			return user.DisplayName
		}
	}
	return fmt.Sprintf("%s %s", m.Sender.FirstName, m.Sender.LastName)
}

func send(b *tb.Bot, m *tb.Message, text string) (*tb.Message, error) {
	return b.Send(m.Sender, text)
}

func sendf(b *tb.Bot, m *tb.Message, text string, a ...interface{}) (*tb.Message, error) {
	return send(b, m, fmt.Sprintf(text, a...))
}

func sendYesNo(b *tb.Bot, m *tb.Message, text string) (*tb.Message, error) {
	yesBtn := tb.ReplyButton{Text: "Yes"}
	noBtn := tb.ReplyButton{Text: "No"}
	replyYesNo := [][]tb.ReplyButton{
		[]tb.ReplyButton{yesBtn, noBtn},
	}
	return b.Send(m.Sender,
		text,
		&tb.ReplyMarkup{
			ReplyKeyboard:       replyYesNo,
			ResizeReplyKeyboard: true,
			OneTimeKeyboard:     true,
		})
}

func sendYesNof(b *tb.Bot, m *tb.Message, text string, a ...interface{}) (*tb.Message, error) {
	return sendYesNo(b, m, fmt.Sprintf(text, a...))
}

func sendAndHideKeyboard(b *tb.Bot, m *tb.Message, text string) (*tb.Message, error) {
	return b.Send(m.Sender, text, &tb.ReplyMarkup{ReplyKeyboardRemove: true})
}

func sendfAndHideKeyboard(b *tb.Bot, m *tb.Message, text string, a ...interface{}) (*tb.Message, error) {
	return sendAndHideKeyboard(b, m, fmt.Sprintf(text, a...))
}

func next(b *tb.Bot, m *tb.Message) {
	if user, ok := userMap[m.Sender.ID]; ok {
		log.Printf("next: registrationStep=%d for User id=%d send msg=%s", user.RegistrationStep, m.Sender.ID, m.Text)
		updateUserInfo(user, m.Sender.ID, defaultBucket)
		funcArray := []func(*tb.Bot, *tb.Message){
			confirmDisplayName,
			askDisplayName,
			askTos,
			askSubcription,
			doCreateAccount,
			sendSigninLink}
		funcArray[user.RegistrationStep](b, m)
	} else {
		// registration
		startRegistration(b, m)
	}
}

func sendSigninLink(b *tb.Bot, m *tb.Message) {
	user := userMap[m.Sender.ID]
	last := user.LastSigninRequest
	log.Printf("sendSigninLink: send to user id=%d lastSigninRequest=%d registrationStep=%d,", m.Sender.ID, last, user.RegistrationStep)
	if !last.IsZero() {
		elapsed := time.Now().Sub(last).Minutes()
		if elapsed < 1.0 {
			send(b, m, "Please use the last sign-in URL provided, it is still valid.")
			return
		}
	}

	code := randString(10)
	_, err := sendfAndHideKeyboard(b, m,
		"Welcome %s, you may use this link to sign-in Kyber Network (the link will expire in 1 minute) - https://kyber.network/signin?code=%s&account=%d",
		getFullName(m),
		code,
		m.Sender.ID,
	)
	if err == nil {
		user.LastSigninRequest = time.Now()
	}
}

func containAny(array []string, item string) bool {
	for _, element := range array {
		if strings.EqualFold(element, item) {
			return true
		}
	}

	return false
}

func isYes(text string) bool {
	values := []string{"yes", "sure", "certainly", "ok", "okay", "fine", "indeed",
		"definitely", "of course", "affirmative", "obviously", "absolutely",
		"indubitably", "undoubtedly", "by all means"}
	return containAny(values, strings.TrimSpace(text))
}

func isNo(text string) bool {
	values := []string{"no", "never", "by no means", "no way", "veto"}
	return containAny(values, strings.TrimSpace(text))
}

func handleReply(b *tb.Bot, m *tb.Message) {
	if user, ok := userMap[m.Sender.ID]; ok {
		log.Printf("handleReply: User id=%d with registrationStep=%d", m.Sender.ID, user.RegistrationStep)
		switch user.RegistrationStep {
		case stepToConfirmFulName:
			if isYes(m.Text) {
				user.DisplayName = fmt.Sprintf("%s %s", m.Sender.FirstName, m.Sender.LastName)
				user.RegistrationStep = stepToAskTos
				next(b, m)
			} else if isNo(m.Text) {
				user.RegistrationStep = stepToAskFullName
				next(b, m)
			} else {
				next(b, m)
			}
		case stepToAskFullName:
			user.DisplayName = strings.Title(strings.TrimSpace(m.Text))
			user.RegistrationStep = stepToAskTos
			next(b, m)
		case stepToAskTos:
			if isYes(m.Text) {
				user.TosAgreed = true
				user.RegistrationStep = stepToAskSubscription
				next(b, m)
			} else {
				next(b, m)
			}
		case stepToAskSubscription:
			if isYes(m.Text) {
				user.Subscription = true
				user.RegistrationStep = stepToCreateAcount
				next(b, m)
			} else if isNo(m.Text) {
				user.Subscription = false
				user.RegistrationStep = stepToCreateAcount
				next(b, m)
			} else {
				next(b, m)
			}
		case stepToCreateAcount:
			// TODO: should done earlier, from the time acount created
			user.RegistrationStep = stepDone
			if isYes(m.Text) {
				sendSigninLink(b, m)
			} else {
				sendAndHideKeyboard(b, m, "Whenever you would like to sign-in Kyber Network, just type /signin")
			}
		default:
			informSignin(b, m)
		}
	} else {
		informSignin(b, m)
	}
}

func startRegistration(b *tb.Bot, m *tb.Message) {
	newUserInfo := userInfo{RegistrationStep: stepToConfirmFulName}
	userMap[m.Sender.ID] = &newUserInfo
	log.Printf("startRegistration: start Registration for user id=%d registrationStep=%d!", m.Sender.ID, newUserInfo.RegistrationStep)
	updateUserInfo(userMap[m.Sender.ID], m.Sender.ID, defaultBucket)
	confirmDisplayName(b, m)
}

func confirmDisplayName(b *tb.Bot, m *tb.Message) {
	log.Printf("confirmDisplayName: send confirm display name to user id=%d", m.Sender.ID)
	sendYesNof(b, m, "Would you like your display name to be \"%s\"?", getFullName(m))
}

func askDisplayName(b *tb.Bot, m *tb.Message) {
	log.Printf("askDisplayName: ask display name to user id=%d", m.Sender.ID)
	sendAndHideKeyboard(b, m, "What would you like your display name to be?")
}

func askTos(b *tb.Bot, m *tb.Message) {
	log.Printf("askTos: send term service to user id=%d", m.Sender.ID)
	sendYesNo(b, m, "Do you agree with our Term of Service? You could view the PDF version here https://home.kyber.network/assets/tac.pdf")
}

func askSubcription(b *tb.Bot, m *tb.Message) {
	log.Printf("askSubcription: ask user id=%d", m.Sender.ID)
	sendYesNo(b, m, "Would you like to receive important updates regarding your account?")
}

func boolToYesNo(value bool) string {
	if value {
		return "Yes"
	}

	return "No"
}

func doCreateAccount(b *tb.Bot, m *tb.Message) {
	user := userMap[m.Sender.ID]
	log.Printf("doCreateAccount: user id=%d name=%s subcribe=%s", m.Sender.ID, user.DisplayName, boolToYesNo(user.Subscription))
	text := fmt.Sprintf(
		"Hurrah! your account has been created!\n\nDisplay Name: %s\nTerm of Service: Agreed\nSubscribe to Updates: %s\n\nWould you like to sign-in Kyber Network now?",
		user.DisplayName,
		boolToYesNo(user.Subscription))

	sendYesNo(b, m, text)
}

func informSignin(b *tb.Bot, m *tb.Message) {
	log.Printf("informSignin: send inform msg to user id=%d", m.Sender.ID)
	send(b, m, "To sign-in Kyber Network, please type /signin")
}
