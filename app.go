package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	//"github.com/aws/aws-sdk-go/aws"
	//"github.com/aws/aws-sdk-go/aws/awserr"
	//"github.com/aws/aws-sdk-go/aws/session"
	//"github.com/aws/aws-sdk-go/service/ses"
	"github.com/barnybug/gogsmmodem"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/tarm/serial"
	//sms "github.com/peddlrph/lib/smsgateway"
	"bytes"
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

//func (a *App) Initialize(user, password, dbname string) {
func (a *App) Initialize() {

	//var config Config

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

	//SMSLog.Println("Hello")

	//f, err := os.OpenFile("messages", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	//if err != nil {
	//	log.Fatal(err)
	//}

	//defer f.Close()

	li, _ := a.Modem.ListMessages("ALL")
	fmt.Printf("%d stored messages\n", len(*li))
	//for _, msg := range *li {
	//	fmt.Println(string(msg.Index), msg.Status, msg.Telephone, msg.Body)
	//	str := strconv.Itoa(msg.Index) + "|" + msg.Status + "|" + msg.Telephone + "|" + msg.Timestamp.Format("2006-01-02 15:04:05") + "|" + msg.Body + "\n"
	//	_, _ = f.WriteString(str)
	//}

	admin_num := "09176530409"

	//a.Modem.SendMessage(admin_num, "Started Up")
	a.SMSLog.Println("Started Up")
	//}

	//fmt.Println(a.conf)

	//fmt.Println(len(os.Args))
	//a.SMSDevice = os.Args[1]
	//fmt.Println("SMS Device:", a.SMSDevice)

	//a.WebHookURL = "https://ngurlxx.ngrok.com/status"
	// go a.SMSStatusChecker(a.SMSDevice)

	a.SendSMSURL = GetSendSMSURL()
	a.LogSMSURL = GetLogSMSURL()
	a.Modem.SendMessage(admin_num, "Started Up With SendSMSURL: "+a.LogSMSURL)

	a.SMSLog.Println("WebHookUrl:", a.SendSMSURL)

	// SendNGURL(a.SMSDevice, admin_num, a.WebHookURL)
	//mesg := "Webhook URL: " + a.WebHookURL + "\n\nTime Stamp: " + time.Now().Format("2006-01-02 03:04:05 PM")
	//var recipients []string = []string{"peddler@cloudpeddler.com", "dondonvergara@yahoo.com"}
	//sendmail("peddler@cloudpeddler.com", recipients, "New WebHook URL", mesg)
	fmt.Println("APIURL: "+a.conf.APIURL, "SendSMSURL: "+a.SendSMSURL, "LogSMSURL: "+a.LogSMSURL)
	a.PostNGURL(a.conf.APIURL+"/setngurl", a.SendSMSURL)
	//connectionString := a.conf.DBConnect
	//a.DB, err = sql.Open("mysql", connectionString)
	//if err != nil {
	//	fmt.Println(err)
	//}

	a.Router = mux.NewRouter()
	a.initializeRoutes()

}

func (a *App) initializeRoutes() {
	//a.Router.HandleFunc("/message/last", a.getLastMessage).Methods("GET")
	//a.Router.HandleFunc("/messages", a.getMessages).Methods("GET")
	//a.Router.HandleFunc("/", a.HelloWorld).Methods("GET")
	//a.Router.HandleFunc("/message", a.createMessage).Methods("POST")
	//a.Router.HandleFunc("/messages", a.createMessages).Methods("POST")
	//a.Router.HandleFunc("/message/{id:[0-9]+}", a.getMessage).Methods("GET")
	//a.Router.Use(loggingMiddleware)
	//a.Router.HandleFunc("/status", a.postPaymentStatus).Methods("POST")
	//a.Router.HandleFunc("/status", a.getLastState).Methods("GET")
	//a.Router.Use(a.AuthHandler)
	a.Router.HandleFunc("/log", a.DisplayLog).Methods("GET")
	a.Router.HandleFunc("/smslog", a.DisplaySMSLog).Methods("GET")
	a.Router.HandleFunc("/sendsms", a.SendSMS).Methods("POST")
	a.Router.Use(a.LogHandler)
}

func (a *App) SendSMS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		//			tmpl.Execute(w, nil)
		fmt.Println("Not Post")
		return
	}

	if r.Header.Get("Authorization") == "Bearer "+a.conf.Token {
		//fmt.Println(r.Body)
		//dat := []byte(r.FormValue("status"))
		//dat := []byte("HELLO2")
		//err := ioutil.WriteFile("./status.txt", dat, 0644)
		//check(err)
		message := r.FormValue("message")
		receiver := r.FormValue("receiver")

		a.Modem.SendMessage(receiver, message)
		a.SMSLog.Println("To "+receiver+": ", message)
		a.UploadSMSLog(a.conf.APIURL, "SENT", receiver, sanitize(message))
		a.ReceiveMessages()
		//
		//_, _ = a.Modem.ListMessages("REC UNREAD")

		//mesg := "Status has changed to " + status + "\nTime Stamp: " + time.Now().Format("2006-01-02 03:04:05 PM")

		//sendmail("peddler@cloudpeddler.com", recipients, "Status: ", mesg)

		//fmt.Println("Set Status: ", status)
	} else {
		fmt.Println("Attempt to send UNsuccessful. ")
	}
	//stat := status{"status": "OK"}
	//respondWithOK(w, http.StatusCreated, stat)
}

func respondWithOK(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func (a *App) ReadMessages() {

	// time.Sleep(1 * time.Minute)
	li, _ := a.Modem.ListMessages("REC UNREAD")
	//fmt.Printf("%d stored messages\n", len(*li))
	for _, msg := range *li {
		a.SMSLog.Println("From "+msg.Telephone+": ", msg.Index, msg.Status, msg.Body)
		a.UploadSMSLog(a.conf.APIURL, "RECEIVED", msg.Telephone, sanitize(msg.Body))
		//str := strconv.Itoa(msg.Index) + "|" + msg.Status + "|" + msg.Telephone + "|" + msg.Timestamp.Format("2006-01-02 15:04:05") + "|" + msg.Body + "\n"
		//_, _ = f.WriteString(str)
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
			//msg, err := a.Modem.GetMessage(p.Index)
			//if err == nil {
			//	a.SMSLog.Printf("Message from %s: %s\n", msg.Telephone, msg.Body)
			//	a.Modem.DeleteMessage(p.Index)
			//}
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
	//fmt.Println("v: ", v)
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

	//state := "Hello World"
	//fmt.Println(state)
	//fmt.Println("Hello")
	//out,err := exec.Command("tail -20 C:/Users/vergarad/Documents/personal/peddlrph/testbed/modempool/nohup.out").Output()

	http.ServeFile(w, r, "nohup.out")
	//respondWithText(w, http.StatusOK, state)
}

func (a *App) DisplaySMSLog(w http.ResponseWriter, r *http.Request) {

	//state := "Hello World"
	//fmt.Println(state)
	//fmt.Println("Hello")
	//out,err := exec.Command("tail -20 C:/Users/vergarad/Documents/personal/peddlrph/testbed/modempool/nohup.out").Output()

	http.ServeFile(w, r, "sms.out")
	//respondWithText(w, http.StatusOK, state)
}

func GetSendSMSURL() string {
	//url := "http://" + device_ip + ":8080/v1/sms/"

	var whurl string

	//res, err := http.Post(send/?phone=" + gateway_num + "&message=" + command)
	res, err := http.Get("http://localhost:4040/api/tunnels")

	//res := [{"name":"command_line","uri":"/api/tunnels/command_line","public_url":"https://c027894d.ngrok.io","proto":"https","config":{"addr":"localhost:8080","inspect":true},"metrics":{"conns":{"count":0,"gauge":0,"rate1":0,"rate5":0,"rate15":0,"p50":0,"p90":0,"p95":0,"p99":0},"http":{"count":0,"rate1":0,"rate5":0,"rate15":0,"p50":0,"p90":0,"p95":0,"p99":0}}},{"name":"command_line (http)","uri":"/api/tunnels/command_line+%28http%29","public_url":"http://c027894d.ngrok.io","proto":"http","config":{"addr":"localhost:8080","inspect":true},"metrics":{"conns":{"count":0,"gauge":0,"rate1":0,"rate5":0,"rate15":0,"p50":0,"p90":0,"p95":0,"p99":0},"http":{"count":0,"rate1":0,"rate5":0,"rate15":0,"p50":0,"p90":0,"p95":0,"p99":0}}}],"uri":"/api/tunnels"}
	//res, err := http.Get("http://" + device_ip + ":8080/v1/sms/send/?phone=" + gateway_num + "&message=" + command)
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
	// whurl, _ := json.Marshal(output)
	return whurl
}

func GetLogSMSURL() string {
	//url := "http://" + device_ip + ":8080/v1/sms/"

	var whurl string

	//res, err := http.Post(send/?phone=" + gateway_num + "&message=" + command)
	res, err := http.Get("http://localhost:4040/api/tunnels")

	//res := [{"name":"command_line","uri":"/api/tunnels/command_line","public_url":"https://c027894d.ngrok.io","proto":"https","config":{"addr":"localhost:8080","inspect":true},"metrics":{"conns":{"count":0,"gauge":0,"rate1":0,"rate5":0,"rate15":0,"p50":0,"p90":0,"p95":0,"p99":0},"http":{"count":0,"rate1":0,"rate5":0,"rate15":0,"p50":0,"p90":0,"p95":0,"p99":0}}},{"name":"command_line (http)","uri":"/api/tunnels/command_line+%28http%29","public_url":"http://c027894d.ngrok.io","proto":"http","config":{"addr":"localhost:8080","inspect":true},"metrics":{"conns":{"count":0,"gauge":0,"rate1":0,"rate5":0,"rate15":0,"p50":0,"p90":0,"p95":0,"p99":0},"http":{"count":0,"rate1":0,"rate5":0,"rate15":0,"p50":0,"p90":0,"p95":0,"p99":0}}}],"uri":"/api/tunnels"}
	//res, err := http.Get("http://" + device_ip + ":8080/v1/sms/send/?phone=" + gateway_num + "&message=" + command)
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
	// whurl, _ := json.Marshal(output)
	return whurl
}

func (a *App) PostNGURL(web_url string, ngurl string) {
	//url := "http://" + device_ip + ":8080/v1/sms/"

	//res, err := http.Post(send/?phone=" + gateway_num + "&message=" + command)

	//postdata,err := ioutil.W
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

	//output, err := ioutil.ReadAll(res.Body)
	//res.Body.Close()
	//if err != nil {
	//	log.Fatal(err)
	//}
	//fmt.Printf("%s", output)
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

	//`�@oh!�@the Globe Rewa" "rds ' ' app via http://glbe.co/GRewardsApp to know the list of items you can redeem and partner stores where you can use your points as cash.`

	// Make a Regex to say we only want letters and numbers
	reg, err := regexp.Compile("[^a-zA-Z0-9 .,!()$#/:{}*@\\[\\]]+")
	if err != nil {
		log.Fatal(err)
	}
	processedString := reg.ReplaceAllString(raw_str, "")

	return processedString
}

/*
func (a *App) SMSStatusChecker(sms_device string) {
	for {
		if sms.CheckStatus(sms_device) == "ready" {
			a.State = "ONLINE"
		} else {
			a.State = "OFFLINE"
		}
		//fmt.Println("Infinite Loop 1: ", a.State)
		fmt.Println(time.Now().Format("2006-01-02 03:04:05 PM"), a.State)
		time.Sleep(time.Second * 60)
	}
}

func SendNGURL(device_ip string, admin_num string, url string) {
	//url := "http://" + device_ip + ":8080/v1/sms/"

	//res, err := http.Post(send/?phone=" + gateway_num + "&message=" + command)

	res, err := http.Get("http://" + device_ip + ":8080/v1/sms/send/?phone=" + admin_num + "&message=" + url)
	if err != nil {
		fmt.Println(err)
	}
	output, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%s", output)
}

func (a *App) HelloWorld(w http.ResponseWriter, r *http.Request) {

	state := "Hello World"
	fmt.Println(state)

	respondWithJSON(w, http.StatusOK, state)
}

func (a *App) postPaymentStatus(w http.ResponseWriter, r *http.Request) {
	var ps paymentstatus
	//var button_data buttondata
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&ps); err != nil {
		fmt.Println("Invalid request payload")
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	if ps.Secret != a.conf.WHSecret {
		fmt.Println("Invalid request secret")
		respondWithError(w, http.StatusBadRequest, "Invalid request secret")
		return
	}
	defer r.Body.Close()

	//fmt.Println(r.Body)

	fmt.Println(ps.Payment.Id, ps.Payment.ButtonData, ps.Payment.Amount, ps.Payment.Status)

	//if err := ps.postPaymentStatus(a.DB); err != nil {
	//	respondWithError(w, http.StatusInternalServerError, err.Error())
	//	return
	//}

	//bdata, err := ioutil.ReadAll(ps.Payment.ButtonData)

	// fmt.Printf("ButtonData type: %T : %v\n", ps.Payment.ButtonData, ps.Payment.ButtonData)

	button_data := buttondata{}

	err := json.Unmarshal([]byte(ps.Payment.ButtonData), &button_data)

	//decoder2 := json.NewDecoder(strings.NewReader(ps.Payment.ButtonData))
	//if err := decoder2.Decode(&button_data);
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid message payload")
		return
	}

	fmt.Printf("button_data: %v %v \n", button_data.MobileNum, button_data.Product)

	if ps.Payment.Status != "RECEIVED" { // Ignores all other payment status
		return
	}

	// call api to create transaction here with ps.Payment.Id, button_data.MobileNum and button_data.Product

	mesg := "Mobile Number: " + button_data.MobileNum + "\nProduct: " + button_data.Product + "\nPayment Status: " + ps.Payment.Status + "\nTime Stamp: " + time.Now().Format("2006-01-02 03:04:05 PM")

	var recipients []string = []string{"peddler@cloudpeddler.com", "dondonvergara@yahoo.com"}
	sendmail("peddler@cloudpeddler.com", recipients, "Payment", mesg)

	lmcommand := getLoadCommand(button_data.MobileNum, button_data.Product)

	fmt.Printf("Command Sent to %v: %v\n", a.conf.GatewayNumber, lmcommand)

	if button_data.MobileNum == "09189902085" {
		// loadWallet(a.SMSDevice, "09176530409", lmcommand)
		fmt.Println("Send SMS to admin")
	} else {
		// loadWallet(a.SMSDevice, a.conf.GatewayNumber, lmcommand)
		fmt.Println("Load to ", button_data.MobileNum)
	}
	respondWithJSON(w, http.StatusCreated, ps)
}

func (a *App) SMSStatusChecker(sms_device string) {
	for {
		if sms.CheckStatus(sms_device) == "ready" {
			a.State = "ONLINE"
		} else {
			a.State = "OFFLINE"
		}
		//fmt.Println("Infinite Loop 1: ", a.State)
		fmt.Println(time.Now().Format("2006-01-02 03:04:05 PM"), a.State)
		time.Sleep(time.Second * 60)
	}
}

func (a *App) getLastState(w http.ResponseWriter, r *http.Request) {

	state := a.State
	//if err := m.getLastMessage(a.DB); err != nil {
	//	fmt.Println(err)
	//	switch err {
	//	case sql.ErrNoRows:
	//		//respondWithError(w, http.StatusNotFound, "id:0")
	//		m = message{Id: "0"}
	//		respondWithJSON(w, http.StatusOK, m)
	//	default:
	//		respondWithError(w, http.StatusInternalServerError, err.Error())
	//	}
	//	return
	//}
	fmt.Println(state)

	respondWithJSON(w, http.StatusOK, state)
}



func sendmail(sender string, recipients []string, topic string, message string) {
	// Create a new session in the us-west-2 region.
	// Replace us-west-2 with the AWS Region you're using for Amazon SES.
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)

	CharSet := "UTF-8"

	// Create an SES session.
	svc := ses.New(sess)

	// Assemble the email.
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses: []*string{},
			ToAddresses: aws.StringSlice(recipients),
			//			ToAddresses: []*string{
			//				aws.String(recipient),
			//			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				//	Html: &ses.Content{
				//		Charset: aws.String(CharSet),
				//		Data:    aws.String(HtmlBody),
				//	},
				Text: &ses.Content{
					Charset: aws.String(CharSet),
					Data:    aws.String(message),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(CharSet),
				Data:    aws.String(topic),
			},
		},
		Source: aws.String(sender),
		// Uncomment to use a configuration set
		//ConfigurationSetName: aws.String(ConfigurationSet),
	}

	// Attempt to send the email.
	result, err := svc.SendEmail(input)

	// Display error messages if they occur.
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ses.ErrCodeMessageRejected:
				fmt.Println(ses.ErrCodeMessageRejected, aerr.Error())
			case ses.ErrCodeMailFromDomainNotVerifiedException:
				fmt.Println(ses.ErrCodeMailFromDomainNotVerifiedException, aerr.Error())
			case ses.ErrCodeConfigurationSetDoesNotExistException:
				fmt.Println(ses.ErrCodeConfigurationSetDoesNotExistException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}

		return
	}

	fmt.Println("Email Sent to address: "+recipients[0], recipients[1])
	fmt.Println(result)

}



func (a *App) Run(addr string) {
	fmt.Println(time.Now().Format("2006-01-02 03:04:05 PM"), "Running HTTP on port "+a.conf.Port)
	log.Fatal(http.ListenAndServe(":"+a.conf.Port, a.Router))
}



func (a *App) AuthHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer "+a.conf.Token {
			next.ServeHTTP(w, r)
		} else {
			fmt.Println("Not Authorized" + r.Header.Get("Authorization"))
		}
	})
}

func (a *App) SignatureHandler(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "Bearer "+a.conf.Token {
			next.ServeHTTP(w, r)
		} else {
			fmt.Println("Not Authorized" + r.Header.Get("Authorization"))
		}
	})
}

func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.RequestURI)
		next.ServeHTTP(w, r)
	})
}

func loadWallet(device_ip string, gateway_num string, command string) {
	//url := "http://" + device_ip + ":8080/v1/sms/"

	//res, err := http.Post(send/?phone=" + gateway_num + "&message=" + command)

	res, err := http.Get("http://" + device_ip + ":8080/v1/sms/send/?phone=" + gateway_num + "&message=" + command)
	if err != nil {
		fmt.Println(err)
	}
	output, err := ioutil.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Printf("%s", output)

	//mesg := "Gateway Number: " + gateway_num + "\nCommand: " + command + "\nLoad Status: " + string(output) + "\nTime Stamp: " + time.Now().Format("2006-01-02 03:04:05 PM")

	//var recipients []string = []string{"peddler@cloudpeddler.com", "dondonvergara@yahoo.com"}
	//sendmail("peddler@cloudpeddler.com", recipients, "Load", mesg)
}



func (a *App) getLastMessage(w http.ResponseWriter, r *http.Request) {

	m := message{}
	if err := m.getLastMessage(a.DB); err != nil {
		fmt.Println(err)
		switch err {
		case sql.ErrNoRows:
			//respondWithError(w, http.StatusNotFound, "id:0")
			m = message{Id: "0"}
			respondWithJSON(w, http.StatusOK, m)
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}
	fmt.Println(m)

	respondWithJSON(w, http.StatusOK, m)
}

func (a *App) getMessage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	//id, err := strconv.Atoi(vars["id"])
	id := vars["id"]
	//if err != nil {
	//	respondWithError(w, http.StatusBadRequest, "Invalid message ID")
	//	return
	//}

	m := message{Id: id}
	if err := m.getMessage(a.DB); err != nil {
		switch err {
		case sql.ErrNoRows:
			respondWithError(w, http.StatusNotFound, "Message not found")
		default:
			respondWithError(w, http.StatusInternalServerError, err.Error())
		}
		return
	}

	respondWithJSON(w, http.StatusOK, m)
}

func (a *App) getMessages(w http.ResponseWriter, r *http.Request) {
	count, _ := strconv.Atoi(r.FormValue("count"))
	start, _ := strconv.Atoi(r.FormValue("start"))

	if count > 10 || count < 1 {
		count = 10
	}
	if start < 0 {
		start = 0
	}

	messages, err := getMessages(a.DB, start, count)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusOK, messages)
}

func (a *App) createMessage(w http.ResponseWriter, r *http.Request) {
	var m message
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&m); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	if err := m.createMessage(a.DB); err != nil {
		respondWithError(w, http.StatusInternalServerError, err.Error())
		return
	}

	respondWithJSON(w, http.StatusCreated, m)
}

func (a *App) createMessages(w http.ResponseWriter, r *http.Request) {
	var m []message
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&m); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid request payload")
		return
	}
	defer r.Body.Close()

	for i := 0; i < len(m); i++ {
		if err := m[i].createMessage(a.DB); err != nil {
			respondWithError(w, http.StatusInternalServerError, err.Error())
			return
		}
	}

	respondWithJSON(w, http.StatusCreated, m)
}

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, _ := json.Marshal(payload)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write(response)
}

func getLoadCommand(mobile_num string, product string) string {
	switch product {
	case "SmartLoad10":
		return mobile_num + "%20" + "10"
	case "SmartLoad15":
		return mobile_num + "%20" + "15"
	case "SmartLoad20":
		return mobile_num + "%20" + "20"
	case "SmartLoad30":
		return mobile_num + "%20" + "30"
	case "SmartLoad50":
		return mobile_num + "%20" + "50"
	case "SmartLoad60":
		return mobile_num + "%20" + "60"
	case "SmartLoad100":
		return mobile_num + "%20" + "100"
	case "SmartLoad115":
		return mobile_num + "%20" + "115"
	case "SmartLoad200":
		return mobile_num + "%20" + "200"
	case "SmartLoad250":
		return mobile_num + "%20" + "250"
	case "SmartLoad300":
		return mobile_num + "%20" + "300"
	case "SmartLoad500":
		return mobile_num + "%20" + "P500"
	case "SmartLoad1000":
		return mobile_num + "%20" + "P1000"
	case "SmartTalk100":
		return mobile_num + "%20" + "ST100"
	case "AllText10":
		return mobile_num + "%20" + "10"
	case "AllText20":
		return mobile_num + "%20" + "C20"
	case "AllText30":
		return mobile_num + "%20" + "AT30"
	case "AllOutSurf50":
		return mobile_num + "%20" + "ALLOUT50"
	case "AllOutSurf99":
		return mobile_num + "%20" + "ALLOUT99"
	case "Sakto20":
		return mobile_num + "%20" + "SAKTO20"
	case "Allin25":
		return mobile_num + "%20" + "AI25"
	case "Allin99":
		return mobile_num + "%20" + "ALLIN99"
	case "MegaAllin250":
		return mobile_num + "%20" + "AI250"
	case "UnliCall/Text25":
		return mobile_num + "%20" + "UCT25"
	case "UnliCall/Text30":
		return mobile_num + "%20" + "UCT30"
	case "UnliCall/Text50":
		return mobile_num + "%20" + "UCT50"
	case "UnliCall/Text100":
		return mobile_num + "%20" + "UCT100"
	case "UnliCall/Text350":
		return mobile_num + "%20" + "UCT350"
	case "GigaSurf50":
		return mobile_num + "%20" + "GIGA50"
	case "GigaSurf99":
		return mobile_num + "%20" + "GIGA99"
	case "GigaSurf299":
		return mobile_num + "%20" + "GIGA299"
	case "GigaSurf799":
		return mobile_num + "%20" + "GIGA799"
	case "SurfMax50":
		return mobile_num + "%20" + "SURFMAX50"
	case "SurfMax85":
		return mobile_num + "%20" + "SURFMAX85"
	case "SurfMax200":
		return mobile_num + "%20" + "SURFMAX200"
	case "SurfMax250":
		return mobile_num + "%20" + "SURFMAX250"
	case "SurfMax500":
		return mobile_num + "%20" + "SURFMAX500"
	case "SurfMax995":
		return mobile_num + "%20" + "SURFMAX995"
	case "BigBytes5":
		return mobile_num + "%20" + "BIG5"
	case "BigBytes10":
		return mobile_num + "%20" + "BIG10"
	case "BigBytes15":
		return mobile_num + "%20" + "BIG15"
	case "BigBytes30":
		return mobile_num + "%20" + "BIG30"
	case "BigBytes70":
		return mobile_num + "%20" + "BIG70"
	case "LahatText20":
		return mobile_num + "%20" + "L20"
	case "LahatText30":
		return mobile_num + "%20" + "L30"
	case "TNTLoad5":
		return mobile_num + "%20" + "5"
	case "TNTLoad10":
		return mobile_num + "%20" + "10"
	case "TNTLoad15":
		return mobile_num + "%20" + "15"
	case "TNTLoad20":
		return mobile_num + "%20" + "20"
	case "TNTLoad30":
		return mobile_num + "%20" + "30"
	case "TNTLoad50":
		return mobile_num + "%20" + "50"
	case "TNTLoad60":
		return mobile_num + "%20" + "60"
	case "TNTLoad100":
		return mobile_num + "%20" + "100"
	case "TNTLoad115":
		return mobile_num + "%20" + "115"
	case "TNTLoad200":
		return mobile_num + "%20" + "200"
	case "TNTLoad250":
		return mobile_num + "%20" + "250"
	case "TNTLoad300":
		return mobile_num + "%20" + "300"
	case "TNTLoad500":
		return mobile_num + "%20" + "P500"
	case "TNTLoad1000":
		return mobile_num + "%20" + "P1000"
	case "TNTFB10":
		return mobile_num + "%20" + "FB10"
	case "TNTUTP15":
		return mobile_num + "%20" + "UTP15"
	case "TNTT20":
		return mobile_num + "%20" + "T20"
	case "TNTSC20":
		return mobile_num + "%20" + "SC20"
	case "TNTSC30":
		return mobile_num + "%20" + "SC30"
	case "UnliTalkPlus15":
		return mobile_num + "%20" + "T15"
	case "UnliTalkPlus20":
		return mobile_num + "%20" + "TP20"
	case "UnliTalkPlus100":
		return mobile_num + "%20" + "T100"
	case "UnliTextPlus10":
		return mobile_num + "%20" + "TP10"
	case "GaanText10":
		return mobile_num + "%20" + "T10"
	case "GaanText20":
		return mobile_num + "%20" + "GT20"
	case "UnliTextExtra30":
		return mobile_num + "%20" + "U30"
	case "UnliTextExtra150":
		return mobile_num + "%20" + "U150"
	case "GaanUnliTxtPlus15":
		return mobile_num + "%20" + "GU15"
	case "GaanUnlitxtPlus30":
		return mobile_num + "%20" + "GU30"
	case "UnliAllNetText15":
		return mobile_num + "%20" + "UAT15"
	case "UnliAllNetText30":
		return mobile_num + "%20" + "UAT30"
	case "UnliTexttoAll20":
		return mobile_num + "%20" + "UA20"
	case "UnliTexttoAll300":
		return mobile_num + "%20" + "UA300"
	case "AutoloadMAX10":
		return mobile_num + "%20" + "10"
	case "AutoloadMAX15":
		return mobile_num + "%20" + "15"
	case "AutoloadMAX20":
		return mobile_num + "%20" + "20"
	case "AutoloadMAX30":
		return mobile_num + "%20" + "30"
	case "AutoloadMAX50":
		return mobile_num + "%20" + "50"
	case "AutoloadMAX60":
		return mobile_num + "%20" + "60"
	case "AutoloadMAX100":
		return mobile_num + "%20" + "100"
	case "AutoloadMAX115":
		return mobile_num + "%20" + "115"
	case "AutoloadMAX150":
		return mobile_num + "%20" + "150"
	case "AutoloadMAX350":
		return mobile_num + "%20" + "P350"
	case "AutoloadMAX450":
		return mobile_num + "%20" + "P450"
	case "AutoloadMAX550":
		return mobile_num + "%20" + "P550"
	case "AutoloadMAX650":
		return mobile_num + "%20" + "P650"
	case "AutoloadMAX700":
		return mobile_num + "%20" + "P700"
	case "AutoloadMAX900":
		return mobile_num + "%20" + "P900"
	case "GoSurf10":
		return mobile_num + "%20" + "GSURF10"
	case "GoSurf15":
		return mobile_num + "%20" + "GSURF15"
	case "GoSurf50":
		return mobile_num + "%20" + "GSURF50"
	case "GoSurf299":
		return mobile_num + "%20" + "GSURF299"
	default:
		return ""
	}
}
*/
