package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"splitwiseAngularAPI/expense"

	"github.com/dghubble/oauth1"
	"github.com/gorilla/securecookie"
	"github.com/pkg/errors"
)

type sessionValues struct {
	sessionID string
	token     *oauth1.Token
}

/*Configuration - structure for configuration*/
type Configuration struct {
	AccessTokenURL  string `json:"AccessTokenURL"`
	AuthorizeURL    string `json:"AuthorizeURL"`
	RequestTokenURL string `json:"RequestTokenURL"`
	ConsumerKey     string `json:"ConsumerKey"`
	ConsumerSecret  string `json:"ConsumerSecret"`
	CallbackURL     string `json: "CallbackURL"`
	AngularHandler  string `json: "AngularHandler"`
}

//Trace - logger
var Trace *log.Logger

var splitwiseEndPoint = new(oauth1.Endpoint)

var splitwiseAuthConfig = new(oauth1.Config)

var requestTok = ""
var requestSec = ""

var cookieHandler = securecookie.New(securecookie.GenerateRandomKey(64), securecookie.GenerateRandomKey(32))
var sessionMapper = make(map[string]*sessionValues)

var config = new(Configuration)

//ConfigFilePath - config file path
var ConfigFilePath string

//InitializeConfig initialize config file
func InitializeConfig(filePath string) {
	//read json file
	file, err := ioutil.ReadFile(filePath)
	if err != nil {
		fmt.Println("error reading config file - Exiting")
		os.Exit(1)
	}

	//marshall configuration object
	err = json.Unmarshal(file, config)
	if err != nil {
		fmt.Println("error reading config file - Exiting", err)
		os.Exit(1)
	}

	splitwiseEndPoint = &oauth1.Endpoint{
		AccessTokenURL:  config.AccessTokenURL,
		AuthorizeURL:    config.AuthorizeURL,
		RequestTokenURL: config.RequestTokenURL,
	}

	splitwiseAuthConfig = &oauth1.Config{
		ConsumerKey:    config.ConsumerKey,
		ConsumerSecret: config.ConsumerSecret,
		CallbackURL:    config.CallbackURL,
		Endpoint:       *splitwiseEndPoint,
	}
}

/*InitLogger - log initializer*/
func InitLogger(file *os.File) {
	if file != nil {
		Trace = log.New(file,
			"TRACE: ",
			log.Ldate|log.Ltime|log.Lshortfile)
	}
}

/*IndexHandler - Handler for / */
func IndexHandler(w http.ResponseWriter, r *http.Request) {

	Trace.Println("Got request for:", r.URL.String())

	//if this is just a refresh
	if refreshSession(w, r) {
		return
	}

	//1. Your application requests authorization
	requestToken, requestSecret, err := splitwiseAuthConfig.RequestToken()
	requestTok = requestToken
	requestSec = requestSecret
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	authorizationURL, err := splitwiseAuthConfig.AuthorizationURL(requestToken)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	http.Redirect(w, r, authorizationURL.String(), http.StatusFound)
}

/*CompleteAuth - Handler for authorization callback*/
func CompleteAuth(w http.ResponseWriter, r *http.Request) {

	// use the token to get an authenticated client
	requestTok, verifier, err := oauth1.ParseAuthorizationCallback(r)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}
	accessToken, accessSecret, err := splitwiseAuthConfig.AccessToken(requestTok, requestSec, verifier)
	if err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	sessionToken := oauth1.NewToken(accessToken, accessSecret)
	//cache = map[user]{sessionid,sessiontoken}
	//cookie = {user,sessionid}
	setCookieAndCache(w, sessionToken)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", config.AngularHandler)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	http.Redirect(w, r, config.AngularHandler, http.StatusFound)
}

/*Sets cookie and internal map
//cache = map[user]{sessionid,sessiontoken}
//cookie = {user,sessionid}
*/
func setCookieAndCache(w http.ResponseWriter, sessionToken *oauth1.Token) {

	user := getCurrentUserID(sessionToken)
	if user == "" {
		return
	}

	sessionID := createSessionID()

	cookieVal := map[string]string{
		"username":  user,
		"sessionid": sessionID,
	}
	cookieEncoded, err := cookieHandler.Encode("clientMap", cookieVal)
	if err != nil {
		Trace.Println("error encoding cookie")
		errors.Wrap(err, "error encoding cookie")

	}
	cookie := &http.Cookie{
		Name:   "clientMap",
		Value:  cookieEncoded,
		Path:   "/",
		MaxAge: 300,
	}
	http.SetCookie(w, cookie)

	//save session in a map
	sessionMapper[user] = &sessionValues{sessionID: sessionID, token: sessionToken}
}

func clearCookieAndCache(w http.ResponseWriter, request *http.Request) {
	cookie := &http.Cookie{
		Name:   "clientMap",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	}
	http.SetCookie(w, cookie)

	var cookieUserName string

	//get cookie from client request
	cookie, err := request.Cookie("clientMap")
	if err != nil {
		Trace.Println(err)
	}

	cookieValue := make(map[string]string)
	err = cookieHandler.Decode("clientMap", cookie.Value, &cookieValue)
	if err != nil {
		Trace.Println(err)
	}

	cookieUserName = cookieValue["username"]

	//save session in a map
	delete(sessionMapper, cookieUserName)
}
func createSessionID() string {
	//return a random string
	timeNow := time.Now()
	randSource := timeNow.Hour()*3600 + timeNow.Minute()*60 + timeNow.Second()
	r := rand.New(rand.NewSource(int64(randSource)))
	return strconv.Itoa(r.Intn(100000))
}

/*called from each http get request from angular front end*/
func validateSessionAndGetUser(request *http.Request) *sessionValues {
	var cookieUserName string
	var cookieSession string
	var storedSession string

	//get cookie from client request
	cookie, err := request.Cookie("clientMap")
	if err != nil {
		Trace.Println(err)
		return nil

	}
	cookieValue := make(map[string]string)
	err = cookieHandler.Decode("clientMap", cookie.Value, &cookieValue)
	if err != nil {
		Trace.Println(err)
		return nil
	}

	cookieUserName = cookieValue["username"]
	cookieSession = cookieValue["sessionid"]
	storedSession = sessionMapper[cookieUserName].sessionID
	//compare session ids
	if cookieSession != storedSession {
		return nil
	}

	//get stored sessionid from server
	return sessionMapper[cookieUserName]

}

/*Logout - clear cookie nad cache*/
func Logout(w http.ResponseWriter, r *http.Request) {
	sessionVals := validateSessionAndGetUser(r)
	if sessionVals == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	clearCookieAndCache(w, r)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", config.AngularHandler)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
}

func refreshSession(w http.ResponseWriter, r *http.Request) bool {
	sessionVals := validateSessionAndGetUser(r)
	if sessionVals == nil {
		return false
	}

	setCookieAndCache(w, sessionVals.token)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", config.AngularHandler)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	http.Redirect(w, r, config.AngularHandler, http.StatusFound)
	return true
}

/*getCurrentUserID - Given a session token return current user ID*/
func getCurrentUserID(sessionToken *oauth1.Token) string {
	// httpClient will automatically authorize http.Request's
	httpClient := splitwiseAuthConfig.Client(oauth1.NoContext, sessionToken)
	response, err := httpClient.Get("https://secure.splitwise.com/api/v3.0/get_current_user")
	if err != nil {
		return ""
	}

	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)

	var userData interface{}
	err = json.Unmarshal(contents, &userData)
	if err != nil {
		return ""
	}

	userDataObj := userData.(map[string]interface{})
	userDataMap := userDataObj["user"].(map[string]interface{})

	return strconv.FormatFloat(userDataMap["id"].(float64), 'f', 0, 64)

}

/*GetExpenseURLForGroup - Expense for a group*/
func GetExpenseURLForGroup(groupID string, startDate time.Time, endDate time.Time) string {
	requestURL, _ := url.Parse("https://secure.splitwise.com/api/v3.0/get_expenses")
	requestQuery := requestURL.Query()
	requestQuery.Set("group_id", groupID)
	requestQuery.Set("dated_after", startDate.String())
	requestQuery.Set("dated_before", endDate.String())
	requestQuery.Set("limit", "0")
	requestURL.RawQuery = requestQuery.Encode()
	return requestURL.String()
}

/*getUserInfo- User info within a group*/
func getUserInfo(userArr []interface{}) string {
	var userLine = ""
	for _, user := range userArr {
		userMap := user.(map[string]interface{})
		userInfoMap := userMap["user"].(map[string]interface{})
		userName := userInfoMap["first_name"].(string)
		userName = strings.Replace(userName, ",", "", -1)
		userShare := ""
		if userMap["owed_share"] != nil {
			userShare = userMap["owed_share"].(string)
			userShare = strings.Replace(userShare, ",", "", -1)

		}

		tempStrArr := []string{userName, userShare}
		tempStr := strings.Join(tempStrArr, "_")
		if userLine == "" {
			userLine = tempStr
		} else {
			userLineArr := []string{userLine, tempStr}
			userLine = strings.Join(userLineArr, ",")
		}

	}
	return userLine
}

/*GetGroups - get groups for current user*/
func GetGroups(w http.ResponseWriter, r *http.Request) {
	//get session values
	sessionVals := validateSessionAndGetUser(r)
	if sessionVals == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	httpClient := splitwiseAuthConfig.Client(oauth1.NoContext, sessionVals.token)
	response, err := httpClient.Get("https://secure.splitwise.com/api/v3.0/get_groups")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer response.Body.Close()
	contents, err := ioutil.ReadAll(response.Body)

	var groupData interface{}
	err = json.Unmarshal(contents, &groupData)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	//create object to send

	groupDataArr := groupData.(map[string]interface{})["groups"].([]interface{})
	groupIDNameMap := make(map[string]string)

	for _, group := range groupDataArr {
		groupMap := group.(map[string]interface{})
		groupIDNameMap[strconv.FormatFloat(groupMap["id"].(float64), 'f', 0, 64)] = groupMap["name"].(string)

	}

	//send response
	groupIDNameJSON, err := json.Marshal(groupIDNameMap)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", config.AngularHandler)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Write(groupIDNameJSON)
}

/*GetGroupUsers - get users for a group*/
func GetGroupUsers(w http.ResponseWriter, r *http.Request) {
	//get session values
	sessionVals := validateSessionAndGetUser(r)
	if sessionVals == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//read request body
	defer r.Body.Close()
	reqURL := r.RequestURI
	u, err := url.Parse(reqURL)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	q := u.Query()
	groupID := q["groupID"][0]

	httpClient := splitwiseAuthConfig.Client(oauth1.NoContext, sessionVals.token)

	requestURL, _ := url.Parse("https://secure.splitwise.com/api/v3.0/get_group")
	requestQuery := requestURL.Query()
	requestQuery.Set("id", groupID)
	requestURL.RawQuery = requestQuery.Encode()

	response, err := httpClient.Get(requestURL.String())
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	defer response.Body.Close()

	//create object to send
	contents, _ := ioutil.ReadAll(response.Body)

	//unmarshall to expense object
	var groupWrapper expense.GroupWrapper
	json.Unmarshal(contents, &groupWrapper)

	//extract individual expenses
	memberArr := extractMembers(groupWrapper)

	//send response
	contentJSON, err := json.Marshal(memberArr)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", config.AngularHandler)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Write(contentJSON)
}

/*GetGroupData - Get group data*/
func GetGroupData(w http.ResponseWriter, r *http.Request) {
	//get session values
	sessionVals := validateSessionAndGetUser(r)
	if sessionVals == nil {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	//read request body
	defer r.Body.Close()
	reqURL := r.RequestURI
	u, err := url.Parse(reqURL)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
	}
	q := u.Query()
	groupID := q["groupID"][0]
	startDate, endDate := getStartAndEndDate(q)

	httpClient := splitwiseAuthConfig.Client(oauth1.NoContext, sessionVals.token)
	requestURL := GetExpenseURLForGroup(groupID, startDate, endDate)
	expenseResponse, _ := httpClient.Get(requestURL)
	contents, _ := ioutil.ReadAll(expenseResponse.Body)

	//unmarshall to expense object
	var expensesWrapper expense.ExpensesWrapper
	json.Unmarshal(contents, &expensesWrapper)

	//extract individual expenses
	userInfoArr := extractExpenses(expensesWrapper)

	//send response
	contentJSON, err := json.Marshal(userInfoArr)

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", config.AngularHandler)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Write(contentJSON)

}
func extractExpenses(expensesWrapper expense.ExpensesWrapper) []expense.ResponseExpense {
	expenseArr := expensesWrapper.Expenses
	responseExpenseArr := make([]expense.ResponseExpense, 0)
	for _, individualExpense := range expenseArr {
		for _, userInfo := range individualExpense.Users {
			responseExpenseArr = append(responseExpenseArr, expense.ResponseExpense{Category: individualExpense.Category.Name, UserID: userInfo.UserID, OwedShare: userInfo.OwedShare, Date: individualExpense.Date})
		}
	}
	return responseExpenseArr
}

func extractMembers(groupWrapper expense.GroupWrapper) []expense.Members {
	return groupWrapper.Group.Members
}

func getStartAndEndDate(query url.Values) (time.Time, time.Time) {
	startYear, _ := strconv.Atoi(query["startYear"][0])
	startMonth, _ := strconv.Atoi(query["startMonth"][0])
	startDate, _ := strconv.Atoi(query["startDay"][0])
	endYear, _ := strconv.Atoi(query["endYear"][0])
	endMonth, _ := strconv.Atoi(query["endMonth"][0])
	endDate, _ := strconv.Atoi(query["endDay"][0])

	loc := time.FixedZone("UTC-8", 0)

	startTime := time.Date(startYear, time.Month(startMonth), startDate, 0, 0, 0, 0, loc)
	endTime := time.Date(endYear, time.Month(endMonth), endDate, 0, 0, 0, 0, loc)

	return startTime, endTime
}
