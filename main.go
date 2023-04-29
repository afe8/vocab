package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"math/rand"
	"net"
	"net/http"
	"net/smtp"
	"os"
	"strings"
	"time"
)

type Word struct {
	Word       string
	WordTR     string
	Usage      string
	Note       string
	Definition string
}

var (
	collection *mongo.Collection
)

func main() {
	clientOptions := options.Client().ApplyURI(os.Getenv("DB_URI"))
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		log.Fatal(err)
	}
	//fmt.Println("Connected to MongoDB!")
	collection = client.Database("eng").Collection("vocab")
	listenAddr := os.Getenv("LISTEN_ADDR")
	addr := listenAddr + `:` + os.Getenv("PORT")
	http.HandleFunc("/send", send)
	//http.HandleFunc("/getwords", getWords)
	log.Printf("starting server at %s", addr)
	log.Fatal(http.ListenAndServe(addr, nil))
}

func getAllUnknownWords() []*Word {
	var words []*Word
	filterword := bson.D{{"unknowncount", bson.D{{"$gt", 0}}}}

	result, err := collection.Find(context.TODO(), filterword)
	if err != nil {
		log.Fatal(err)
	}

	for result.Next(context.TODO()) {
		var word Word
		err := result.Decode(&word)
		if err != nil {
			log.Fatal(err)
		}
		words = append(words, &word)
	}

	if err := result.Err(); err != nil {
		log.Fatal(err)
	}
	result.Close(context.TODO())
	return words
}

func send(w http.ResponseWriter, r *http.Request) {
	sendEmail(getMessageAndTitle())
	//sendNotification(getMessageAndTitle())
}

func sendEmail(title, message string) {
	from := os.Getenv("MAIL_FROM")
	password := os.Getenv("MAIL_TOKEN")

	to := []string{
		os.Getenv("MAIL_TO"),
	}

	smtpHost := os.Getenv("MAIL_SERVER")
	smtpPort := os.Getenv("MAIL_PORT")
	conn, err := net.Dial("tcp", smtpHost+":"+smtpPort)
	if err != nil {
		println(err)
	}

	c, err := smtp.NewClient(conn, smtpHost)
	if err != nil {
		println(err)
	}

	tlsconfig := &tls.Config{
		ServerName: smtpHost,
	}

	if err = c.StartTLS(tlsconfig); err != nil {
		println(err)
	}

	auth := LoginAuth(from, password)

	if err = c.Auth(auth); err != nil {
		println(err)
	}
	msg := fmt.Sprintf("Subject: %s\r\n\r\n"+
		"%s\r\n", strings.ToUpper(title[2:]), message)
	mail := []byte(msg)

	err = smtp.SendMail(smtpHost+":"+smtpPort, auth, from, to, mail)
	if err != nil {
		fmt.Println(err)
		return
	}
	fmt.Println("Email Sent Successfully!")
}

func sendNotification(title, message string) {
	fcmUrl := "https://fcm.googleapis.com/fcm/send"
	fmt.Println("Notification is sending")
	requestData := fmt.Sprintf("{\"to\": \"/topics/vocabulary\",\"notification\": {\"title\": \"%s\",\"body\": \"%s\"}}", strings.ToUpper(title[2:]), message)
	var jsonStr = []byte(requestData)
	req, err := http.NewRequest("POST", fcmUrl, bytes.NewBuffer(jsonStr))
	req.Header.Set("Authorization", os.Getenv("TOKEN"))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	fmt.Println("Response Status:", resp.Status)
}

func getMessageAndTitle() (string, string) {
	title := ""
	message := ""
	maxWord := 3
	words := getAllUnknownWords()
	for i := 0; i < maxWord; i++ {
		random := getRandomNumber(len(words))
		title = fmt.Sprintf("%s - %s", title, words[random].Word)
		note := ""
		if words[random].Usage != "" {
			note = words[random].Usage
		} else {
			note = words[random].Note
		}
		message = fmt.Sprintf("%s- %s | %s | %s\n", message, words[random].Word, words[random].WordTR, note)
	}
	return title, message
}

func getWords(w http.ResponseWriter, r *http.Request) {
	var list [][]string
	maxWord := 20
	words := getAllUnknownWords()
	for i := 0; i < maxWord; i++ {
		random := getRandomNumber(len(words))
		definition := ""
		if words[random].Definition != "" {
			definition = words[random].Definition
		} else {
			definition = words[random].WordTR
		}
		list = append(list, []string{words[random].Word, definition})
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(list)
}

func getRandomNumber(max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max)
}

type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		}
	}
	return nil, nil
}
