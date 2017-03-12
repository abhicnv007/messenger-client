package client

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

const (
	threadsURI       = "/threads"
	singleThreadURI  = "/threads/{threadID}"
	allMessagesURI   = "/threads/{threadID}/messages"
	singleMessageURI = "/threads/{threadID}/messages/{messageID}"
	userURI          = "/users"
)

//Link is a link
type Link struct {
	Href string `json:"href"`
}

//Message Type for storing IMs
type Message struct {
	Link
	ParentThread Link   `json:"threadid"`
	From         Link   `json:"from"`
	Content      string `json:"content"`
	Time         string `json:"time"`
}

//Thread represents a conversation
type Thread struct {
	Link
	Participants []Link `json:"participants"`
}

//AllThreads are just that
type AllThreads struct {
	Threads []Link `json:"threads"`
}

//User Defines a user
type User struct {
	Link
	Name   string `json:"name"`
	Secret string `json:"secret"`
}

var cache map[string]string

const (
	withAuth    = true
	withoutAuth = false
)

func addName(href string, name string) {
	cache[href] = name
}

func getName(href string) string {
	return cache[href]
}

var host string

var myUser struct {
	Name     string
	Pass     string
	Link     string
	UID      string
	Secret   string
	Threads  []Link
	LastTime time.Time
}

func getIDFromLink(link string) (int64, error) {
	s := strings.Split(link, "/")
	i, err := strconv.ParseInt(s[len(s)-1], 10, 64)
	if err != nil {
		return i, errors.New("Could not parse id")
	}
	return i, nil
}

func setMyUser(ru User) {
	myUser.Link = ru.Href
	s := strings.Split(ru.Href, "/")
	myUser.UID = s[len(s)-1]
	myUser.Secret = ru.Secret
}

var (
	errRequestFailed = errors.New("The request failed")
)

func readFromResponse(res *http.Response, dat interface{}, okStatus int) error {
	defer res.Body.Close()

	if res.StatusCode != okStatus {
		log.Println("[readFromResponse] Status not good, got ", res.Status)
		return errRequestFailed
	}

	if dat != nil {
		b, err := ioutil.ReadAll(res.Body)
		if err != nil {
			return err
		}
		json.Unmarshal(b, dat)
	}
	return nil
}

/*Checks if the name and password in myUser is a valid user, if is then sets its
secret and uid
*/
func checkUser() bool {
	c := &http.Client{}
	req, err := http.NewRequest("GET", host+userURI, nil)
	if err != nil {
		log.Println("[checkUser] new request error : ", err)
	}
	q := req.URL.Query()
	q.Add("name", myUser.Name)
	q.Add("password", myUser.Pass)
	req.URL.RawQuery = q.Encode()

	res, err := c.Do(req)
	if err != nil {
		log.Println("[checkUser] do error : ", err)
		return false
	}

	var ru User
	if err = readFromResponse(res, &ru, http.StatusOK); err != nil {
		return false
	}

	setMyUser(ru)

	return true
}

func login() {
	var name, pass string
	for {
		fmt.Print("Username:")
		fmt.Scanf("%s", &name)
		fmt.Print("Password:")
		fmt.Scanf("%s", &pass)

		myUser.Name = name
		myUser.Pass = pass

		if !checkUser() {
			fmt.Println("Cannot login, try again")
		} else {
			break
		}
	}

	fmt.Println("User Href : ", myUser.Link)
	loadAllThreads()
	showMainScreen()
}

func register() {
	var name, pass string
	for {
		fmt.Print("Username:")
		fmt.Scanf("%s", &name)
		fmt.Print("Password:")
		fmt.Scanf("%s", &pass)

		myUser.Name = name
		myUser.Pass = pass

		if !newUser() {
			fmt.Println("Cannot login, try again")
		} else {
			break
		}
	}
	fmt.Println("User Href : ", myUser.Link)
	showMainScreen()
}

func newUser() bool {

	res, err := requestPOST(userURI, []byte(`
	{
		"name":"`+myUser.Name+`",
		"password":"`+myUser.Pass+`"
	}`), withoutAuth)
	if err != nil {
		log.Println("[newUser] requestPOST error : ", err)
		return false
	}

	var ru User

	if err = readFromResponse(res, &ru, http.StatusCreated); err != nil {
		log.Println("[newUser] readFromResponse error : ", err)
		return false
	}

	setMyUser(ru)

	return true
}

func showMainScreen() {
	var i = 0

	for i != 3 {
		fmt.Println("1. Show current threads")
		fmt.Println("2. Create New Thread")
		fmt.Println("3. Exit")
		fmt.Scanf("%d", &i)

		switch i {
		case 1:
			selectThreads()
		case 2:
			createNewThread()
		case 3:
			break
		}
	}
}

func selectThreads() {
	var i int
	displayAllThreads()
	fmt.Println("Enter -1 to return : ")
	fmt.Scanf("%d", &i)
	if i == -1 {
		return
	}
	chat(myUser.Threads[i])
}

//Just display all threads from the user
func displayAllThreads() {
	for i, l := range myUser.Threads {
		fmt.Println(i, " : ", l)
	}
}

//get all threads that were created before the start
func loadAllThreads() {

	res, err := requestGET(threadsURI, withAuth)
	if err != nil {
		log.Println("[loadAllThreads] requestGET error: ", err)
		return
	}

	var rt AllThreads
	if err = readFromResponse(res, &rt, http.StatusOK); err != nil {
		log.Println("[loadAllThreads] readFromResponse error: ", err)
		return
	}

	myUser.Threads = rt.Threads
}

func createNewThread() {
	var s string
	var i int
	var t Thread
	fmt.Println("Enter number of participants excluding you")
	fmt.Scanf("%d", &i)
	for i > 0 {
		fmt.Println("Enter participants link")
		fmt.Scanf("%s", &s)
		t.Participants = append(t.Participants, Link{Href: s})
		i--
	}

	b, _ := json.Marshal(&t)

	res, err := requestPOST(threadsURI, b, withAuth)
	if err != nil {
		log.Println("[createNewThread] requestPOST error: ", err)
		return
	}

	var rt Thread
	if err = readFromResponse(res, &rt, http.StatusCreated); err != nil {
		log.Println("[createNewThread] readFromResponse error: ", err)
		return
	}

	myUser.Threads = append(myUser.Threads, rt.Link)
}

func getThreadParticipants(l Link) {

	res, err := requestGET(l.Href, withAuth)
	if err != nil {
		log.Println("[getThreadParticipants] requestGET error: ", err)
		return
	}

	var rt Thread
	if err = readFromResponse(res, &rt, http.StatusOK); err != nil {
		log.Println("[getThreadParticipants] readFromResponse error: ", err)
		return
	}

	for _, p := range rt.Participants {
		u := getUsername(p)
		fmt.Println(u)
		addName(p.Href, u)
	}
}

func getUsername(l Link) string {

	res, err := requestGET(l.Href, withAuth)
	if err != nil {
		log.Println("[getUsername] requestGET error: ", err)
		return ""
	}

	var ru User
	if err = readFromResponse(res, &ru, http.StatusOK); err != nil {
		log.Println("[getUsername] readFromResponse error: ", err)
		return ""
	}

	return ru.Name
}

func chat(l Link) {
	//ticker = time.NewTicker(1 * time.Second)
	//quit = make(chan struct{})

	getThreadParticipants(l)

	displayMessages(l, false)

	reader := bufio.NewReader(os.Stdin)
	go displayMessages(l, true)
	for {
		s, _ := reader.ReadString('\n')
		sendMessage(s, l)
	}
}

func displayMessages(l Link, justNew bool) {
	d, _ := time.ParseDuration("1s")
	ch := time.Tick(d)

	if !justNew {
		m := loadMessages(l, justNew)
		if len(m) > 0 {
			for _, ms := range m {
				if justNew {
					if ms.From.Href != myUser.Link {
						printMessage(ms)
					}
				} else {
					printMessage(ms)
				}
			}
			myUser.LastTime, _ = time.Parse(time.RFC3339, m[len(m)-1].Time)
		}
		return
	}

	for {
		select {
		case <-ch:
			m := loadMessages(l, justNew)
			if len(m) > 0 {
				for _, ms := range m {
					if justNew {
						if ms.From.Href != myUser.Link {
							printMessage(ms)
						}
					} else {
						printMessage(ms)
					}
				}
				myUser.LastTime, _ = time.Parse(time.RFC3339, m[len(m)-1].Time)
			}
		}
	}
}

func loadMessages(l Link, new bool) []Message {

	c := http.DefaultClient
	req, err := http.NewRequest("GET", host+l.Href+"/messages", nil)
	if err != nil {
		log.Println("[loadMessages] NewRequest error: ", err)
		return nil
	}
	req.SetBasicAuth(myUser.UID, myUser.Secret)

	if new {
		q := req.URL.Query()
		q.Add("time", myUser.LastTime.Format(time.RFC3339))
		req.URL.RawQuery = q.Encode()
	}

	res, err := c.Do(req)
	if err != nil {
		log.Println("[loadMessages] Do error: ", err)
		return nil
	}

	var m []Message
	if err = readFromResponse(res, &m, http.StatusOK); err != nil {
		log.Println("[loadMessage] readFromResponse")
		return nil
	}
	return m
}

func printMessage(ms Message) {
	fmt.Println(getName(ms.From.Href), "sent : ", ms.Content)
}

func sendMessage(con string, t Link) {
	var msg Message
	msg.Content = con
	msg.From.Href = myUser.Link
	msg.ParentThread = t
	msg.Time = time.Now().Format(time.RFC3339)

	b, _ := json.Marshal(&msg)

	res, err := requestPOST(t.Href+"/messages", b, withAuth)
	if err != nil {
		log.Println("[sendMessage] requestPOST error: ", err)
		return
	}

	if err = readFromResponse(res, nil, http.StatusCreated); err != nil {
		log.Println("[sendMessage] Could not create message")
		return
	}
}

func getMessages() {
}

func init() {
	flag.StringVar(&host, "host", "localhost:8080", "The host of the server")
	flag.Parse()
	if host[:4] != "http" {
		host = "http://" + host
	}
	cache = make(map[string]string)
}

func main() {

	var i int
	for {
		fmt.Println("1. Login")
		fmt.Println("2. Register")
		fmt.Println("3. Exit")
		fmt.Scanf("%d", &i)

		switch i {
		case 1:
			login()
		case 2:
			register()
		case 3:
			return
		}
	}
}
