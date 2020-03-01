package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"github.com/barnybug/gogsmmodem"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/tarm/serial"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"
)

type App struct {
	Router     *mux.Router
	DB         *sql.DB
	conf       *Config
	Modem      *gogsmmodem.Modem
	SMSLog     *log.Logger
	SMSDevice  string
	SendSMSURL string
	LogSMSURL  string
	State      string
}

type Config struct {
	Port          string `json:"port"`
	Host          string `json:"host"`
	Token         string `json:"token"`
	DBConnect     string `json:"db_connect"`
	GatewayNumber string `json:"gateway_number"`
	WebURL        string `json:"weburl"`
	APIURL        string `json:"apiurl"`
	WHSecret      string `json:"whsecret"`
	SerialName    string `json:"serialname"`
	SerialBaud    int    `json:"serialbaud"`
}

type Tunnel struct {
	Name      string `json:"name"`
	Proto     string `json:"proto"`
	PublicURL string `json:"public_url"`
}

type Tunnels struct {
	Tunnels []Tunnel `json:"tunnels"`
	URI     string   `json:"uri"`
}

func (a *App) Initialize() {

	file, err := os.Open("./config.json")
	if err != nil {
		fmt.Println(err)
	}
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&a.conf)
	if err != nil {
		fmt.Println(err)
	}

	modemconf := serial.Config{Name: a.conf.SerialName, Baud: a.conf.SerialBaud}
	a.Modem, err = gogsmmodem.Open(&modemconf, true)
	if err != nil {
		panic(err)
	}

	smslogfile, err := os.OpenFile("sms.out", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalln("Failed to open log file", "sms.out", ":", err)
	}

	multi := io.MultiWriter(smslogfile, os.Stdout)

	a.SMSLog = log.New(multi,
		"SMSLOG: ",
		log.Ldate|log.Ltime|log.Lshortfile)

	li, _ := a.Modem.ListMessages("ALL")
	fmt.Printf("%d stored messages\n", len(*li))

	admin_num := "09876543210"

	a.SMSLog.Println("Started Up")
	//}

	a.SendSMSURL = GetSendSMSURL()
	a.LogSMSURL = GetLogSMSURL()
	a.Modem.SendMessage(admin_num, "Started Up With SendSMSURL: "+a.LogSMSURL)

	a.SMSLog.Println("WebHookUrl:", a.SendSMSURL)

	fmt.Println("APIURL: "+a.conf.APIURL, "SendSMSURL: "+a.SendSMSURL, "LogSMSURL: "+a.LogSMSURL)
	a.PostNGURL(a.conf.APIURL+"/setngurl", a.SendSMSURL)

	a.Router = mux.NewRouter()
	a.initializeRoutes()

}

func (a *App) initializeRoutes() {

	a.Router.HandleFunc("/log", a.DisplayLog).Methods("GET")
	a.Router.HandleFunc("/smslog", a.DisplaySMSLog).Methods("GET")
	a.Router.HandleFunc("/sendsms", a.SendSMS).Methods("POST")
	a.Router.Use(a.LogHandler)
}

func (a *App) SendSMS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		fmt.Println("Not Post")
		return
	}

	if r.Header.Get("Authorization") == "Bearer "+a.conf.Token {
		message := r.FormValue("message")
		receiver := r.FormValue("receiver")

		a.Modem.SendMessage(receiver, message)
		a.SMSLog.Println("To "+receiver+": ", message)
		a.UploadSMSLog(a.conf.APIURL, "SENT", receiver, sanitize(message))
		a.ReceiveMessages()
	} else {
		fmt.Println("Attempt to send UNsuccessful. ")
	}
}

func respondWithOK(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func (a *App) ReadMessages() {

	li, _ := a.Modem.ListMessages("REC UNREAD")
	for _, msg := range *li {
		a.SMSLog.Println("From "+msg.Telephone+": ", msg.Index, msg.Status, msg.Body)
		a.UploadSMSLog(a.conf.APIURL, "RECEIVED", msg.Telephone, sanitize(msg.Body))
		a.Modem.DeleteMessage(msg.Index)
		a.SMSLog.Println("Message: ", msg.Index, " deleted.")
	}
}

func (a *App) ReceiveMessages() {

	for packet := range a.Modem.OOB {
		a.SMSLog.Printf("Received: %#v\n", packet)
		switch p := packet.(type) {
		case gogsmmodem.MessageNotification:
			a.SMSLog.Println("Message notification:", p)
			a.ReadMessages()
		}
		return
	}
}

func (a *App) UploadSMSLog(api_url string, action string, address string, body string) {

	v := make(map[string]string)

	v["action"] = action
	v["address"] = address
	v["body"] = body

	s, _ := json.Marshal(v)
	fmt.Println("s: ", s)

	logsms_url := api_url + "/logsms"

	req, err := http.NewRequest("POST", logsms_url, bytes.NewBuffer(s))
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Add("Content-Type", "application/json")
	req.Header.Add("Authorization", "Bearer "+a.conf.Token)

	c := &http.Client{}
	resp, err := c.Do(req)

	if err != nil {
		fmt.Printf("http.Do() error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ioutil.ReadAll() error: %v\n", err)
		return
	}

	fmt.Printf("read resp.Body successfully:\n%v\n", string(data))

}

func (a *App) DisplayLog(w http.ResponseWriter, r *http.Request) {

	http.ServeFile(w, r, "nohup.out")
}

func (a *App) DisplaySMSLog(w http.ResponseWriter, r *http.Request) {

	http.ServeFile(w, r, "sms.out")
}

func GetSendSMSURL() string {

	var whurl string

	res, err := http.Get("http://localhost:4040/api/tunnels")

	if err != nil {
		fmt.Println(err)
	}
	output, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%s", output)

	tunnels := Tunnels{}

	err = json.Unmarshal(output, &tunnels)
	if err != nil {
		return "No_NG_Tunnel"
	}

	for i := 0; i < len(tunnels.Tunnels); i++ {
		if tunnels.Tunnels[i].Proto == "https" {
			whurl = tunnels.Tunnels[i].PublicURL + "/sendsms"
		} else {
			whurl = "NO_https_NG_Tunnel"
		}
	}

	fmt.Println(whurl)
	return whurl
}

func GetLogSMSURL() string {

	var whurl string

	res, err := http.Get("http://localhost:4040/api/tunnels")

	if err != nil {
		fmt.Println(err)
	}
	output, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%s", output)

	tunnels := Tunnels{}

	err = json.Unmarshal(output, &tunnels)
	if err != nil {
		return "No_NG_Tunnel"
	}

	for i := 0; i < len(tunnels.Tunnels); i++ {
		if tunnels.Tunnels[i].Proto == "https" {
			whurl = tunnels.Tunnels[i].PublicURL + "/smslog"
		} else {
			whurl = "NO_https_NG_Tunnel"
		}
	}

	fmt.Println(whurl)
	return whurl
}

func (a *App) PostNGURL(web_url string, ngurl string) {
	v := url.Values{}
	v.Set("ngurl", ngurl)

	s := v.Encode()

	req, err := http.NewRequest("POST", web_url, strings.NewReader(s))
	if err != nil {
		fmt.Println(err)
	}

	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Bearer "+a.conf.Token)

	c := &http.Client{}
	resp, err := c.Do(req)

	if err != nil {
		fmt.Printf("http.Do() error: %v\n", err)
		return
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("ioutil.ReadAll() error: %v\n", err)
		return
	}

	fmt.Printf("read resp.Body successfully:\n%v\n", string(data))

}

func (a *App) LogHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(time.Now().Format("2006-01-02 03:04:05 PM"), r.RemoteAddr, r.Method, r.URL)
		next.ServeHTTP(w, r)
	})
}

func (a *App) Run(addr string) {
	fmt.Println(time.Now().Format("2006-01-02 03:04:05 PM"), "Running HTTP on port "+a.conf.Port)
	log.Fatal(http.ListenAndServe(":"+a.conf.Port, a.Router))
}

func sanitize(str string) string {
	raw_str := "`" + str + "`"

	reg, err := regexp.Compile("[^a-zA-Z0-9 .,!()$#/:{}*@\\[\\]]+")
	if err != nil {
		log.Fatal(err)
	}
	processedString := reg.ReplaceAllString(raw_str, "")

	return processedString
}
