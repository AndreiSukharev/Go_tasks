package main

//import (
//	"context"
//	"encoding/json"
//	"fmt"
//	"net/http"
//	"net/url"
//	"strconv"
//	"sync"
//)
//
//// вы можете использовать ApiError в коде, который получается в результате генерации
//// считаем что это какая-то общеизвестная структура
//type ApiError struct {
//	HTTPStatus int
//	Err        error
//}
//
//func (ae ApiError) Error() string {
//	return ae.Err.Error()
//}
//
//// ----------------
//
//const (
//	statusUser      = 0
//	statusModerator = 10
//	statusAdmin     = 20
//)
//
//type MyApi struct {
//	statuses map[string]int
//	users    map[string]*User
//	nextID   uint64
//	mu       *sync.RWMutex
//}
//
//type ResponseStruct map[string]interface{}
//
//func checkEnum(status string, enum []string) bool {
//	check := false
//	for _, item := range enum {
//		if item == status {
//			check = true
//			break
//		}
//	}
//	return check
//}
//
//func sendError(w http.ResponseWriter, err string, status int) {
//	response := fmt.Sprintf(`{"error": "%s"}`, err)
//	w.WriteHeader(status)
//	w.Write([]byte(response))
//}
//
//func isErrorApi(err error, w http.ResponseWriter) bool {
//	switch err.(type) {
//	case ApiError:
//		apiError := err.(ApiError)
//		sendError(w, err.Error(), apiError.HTTPStatus)
//		return true
//	case error:
//		sendError(w, err.Error(), http.StatusInternalServerError)
//		return true
//	}
//	return false
//}
//
//func checkAuth(auth bool, w http.ResponseWriter, xAuth string) bool {
//	if !auth {
//		return true
//	}
//	if xAuth == "100500" {
//		return true
//	}
//	sendError(w, "unauthorized", http.StatusForbidden)
//	return false
//}
//
//func checkMethod(allowedMethod string, method string, w http.ResponseWriter) bool {
//	if allowedMethod == "" {
//		return true
//	}
//	if allowedMethod == method {
//		return true
//	}
//	sendError(w, "bad method", http.StatusNotAcceptable)
//	return false
//}
//func getParams(r *http.Request) url.Values {
//	var params url.Values
//
//	switch r.Method {
//	case http.MethodGet:
//		params = r.URL.Query()
//	case http.MethodPost:
//		err := r.ParseForm()
//		if err != nil {
//			panic(err)
//		}
//		params = r.Form
//	}
//	return params
//}
//
//func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
//
//	switch r.URL.Path {
//	case "/user/profile":
//		srv.wrapperProfile(w, r)
//	case "/user/create":
//		srv.wrapperCreate(w, r)
//	default:
//		sendError(w, "unknown method", http.StatusNotFound)
//		return
//	}
//}
//
//func (srv *MyApi) wrapperCreate(w http.ResponseWriter, r *http.Request) {
//	if !checkMethod(http.MethodPost, r.Method, w) {
//		return
//	}
//	if !checkAuth(true, w, r.Header.Get("X-Auth")) {
//		return
//	}
//	paramsMap := getParams(r)
//	param := CreateParams{}
//	if !param.unpackCreate(paramsMap, w) {
//		return
//	}
//	resApi, err := srv.Create(r.Context(), param)
//	if isErrorApi(err, w) {
//		return
//	}
//	responseStruct := ResponseStruct{
//		"error":    "",
//		"response": resApi,
//	}
//	response, _ := json.Marshal(responseStruct)
//	w.Write(response)
//}
//
//func (srv *MyApi) wrapperProfile(w http.ResponseWriter, r *http.Request) {
//	if !checkMethod("", r.Method, w) {
//		return
//	}
//	if !checkAuth(false, w, r.Header.Get("X-Auth")) {
//		return
//	}
//	paramsMap := getParams(r)
//	params := ProfileParams{}
//	if !params.unpackProfile(paramsMap, w) {
//		return
//	}
//	resApi, err := srv.Profile(r.Context(), params)
//	if isErrorApi(err, w) {
//		return
//	}
//	responseStruct := ResponseStruct{
//		"error":    "",
//		"response": resApi,
//	}
//	response, _ := json.Marshal(responseStruct)
//	w.Write(response)
//}
//
//
//func (in *ProfileParams) unpackProfile(params url.Values, w http.ResponseWriter) bool {
//	login, ok := params["login"]
//	if !ok {
//		sendError(w, "login must me not empty", http.StatusBadRequest)
//		return false
//	}
//	in.Login = login[0]
//	return true
//}
//
//func (in *CreateParams) unpackCreate(params url.Values, w http.ResponseWriter) bool {
//
//	login, ok := params["login"]
//	if !ok {
//		sendError(w, "login must me not empty", http.StatusBadRequest)
//		return false
//	}
//	if len(login[0]) < 10 {
//		sendError(w, "login len must be >= 10", http.StatusBadRequest)
//		return false
//	}
//	in.Login = login[0]
//
//	name := params["full_name"][0]
//	in.Name = name
//	status, ok := params["status"]
//	if ok {
//		if !checkEnum(status[0], []string{"user", "moderator", "admin"}) {
//			sendError(w, "status must be one of [user, moderator, admin]", http.StatusBadRequest)
//			return false
//		}
//		in.Status = status[0]
//	} else {
//		in.Status = "user"
//	}
//	//in.Status = status[0]
//	ageStr := params["age"][0]
//	ageInt, err := strconv.Atoi(ageStr)
//	if err != nil {
//		sendError(w, "age must be int", http.StatusBadRequest)
//		return false
//	}
//	if ageInt < 0 {
//		sendError(w, "age must be >= 0", http.StatusBadRequest)
//		return false
//	}
//	if ageInt > 128 {
//		sendError(w, "age must be <= 128", http.StatusBadRequest)
//		return false
//	}
//	in.Age = ageInt
//	return true
//}
//
//func NewMyApi() *MyApi {
//	return &MyApi{
//		statuses: map[string]int{
//			"user":      0,
//			"moderator": 10,
//			"admin":     20,
//		},
//		users: map[string]*User{
//			"rvasily": &User{
//				ID:       42,
//				Login:    "rvasily",
//				FullName: "Vasily Romanov",
//				Status:   statusAdmin,
//			},
//		},
//		nextID: 43,
//		mu:     &sync.RWMutex{},
//	}
//}
//
//type ProfileParams struct {
//	Login string `apivalidator:"required"`
//}
//
//type CreateParams struct {
//	Login  string `apivalidator:"required,min=10"`
//	Name   string `apivalidator:"paramname=full_name"`
//	Status string `apivalidator:"enum=user|moderator|admin,default=user"`
//	Age    int    `apivalidator:"min=0,max=128"`
//}
//
//type User struct {
//	ID       uint64 `json:"id"`
//	Login    string `json:"login"`
//	FullName string `json:"full_name"`
//	Status   int    `json:"status"`
//}
//
//type NewUser struct {
//	ID uint64 `json:"id"`
//}
//
//// apigen:api {"url": "/user/profile", "auth": false}
//func (srv *MyApi) Profile(ctx context.Context, in ProfileParams) (*User, error) {
//
//	if in.Login == "bad_user" {
//		return nil, fmt.Errorf("bad user")
//	}
//	srv.mu.RLock()
//	user, exist := srv.users[in.Login]
//	srv.mu.RUnlock()
//	if !exist {
//		return nil, ApiError{http.StatusNotFound, fmt.Errorf("user not exist")}
//	}
//
//	return user, nil
//}
//
//// apigen:api {"url": "/user/create", "auth": true, "method": "POST"}
//func (srv *MyApi) Create(ctx context.Context, in CreateParams) (*NewUser, error) {
//	if in.Login == "bad_username" {
//		return nil, fmt.Errorf("bad user")
//	}
//
//	srv.mu.Lock()
//	defer srv.mu.Unlock()
//
//	_, exist := srv.users[in.Login]
//	if exist {
//		return nil, ApiError{http.StatusConflict, fmt.Errorf("user %s exist", in.Login)}
//	}
//
//	id := srv.nextID
//	srv.nextID++
//	srv.users[in.Login] = &User{
//		ID:       id,
//		Login:    in.Login,
//		FullName: in.Name,
//		Status:   srv.statuses[in.Status],
//	}
//	return &NewUser{id}, nil
//}
//
//// 2-я часть
//// это похожая структура, с теми же методами, но у них другие параметры!
//// код, созданный вашим кодогенератором работает с конкретной струткурой, про другие ничего не знает
//// поэтому то что рядом есть ещё походая структура с такими же методами его нисколько не смущает
//
//type OtherApi struct {
//}
//
//func NewOtherApi() *OtherApi {
//	return &OtherApi{}
//}
//
//type OtherCreateParams struct {
//	Username string `apivalidator:"required,min=3"`
//	Name     string `apivalidator:"paramname=account_name"`
//	Class    string `apivalidator:"enum=warrior|sorcerer|rouge,default=warrior"`
//	Level    int    `apivalidator:"min=1,max=50"`
//}
//
//type OtherUser struct {
//	ID       uint64 `json:"id"`
//	Login    string `json:"login"`
//	FullName string `json:"full_name"`
//	Level    int    `json:"level"`
//}
//
//// apigen:api {"url": "/user/create", "auth": true, "method": "POST"}
//func (srv *OtherApi) Create(ctx context.Context, in OtherCreateParams) (*OtherUser, error) {
//	return &OtherUser{
//		ID:       12,
//		Login:    in.Username,
//		FullName: in.Name,
//		Level:    in.Level,
//	}, nil
//}
