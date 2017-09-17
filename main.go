package main

import (
	"encoding/json"
	"fmt"
	"golang.org/x/net/websocket"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"html/template"
	"io/ioutil"
	"net/http"
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
	Email, Password, SessionID string
}

func main() {
	http.HandleFunc("/", home)
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
	sid := r.FormValue("TheyTubeSessionID")
	if sid != "" {
		s, e := mgo.Dial("127.0.0.1")
		checkErr(e)
		defer s.Close()
		uc := s.DB("theytube").C("users")
		u := User{}
		e = uc.Find(bson.M{"sessionid": sid}).One(&u)
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
	b, e := ioutil.ReadAll(ws)
	checkErr(e)
	u := User{}
	e = json.Unmarshal(b, &u)
	checkErr(e)
	fmt.Println(u)
	returnStr(ws, "OK")
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
		panic(e)
	}
}
func returnStr(ws *websocket.Conn, str string) error {
	_, e := ws.Write([]byte(str))
	return e
}
