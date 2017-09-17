package main

import (
	// "bufio"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"golang.org/x/net/websocket"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"html/template"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"
)

type Video struct {
	Clips               []Clip
	Title, Introduction string
	OwnerID             string
	Cover               string
	Count               int
}
type Clip struct {
	Title, Vid string
	Links      []Link
}
type Link struct {
	Title, Str string
	Count      int
}
type User struct {
	BaseInfo
	Email, Password, SessionID string
}
type BaseInfo struct {
	State, Info string
}

func main() {
	http.HandleFunc("/", home)
	http.Handle("/wsLogin", websocket.Handler(wsLogin))
	http.Handle("/wsRegister", websocket.Handler(wsRegister))
	http.ListenAndServe(":8090", nil)
}
func home(w http.ResponseWriter, r *http.Request) {
	hd := struct {
		LoginedIn bool
		Email     string
	}{
		false,
		"",
	}
	sid, e := r.Cookie("TheyTubeSessionID")
	if e == nil {
		s, e := mgo.Dial("127.0.0.1")
		checkErr(e)
		defer s.Close()
		uc := s.DB("theytube").C("users")
		u := User{}
		e = uc.Find(bson.M{"sessionid": sid.Value}).One(&u)
		if e == nil {
			hd.LoginedIn = true
			hd.Email = u.Email
		}
	}
	t, e := template.ParseFiles("index.html")
	checkErr(e)
	t.Execute(w, hd)
}

func wsLogin(ws *websocket.Conn) {
	defer ws.Close()
	b := make([]byte, 1024)
	n, e := ws.Read(b)
	if testErr(e) {
		return
	}
	u := User{}
	e = json.Unmarshal(b[:n], &u)
	if testErr(e) {
		return
	}
	gu, e := findUser(bson.M{"email": u.Email})
	if e != nil {
		returnInfo(ws, "ERR", "用户不存在，请先注册")
		return
	}
	if gu.Password != u.Password {
		returnInfo(ws, "ERR", "密码不正确")
		return
	}
	returnInfo(ws, "OK", gu.SessionID)
}
func wsRegister(ws *websocket.Conn) {
	defer ws.Close()
	b := make([]byte, 1024)
	l, e := ws.Read(b)
	if testErr(e) {
		return
	}
	u := User{}
	e = json.Unmarshal(b[:l], &u)
	if testErr(e) {
		return
	}
	if len(u.Password) < 6 || len(u.Password) > 25 {
		returnInfo(ws, "ERR", "密码长度要在6~25之间")
		return
	}
	_, e = findUser(bson.M{"email": u.Email})
	if e == nil {
		returnInfo(ws, "ERR", "用户已存在，请直接登录")
		return
	}
	u.SessionID = NewToken()
	e = insertUser(u)
	checkErr(e)
	returnInfo(ws, "OK", u.SessionID)
}
func wsNew(ws *websocket.Conn) {

}
func wsUpload(ws *websocket.Conn) {

}
func wsGetVideo(ws *websocket.Conn) {

}
func checkErr(e error) {
	if e != nil {
		fmt.Println(e)
		f, e := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE, 0666)
		f.Write([]byte(e.Error() + "\n"))
		f.Close()
		panic(e)
	}
}
func testErr(e error) bool {
	if e != nil {
		fmt.Println(e)
		f, e := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE, 0666)
		f.Write([]byte(e.Error() + "\n"))
		f.Close()
		return true
	}
	return false
}
func returnStr(ws *websocket.Conn, str string) error {
	_, e := ws.Write([]byte(str))
	return e
}
func returnInfo(ws *websocket.Conn, state, info string) {
	b, e := json.Marshal(BaseInfo{State: state, Info: info})
	checkErr(e)
	ws.Write(b)
}
func findUser(m bson.M) (User, error) {
	s, e := mgo.Dial("127.0.0.1")
	checkErr(e)
	defer s.Close()
	uc := s.DB("theytube").C("users")
	u := User{}
	e = uc.Find(m).One(&u)
	return u, e
}
func insertUser(u interface{}) error {
	s, e := mgo.Dial("127.0.0.1")
	checkErr(e)
	defer s.Close()
	uc := s.DB("theytube").C("users")
	e = uc.Insert(&u)
	return e
}
func NewToken() string {
	ct := time.Now().Unix()
	h5 := md5.New()
	io.WriteString(h5, strconv.FormatInt(ct, 10))
	token := fmt.Sprintf("%x", h5.Sum(nil))
	return token
}
