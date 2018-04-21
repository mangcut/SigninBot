package main

import (
	"fmt"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"

	tb "gopkg.in/tucnak/telebot.v2"
)

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
	displayName       string
	tosAgreed         bool
	subscription      bool
	registrationStep  Step
	lastSigninRequest time.Time
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

var userMap = make(map[int]*userInfo)

func main() {
	b, err := tb.NewBot(tb.Settings{
		Token:  os.Getenv("SIGNIN_BOT_TOKEN"),
		Poller: &tb.LongPoller{Timeout: 10 * time.Second},
	})

	if err != nil {
		log.Fatal(err)
		return
	}

	b.Handle(tb.OnText, func(m *tb.Message) {
		handleReply(b, m)
	})

	b.Handle("/start", func(m *tb.Message) {
		next(b, m)
	})

	b.Handle("/signin", func(m *tb.Message) {
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
	if user, ok := userMap[m.Sender.ID]; ok {
		if user.displayName != "" {
			return user.displayName
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
		funcArray := []func(*tb.Bot, *tb.Message){
			confirmDisplayName,
			askDisplayName,
			askTos,
			askSubcription,
			doCreateAccount,
			sendSigninLink}
		funcArray[user.registrationStep](b, m)
	} else {
		// registration
		startRegistration(b, m)
	}
}

func sendSigninLink(b *tb.Bot, m *tb.Message) {
	user := userMap[m.Sender.ID]
	last := user.lastSigninRequest
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
		user.lastSigninRequest = time.Now()
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
		switch user.registrationStep {
		case stepToConfirmFulName:
			if isYes(m.Text) {
				user.displayName = fmt.Sprintf("%s %s", m.Sender.FirstName, m.Sender.LastName)
				user.registrationStep = stepToAskTos
				next(b, m)
			} else if isNo(m.Text) {
				user.registrationStep = stepToAskFullName
				next(b, m)
			} else {
				next(b, m)
			}
		case stepToAskFullName:
			user.displayName = strings.Title(strings.TrimSpace(m.Text))
			user.registrationStep = stepToAskTos
			next(b, m)
		case stepToAskTos:
			if isYes(m.Text) {
				user.tosAgreed = true
				user.registrationStep = stepToAskSubscription
				next(b, m)
			} else {
				next(b, m)
			}
		case stepToAskSubscription:
			if isYes(m.Text) {
				user.subscription = true
				user.registrationStep = stepToCreateAcount
				next(b, m)
			} else if isNo(m.Text) {
				user.subscription = false
				user.registrationStep = stepToCreateAcount
				next(b, m)
			} else {
				next(b, m)
			}
		case stepToCreateAcount:
			// TODO: should done earlier, from the time acount created
			user.registrationStep = stepDone
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
	newUserInfo := userInfo{registrationStep: stepToConfirmFulName}
	userMap[m.Sender.ID] = &newUserInfo

	confirmDisplayName(b, m)
}

func confirmDisplayName(b *tb.Bot, m *tb.Message) {
	sendYesNof(b, m, "Would you like your display name to be \"%s\"?", getFullName(m))
}

func askDisplayName(b *tb.Bot, m *tb.Message) {
	sendAndHideKeyboard(b, m, "What would you like your display name to be?")
}

func askTos(b *tb.Bot, m *tb.Message) {
	sendYesNo(b, m, "Do you agree with our Term of Service? You could view the PDF version here https://home.kyber.network/assets/tac.pdf")
}

func askSubcription(b *tb.Bot, m *tb.Message) {
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
	text := fmt.Sprintf(
		"Hurrah! your account has been created!\n\nDisplay Name: %s\nTerm of Service: Agreed\nSubscribe to Updates: %s\n\nWould you like to sign-in Kyber Network now?",
		user.displayName,
		boolToYesNo(user.subscription))

	sendYesNo(b, m, text)
}

func informSignin(b *tb.Bot, m *tb.Message) {
	send(b, m, "To sign-in Kyber Network, please type /signin")
}
