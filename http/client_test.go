package http

import (
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

type UsersXml struct {
	XMLName xml.Name  `xml:"root" json:"-"`
	Users   []UserXml `xml:"row" json:"Users"`
}

type UserXml struct {
	XMLName   xml.Name `xml:"row" json:"-"`
	Id        int      `xml:"id"`
	FirstName string   `xml:"first_name" json:"-"`
	LastName  string   `xml:"last_name" json:"-"`
	Age       int      `xml:"age"`
	About     string   `xml:"about"`
	Gender    string   `xml:"gender"`
	Name      string
}

var indexCase = 0
var caseName = ""

func handleError(err error) {
	if err != nil {
		panic(err)
	}
}

func getUsersXml() []UserXml {
	xmlFile, err := os.Open("dataset.xml")
	defer xmlFile.Close()
	handleError(err)
	var users UsersXml
	byteValueXml, _ := ioutil.ReadAll(xmlFile)
	err = xml.Unmarshal(byteValueXml, &users)
	handleError(err)
	for i := range users.Users {
		users.Users[i].Name = users.Users[i].FirstName + " " + users.Users[i].LastName
	}
	return users.Users
}

func toJson(users []UserXml) []byte {
	jsonData, err := json.Marshal(users)
	handleError(err)
	return jsonData
}

func getSearchClient(url string) *SearchClient {
	searchClient := SearchClient{
		AccessToken: "123",
		URL:         url,
	}
	return &searchClient
}
func getSearchRequest(limit, offset, orderBy int, query, orderField string) *SearchRequest {
	request := SearchRequest{
		Limit:      limit,
		Offset:     offset,
		Query:      query,      //Name, About, пустой все записи с сортировкой
		OrderField: orderField, //Id, Age, Name, пустой - Name, другое - ошибка
		OrderBy:    orderBy,
	}
	return &request
}

func getParams(r *http.Request) *SearchRequest {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	offset, _ := strconv.Atoi(r.URL.Query().Get("offset"))
	query := r.URL.Query().Get("query")
	orderField := r.URL.Query().Get("orderField")
	orderBy, _ := strconv.Atoi(r.URL.Query().Get("orderBy"))
	request := SearchRequest{
		Limit:      limit,
		Offset:     offset,
		Query:      query,
		OrderField: orderField,
		OrderBy:    orderBy,
	}
	return &request
}

type TestCase struct {
	Request   SearchRequest
	Result    *SearchResponse
	Status    int
	errorName string
}

func filterUsers(users []UserXml, params *SearchRequest) []UserXml {
	if params.Limit <= 0 {
		return []UserXml{}
	}
	filteredUsers := make([]UserXml, 0, params.Limit)
	limit := 0
	for i := range users {
		if strings.Contains(users[i].Name, params.Query) || strings.Contains(users[i].About, params.Query) {
			filteredUsers = append(filteredUsers, users[i])
			limit++
			if limit >= params.Limit {
				break
			}
		}

	}
	return filteredUsers
}

func checkError(w http.ResponseWriter, r *http.Request) bool {
	switch caseName {
	case "TestWrongValues":
		if casesWrong[indexCase].errorName == "bad json" {
			w.Write([]byte("asd"))
			return true
		}
	case "TestStatuses":
		w.WriteHeader(casesStatuses[indexCase].Status)
		switch casesStatuses[indexCase].errorName {
		case "bad json":
			w.Write([]byte("asd"))
		case "ErrorBadOrderField":
			w.Write([]byte(`{"Error": "ErrorBadOrderField"}`))
		case "unknown":
			w.Write([]byte(`{"Error": "asd"}`))
		}
		return true
	case "TestTimeout":
		switch casesTimeout[indexCase].errorName {
		case "timeout":
			time.Sleep(time.Second)
		}
		return true
	}
	return false
}

func SearchServer(w http.ResponseWriter, r *http.Request) {
	params := getParams(r)
	if checkError(w, r) {
		return
	}
	users := getUsersXml()
	filteredUsers := filterUsers(users, params)
	jsonData := toJson(filteredUsers)
	//fmt.Println(string(jsonData))
	w.Write(jsonData)
}

var casesProper = []TestCase{
	// standard
	{
		Request: *getSearchRequest(2, 0, 1, "Nulla", "Name"),
		Result: &SearchResponse{
			Users: []User{
				{
					Id:     0,
					Name:   "Boyd Wolf",
					Age:    22,
					About:  "Nulla cillum enim voluptate consequat laborum esse excepteur occaecat commodo nostrud excepteur ut cupidatat. Occaecat minim incididunt ut proident ad sint nostrud ad laborum sint pariatur. Ut nulla commodo dolore officia. Consequat anim eiusmod amet commodo eiusmod deserunt culpa. Ea sit dolore nostrud cillum proident nisi mollit est Lorem pariatur. Lorem aute officia deserunt dolor nisi aliqua consequat nulla nostrud ipsum irure id deserunt dolore. Minim reprehenderit nulla exercitation labore ipsum.\n",
					Gender: "male",
				},
				{
					Id:     2,
					Name:   "Brooks Aguilar",
					Age:    25,
					About:  "Velit ullamco est aliqua voluptate nisi do. Voluptate magna anim qui cillum aliqua sint veniam reprehenderit consectetur enim. Laborum dolore ut eiusmod ipsum ad anim est do tempor culpa ad do tempor. Nulla id aliqua dolore dolore adipisicing.\n",
					Gender: "male",
				},
			},
			NextPage: true,
		},
	},
	// limit > 25
	{
		Request: *getSearchRequest(27, 0, 1, "Hilda", "Name"),
		Result: &SearchResponse{
			Users: []User{
				{
					Id:     1,
					Name:   "Hilda Mayer",
					Age:    21,
					About:  "Sit commodo consectetur minim amet ex. Elit aute mollit fugiat labore sint ipsum dolor cupidatat qui reprehenderit. Eu nisi in exercitation culpa sint aliqua nulla nulla proident eu. Nisi reprehenderit anim cupidatat dolor incididunt laboris mollit magna commodo ex. Cupidatat sit id aliqua amet nisi et voluptate voluptate commodo ex eiusmod et nulla velit.\n",
					Gender: "female",
				},
			},
			NextPage: false,
		},
	},
}

func TestProperValues(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	searchClient := getSearchClient(ts.URL)
	for caseNum, item := range casesProper {
		indexCase = caseNum
		caseName = "TestProperValues"
		gotRes, err := searchClient.FindUsers(item.Request)
		if !reflect.DeepEqual(item.Result, gotRes) {
			t.Errorf("[%d] wrong result", caseNum)
			t.Errorf("item %#v,", item.Result)
			t.Errorf("got %#v", gotRes)
		}
		handleError(err)
	}
}

var casesWrong = []TestCase{
	//limit -
	{
		Request: *getSearchRequest(-1, 0, 1, "Nulla", "Name"),
		Result:  (*SearchResponse)(nil),
	},
	//offset -
	{
		Request: *getSearchRequest(1, -1, 1, "Hilda", "Name"),
		Result:  (*SearchResponse)(nil),
	},
	//badJson
	{
		Request:   *getSearchRequest(1, 1, 1, "Hilda", "Name"),
		Result:    (*SearchResponse)(nil),
		Status:    http.StatusAccepted,
		errorName: "bad json",
	},
}

func TestWrongValues(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	searchClient := getSearchClient(ts.URL)
	for caseNum, item := range casesWrong {
		indexCase = caseNum
		caseName = "TestWrongValues"
		gotRes, errorPrint := searchClient.FindUsers(item.Request)
		if !reflect.DeepEqual(item.Result, gotRes) {
			t.Errorf("[%d] wrong result", caseNum)
			t.Error("error:", errorPrint)
			t.Errorf("item %#v,", SearchResponse{})
			t.Errorf("got %#v", gotRes)
		}
	}
}

var casesStatuses = []TestCase{
	//StatusUnauthorized
	{
		Request: *getSearchRequest(1, 0, 1, "Naa", "Name"),
		Result:  (*SearchResponse)(nil),
		Status:  http.StatusUnauthorized,
	},
	//StatusInternalServerError
	{
		Request: *getSearchRequest(1, 1, 1, "sad", "Name"),
		Result:  (*SearchResponse)(nil),
		Status:  http.StatusInternalServerError,
	},
	// StatusBadRequest: bad json
	{
		Request:   *getSearchRequest(1, 1, 1, "asd", "Name"),
		Result:    (*SearchResponse)(nil),
		Status:    http.StatusBadRequest,
		errorName: "bad json",
	},
	// StatusBadRequest: ErrorBadOrderField
	{
		Request:   *getSearchRequest(1, 1, 1, "asd", "Name"),
		Result:    (*SearchResponse)(nil),
		Status:    http.StatusBadRequest,
		errorName: "ErrorBadOrderField",
	},
	// StatusBadRequest: ErrorBadOrderField
	{
		Request:   *getSearchRequest(1, 1, 1, "asd", "Name"),
		Result:    (*SearchResponse)(nil),
		Status:    http.StatusBadRequest,
		errorName: "unknown",
	},
}

func TestStatuses(t *testing.T) {

	ts := httptest.NewServer(http.HandlerFunc(SearchServer))
	searchClient := getSearchClient(ts.URL)
	for caseNum, item := range casesStatuses {
		indexCase = caseNum
		caseName = "TestStatuses"
		gotRes, errorPrint := searchClient.FindUsers(item.Request)
		if !reflect.DeepEqual(item.Result, gotRes) {
			t.Errorf("[%d] wrong result", caseNum)
			t.Error("error:", errorPrint)
			t.Errorf("item %#v,", SearchResponse{})
			t.Errorf("got %#v", gotRes)
		}
	}
}

var casesTimeout = []TestCase{
	{
		Request:   *getSearchRequest(1, 0, 1, "Naa", "Name"),
		Result:    (*SearchResponse)(nil),
		errorName: "timeout",
	},
	{
		Request: *getSearchRequest(1, 0, 1, "Naa", "Name"),
		Result:  (*SearchResponse)(nil),
		errorName: "unknown",
	},

}

func TestTimeout(t *testing.T) {

	for caseNum, item := range casesTimeout {
		indexCase = caseNum
		caseName = "TestTimeout"
		ts := httptest.NewServer(http.HandlerFunc(SearchServer))
		url := ts.URL
		if item.errorName == "unknown" {
			url = "sd"
		}
		searchClient := getSearchClient(url)
		gotRes, errorPrint := searchClient.FindUsers(item.Request)
		if !reflect.DeepEqual(item.Result, gotRes) {
			t.Errorf("[%d] wrong result", caseNum)
			t.Error("error:", errorPrint)
			t.Errorf("item %#v,", SearchResponse{})
			t.Errorf("got %#v", gotRes)
		}
	}
}
