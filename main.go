package main

import (
	"bytes"
	"context"
	"fmt"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"math/rand"
	"net/http"
	"os"
	"time"
)

type Word struct {
	Word               string
	WordTR             string
	Note               string
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
	http.HandleFunc("/send", sendNotification)
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

func sendNotification(w http.ResponseWriter, r *http.Request) {
	fcmUrl := "https://fcm.googleapis.com/fcm/send"

	fmt.Println("Notification is sending")
	title := fmt.Sprintf("Vocabulary")
	message := ""
	maxWord := 3
	words := getAllUnknownWords()
	for i := 0; i < maxWord; i++ {
		random := getRandomNumber(len(words))
		message = fmt.Sprintf("%s- %s | %s | %s\n", message, words[random].Word, words[random].WordTR, words[random].Note)
	}
	requestData := fmt.Sprintf("{\"to\": \"/topics/vocabulary\",\"notification\": {\"title\": \"%s\",\"body\": \"%s\"}}", title, message)
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

func getRandomNumber(max int) int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(max)
}
