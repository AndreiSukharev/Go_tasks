package db

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
)

type Response map[string]interface{}

type DbHandler struct {
	DB     *sql.DB
	Tables []string
}

type ParamsTable struct {
	TableName string
	Limit     int
	Offset    int
}

type Table struct {
	Name    string
	Columns []*TableColumn
}

func (t *Table) newRecord(row []interface{}) *Records {
	record := Records{}
	for i := range row {
		record[t.Columns[i].Field] = row[i]
	}
	return &record
}

func (t *Table) validate(data Records) string{
	for _, column := range t.Columns {
		fieldType := column.Field
		val, ok := data[fieldType]
		if ok {
			if column.Key || !column.Type.isValidValue(val){
				return fieldType
			}
		}
	}
	return ""
}

func (t *Table) getFieldKey() string {
	for _, column := range t.Columns {
		if column.Key {
			return column.Field
		}
	}
	return "id"
}
type Records map[string]interface{}

type TableColumn struct {
	Field string
	Type  TypeColumn
	Null  bool
	Key   bool
}

type TypeColumn interface {
	newVar() interface{}
	isValidValue(val interface{}) bool
}

type IntColumn struct {
	Null bool
}

func (v IntColumn) newVar() interface{} {
	if v.Null {
		return new(*int64)
	}
	return new(int64)
}

func (v IntColumn) isValidValue(val interface{}) bool {
	if val == nil {
		return v.Null
	}
	_, ok := val.(int64)
	return ok
}

type StringColumn struct {
	Null bool
}

func (v StringColumn) newVar() interface{} {
	if v.Null {
		return new(*string)
	}
	return new(string)
}

func (v StringColumn) isValidValue(val interface{}) bool {
	if val == nil {
		return v.Null
	}
	_, ok := val.(string)
	return ok
}

func (h *DbHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	parts := strings.Split(path, "/")
	size := len(parts)
	//fmt.Println(size)
	tableName := parts[1]
	if size == 2 {
		if parts[1] == "" {
			h.getAllTables(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.getItems(w, r, tableName)

		}
	} else if size == 3 {
		id := parts[2]
		switch r.Method {
		case http.MethodPut:
			h.addItem(w, r, tableName)
		case http.MethodGet:
			h.getItem(w, r, tableName, id)
		case http.MethodPost:
			h.editItem(w, r, tableName, id)
		case http.MethodDelete:
			h.deleteItem(w, r, tableName, id)
		}
	}
}

func NewDbExplorer(db *sql.DB) (http.Handler, error) {
	dbHandler := DbHandler{DB: db}
	//r := mux.NewRouter()
	//r.HandleFunc("/", dbHandler.getAllTables).Methods(http.MethodGet)
	//r.HandleFunc("/{table}", dbHandler.getItems).Methods(http.MethodGet)
	//r.HandleFunc("/{table}/{id}", dbHandler.getItem).Methods(http.MethodGet)
	//r.HandleFunc("/{table}/{id}", dbHandler.editItem).Methods(http.MethodPost)
	//r.HandleFunc("/{table}/{id}", dbHandler.deleteItem).Methods(http.MethodDelete)
	//r.HandleFunc("/{table}/", dbHandler.addItem).Methods(http.MethodPut)

	return &dbHandler, nil
}

func (h *DbHandler) deleteItem(w http.ResponseWriter, r *http.Request, tableName, id string) {
	if !h.checkTables(tableName) {
		sendError(w, "unknown table", http.StatusNotFound)
		return
	}
	table := h.getFullTable(tableName)
	count := h.handleDeleteItem(&table, id)
	res := createResponsePost("deleted", count)
	w.Write(res)
}

func (h *DbHandler) handleDeleteItem(table *Table, id string) int64 {
	fieldKey := table.getFieldKey()
	sqlText := fmt.Sprintf("DELETE FROM %s WHERE %s = %s;", table.Name, fieldKey, id)
	res, err := h.DB.Exec(sqlText)
	__err_panic("delItem exec", err)

	count, err := res.RowsAffected()
	__err_panic("delItem RowsAffected", err)
	return count
}


func (h *DbHandler) editItem(w http.ResponseWriter, r *http.Request, tableName, id string) {
	if !h.checkTables(tableName) {
		sendError(w, "unknown table", http.StatusNotFound)
		return
	}
	table := h.getFullTable(tableName)
	data := getParsedJson(r.Body)
	typeErr := table.validate(data)
	if typeErr != "" {
		sendError(w, "field "+typeErr+" have invalid type", http.StatusBadRequest)
		return
	}
	count := h.handleEditItem(&table, data, id)
	res := createResponsePost("updated", count)
	w.Write(res)
}

func (h *DbHandler) handleEditItem(table *Table, data Records, id string) int64 {
	values := make([]interface{}, 0)
	sizeData := len(data)
	sqlText := "UPDATE " + table.Name + " SET "
	i := 0
	for _, column := range table.Columns {
		if !column.Key {
			val, ok := data[column.Field]
			if ok {
				i++
				sqlText += fmt.Sprintf("%s = ?", column.Field)
				if i < sizeData {
					sqlText += ", "
				}
				values = append(values, val)
			}
		}
	}
	fieldKey := table.getFieldKey()
	sqlText += fmt.Sprintf(" WHERE %s = %s;", id, fieldKey)
	res, err := h.DB.Exec(sqlText, values...)
	__err_panic("editItem exec", err)
	count, err := res.RowsAffected()
	__err_panic("editItem RowsAffected", err)
	return count
}

func (h *DbHandler) addItem(w http.ResponseWriter, r *http.Request, tableName string) {
	if !h.checkTables(tableName) {
		sendError(w, "unknown table", http.StatusNotFound)
		return
	}
	table := h.getFullTable(tableName)
	data := getParsedJson(r.Body)
	fieldKey := table.getFieldKey()
	lastID := h.handleInsertItem(&table, data)
	res := createResponsePost(fieldKey, lastID)
	w.Write(res)
}

func getParsedJson(body io.ReadCloser) Records {
	data := Records{}
	buff, _ := ioutil.ReadAll(body)
	err := json.Unmarshal(buff, &data)
	__err_panic("Unmarshal", err)
	return data
}

func (h *DbHandler) handleInsertItem(table *Table, data Records) int64 {
	fields := make([]string, 0)
	values := make([]interface{}, 0)
	for _, column := range table.Columns {
		if !column.Key {
			val, ok := data[column.Field]
			fields = append(fields, column.Field)
			if ok {
				values = append(values, val)
			} else {
				values = append(values, column.Type.newVar())
			}
		}
	}
	fieldsText := strings.Join(fields, ", ")
	sizeValues := len(values)
	valuesText := ""
	for i := 0; i < sizeValues; i++ {
		valuesText += "?"
		if i+1 != sizeValues {
			valuesText += ", "
		}
	}
	sqlText := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);", table.Name, fieldsText, valuesText)
	if table.Name == "users" {
		//fmt.Println(sqlText)
		//fmt.Println(values...)
	}
	result, err := h.DB.Exec(sqlText, values...)
	__err_panic("addItem INSERT", err)
	lastID, err := result.LastInsertId()
	__err_panic("addItem LastInsertId", err)
	return lastID
}

func createResponsePost(key string, id int64) []byte {
	response := Response{
		"response": Response{
			key: id,
		},
	}
	res, _ := json.Marshal(response)
	return res
}

func (h *DbHandler) getItem(w http.ResponseWriter, r *http.Request, tableName, id string) {
	if !h.checkTables(tableName) {
		sendError(w, "unknown table", http.StatusNotFound)
		return
	}
	table := h.getFullTable(tableName)
	fieldKey := table.getFieldKey()
	item, errText := h.handleGetItem(&table, fieldKey, id)
	if errText != "" {
		sendError(w, "record not found", http.StatusNotFound)
		return
	}
	res := createResponseRecord(item)
	w.Write(res)
}

func (h *DbHandler) handleGetItem(table *Table, fieldKey, id string) (*Records, string) {
	item := createRow(table)
	sqlText := fmt.Sprintf(`SELECT * FROM %s WHERE %s = ?;`, table.Name, fieldKey)
	row := h.DB.QueryRow(sqlText, id)
	err := row.Scan(item...)
	newRecord := table.newRecord(item)
	if err != nil {
		return newRecord, "record not found"
	}
	return newRecord, ""
}

func createResponseRecord(item *Records) []byte {
	response := Response{
		"response": Response{
			"record": item,
		},
	}
	res, _ := json.Marshal(response)
	return res
}

//GET /$table?limit=5&offset=7 - возвращает список из 5 записей (limit)
//начиная с 7-й (offset) из таблицы $table. limit по-умолчанию 5, offset 0
func (h *DbHandler) getItems(w http.ResponseWriter, r *http.Request, tableName string) {
	if !h.checkTables(tableName) {
		sendError(w, "unknown table", http.StatusNotFound)
		return
	}
	table := h.getFullTable(tableName)
	paramsTable := getLimitOffset(r, tableName)
	items := h.handleGetAllItems(&table, paramsTable)
	res := createResponseRecords(items)
	w.Write(res)
}

func (h *DbHandler) handleGetAllItems(table *Table, paramsTable *ParamsTable) []*Records {
	sqlText := fmt.Sprintf(`SELECT * FROM %s LIMIT ? OFFSET ?;`, paramsTable.TableName)
	rows, err := h.DB.Query(sqlText, paramsTable.Limit, paramsTable.Offset)
	__err_panic("!getItems query:", err)
	defer rows.Close()
	items := []*Records{}
	for rows.Next() {
		item := createRow(table)
		err = rows.Scan(item...)
		__err_panic("getItems rows.Next:", err)
		items = append(items, table.newRecord(item))
	}
	return items
}

func createResponseRecords(items []*Records) []byte {
	response := Response{
		"response": Response{
			"records": items,
		},
	}
	res, _ := json.Marshal(response)
	return res
}

func getLimitOffset(r *http.Request, tableName string) *ParamsTable {
	paramsTable := ParamsTable{
		Limit:     5,
		Offset:    0,
		TableName: tableName,
	}
	limit, err := strconv.Atoi(r.URL.Query().Get("limit"))
	if err == nil {
		paramsTable.Limit = limit
	}
	offset, err := strconv.Atoi(r.URL.Query().Get("offset"))
	if err == nil {
		paramsTable.Offset = offset
	}
	return &paramsTable
}

//GET / - возвращает список все таблиц
// (которые мы можем использовать в дальнейших запросах)
func (h *DbHandler) getAllTables(w http.ResponseWriter, r *http.Request) {
	h.getTables()
	response := Response{
		"response": Response{
			"tables": h.Tables,
		},
	}
	res, _ := json.Marshal(response)
	w.Write(res)
}

func (h *DbHandler) getTables() {
	tables := make([]string, 0, 2)
	rows, err := h.DB.Query("SHOW tables;")
	defer rows.Close()
	__err_panic("!getAllTables error:", err)
	for rows.Next() {
		var table string
		rows.Scan(&table)
		tables = append(tables, table)
	}
	h.Tables = tables
}

func (h *DbHandler) checkTables(curTables string) bool {
	if len(h.Tables) == 0 {
		h.getTables()
	}
	for _, table := range h.Tables {
		if table == curTables {
			return true
		}
	}
	return false
}

func (h *DbHandler) getFullTable(tableName string) Table {
	columns := []*TableColumn{}
	var (
		collation  string
		typeCol    string
		key        interface{}
		null       string
		defaultCol string
		extra      string
		privileges string
		comment    string
	)
	sqlText := "SHOW FULL COLUMNS FROM " + tableName + ";"
	rows, err := h.DB.Query(sqlText)
	defer rows.Close()
	__err_panic("getFullTable err", err)
	for rows.Next() {
		column := &TableColumn{}
		rows.Scan(&column.Field, &typeCol, &collation, &null, &key, &defaultCol, &extra, &privileges, &comment)
		column.Null = null == "YES"
		column.Key = key == nil
		if strings.Contains(typeCol, "int") {
			column.Type = IntColumn{Null: column.Null}
		} else {
			column.Type = StringColumn{Null: column.Null}
		}
		columns = append(columns, column)
	}
	table := Table{Name: tableName, Columns: columns}
	return table
}

func createRow(table *Table) []interface{} {
	row := make([]interface{}, 0, len(table.Columns))
	for _, column := range table.Columns {
		row = append(row, column.Type.newVar())
	}
	return row
}

func sendError(w http.ResponseWriter, err string, status int) {
	response := fmt.Sprintf(`{"error": "%s"}`, err)
	w.WriteHeader(status)
	w.Write([]byte(response))
}

func __err_panic(mes string, err error) {
	if err != nil {
		fmt.Println(mes)
		panic(err)
	}
}
