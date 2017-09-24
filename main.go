package main

import (
	// "bufio"
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"golang.org/x/net/websocket"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"time"
)

type Video struct {
	BaseInfo
	Clips               []Clip
	Vid                 string
	Title, Introduction string
	Tags                string
	Owner               string
	Cover               string
	Count               int
	UploadTime          time.Time
}
type Clip struct {
	Title string
	Links []string
}
type User struct {
	BaseInfo
	Email, Password, SessionID string
}
type BaseInfo struct {
	State, Info string
}

func main() {
	http.Handle("/wshome", websocket.Handler(wshome))
	http.Handle("/wsLogin", websocket.Handler(wsLogin))
	http.Handle("/wsNew", websocket.Handler(wsNew))
	http.Handle("/wsUpload", websocket.Handler(wsUpload))
	http.Handle("/wsEditVideo", websocket.Handler(wsEditVideo))
	http.Handle("/wsRegister", websocket.Handler(wsRegister))
	http.Handle("/wsGetVideo", websocket.Handler(wsGetVideo))
	http.Handle("/wsDeleteVideo", websocket.Handler(wsDeleteVideo))
	http.Handle("/wsGetMyVideos", websocket.Handler(wsGetMyVideos))
	http.Handle("/wsSearch", websocket.Handler(wsSearch))
	http.ListenAndServe(":8090", nil)
}
func wshome(ws *websocket.Conn) {
	defer ws.Close()
	b := make([]byte, 512)
	n, e := ws.Read(b)
	if testErr(e) {
		return
	}
	bi := BaseInfo{}
	e = json.Unmarshal(b[:n], &bi)
	if testErr(e) {
		return
	}
	gu, e := findUser(bson.M{"sessionid": bi.Info})
	if e != nil {
		returnInfo(ws, "ERR", "登录信息失效,请重新登陆")
		return
	}
	returnInfo(ws, "OK", gu.Email)
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
	defer ws.Close()
	b := make([]byte, 1024)
	l, e := ws.Read(b)
	if testErr(e) {
		returnInfo(ws, "ERR", e.Error())
		return
	}
	bi := BaseInfo{}
	e = json.Unmarshal(b[:l], &bi)
	if testErr(e) {
		returnInfo(ws, "ERR", e.Error())
		return
	}
	pageNum64, e := strconv.ParseInt(bi.Info, 10, 64)
	if testErr(e) {
		returnInfo(ws, "ERR", "String err")
		return
	}
	pageNum := int(pageNum64)
	s, e := mgo.Dial("127.0.0.1")
	checkErr(e)
	defer s.Close()
	cv := s.DB("theytube").C("videos")
	count, e := cv.Find(nil).Count()
	if e != nil {
		fmt.Println(e.Error())
	}
	var maxPage = getPages(count)
	if pageNum < 1 || pageNum > maxPage {
		returnInfo(ws, "ERR", "不存在的页面")
		return
	}
	vs := []Video{}
	e = cv.Find(nil).Limit(20).Sort("-uploadtime").Skip((pageNum - 1) * 20).All(&vs)
	if testErr(e) {
		returnInfo(ws, "ERR", e.Error())
		return
	}
	rdata := struct {
		BaseInfo
		Data []Video
	}{
		BaseInfo{"OK", strconv.FormatInt(int64(maxPage), 10)},
		vs,
	}
	data, e := json.Marshal(rdata)
	checkErr(e)
	ws.Write(data)
}
func wsGetMyVideos(ws *websocket.Conn) {
	defer ws.Close()
	b := make([]byte, 1024)
	l, e := ws.Read(b)
	if testErr(e) {
		returnInfo(ws, "ERR", e.Error())
		return
	}
	bi := BaseInfo{}
	e = json.Unmarshal(b[:l], &bi)
	if testErr(e) {
		returnInfo(ws, "ERR", e.Error())
		return
	}
	pageNum64, e := strconv.ParseInt(bi.Info, 10, 64)
	if testErr(e) {
		returnInfo(ws, "ERR", "String err")
		return
	}
	pageNum := int(pageNum64)
	s, e := mgo.Dial("127.0.0.1")
	checkErr(e)
	defer s.Close()
	cv := s.DB("theytube").C("videos")
	count, e := cv.Find(bson.M{"owner": bi.State}).Count()
	if e != nil {
		fmt.Println("count err ", e.Error())
	}
	var maxPage = getPages(count)
	if pageNum < 1 || pageNum > maxPage {
		returnInfo(ws, "ERR", "不存在的页面")
		return
	}
	vs := []Video{}
	e = cv.Find(bson.M{"owner": bi.State}).Limit(20).Sort("-uploadtime").Skip((pageNum - 1) * 20).All(&vs)
	if testErr(e) {
		returnInfo(ws, "ERR", e.Error())
		return
	}

	rdata := struct {
		BaseInfo
		Data []Video
	}{
		BaseInfo{"OK", strconv.FormatInt(int64(maxPage), 10)},
		vs,
	}
	data, e := json.Marshal(rdata)
	checkErr(e)
	ws.Write(data)
}

func wsUpload(ws *websocket.Conn) {
	defer ws.Close()
	b := make([]byte, 2048)
	l, e := ws.Read(b)
	if testErr(e) {
		return
	}
	v := Video{}
	e = json.Unmarshal(b[:l], &v)
	if testErr(e) {
		return
	}
	u, e := findUser(bson.M{"sessionid": v.Info})
	if e != nil {
		returnInfo(ws, "ERR", "您没有权限上传")
		return
	}
	v.Vid = NewToken()
	v.Owner = u.Email
	v.UploadTime = time.Now()
	v.Tags = splitHan(v.Title + v.Introduction)
	e = insertVideo(v)
	if testErr(e) {
		returnInfo(ws, "ERR", "上传失败:"+e.Error())
		return
	}
	returnInfo(ws, "OK", v.Vid)
}
func wsEditVideo(ws *websocket.Conn) {
	defer ws.Close()
	b := make([]byte, 2048)
	l, e := ws.Read(b)
	if testErr(e) {
		return
	}
	v := Video{}
	e = json.Unmarshal(b[:l], &v)
	if testErr(e) {
		return
	}
	u, e := findUser(bson.M{"sessionid": v.Info})
	if e != nil {
		returnInfo(ws, "ERR", "登录信息失效,请重新登陆")
		return
	}
	gv, e := findVideo(bson.M{"vid": v.Vid})
	if e != nil {
		returnInfo(ws, "ERR", "视频已被删除")
		return
	}
	if gv.Owner != u.Email {
		returnInfo(ws, "ERR", "您没有权限修改")
		return
	}
	s, e := mgo.Dial("127.0.0.1")
	checkErr(e)
	defer s.Close()
	e = s.DB("theytube").C("videos").Update(bson.M{"vid": v.Vid}, bson.M{"$set": bson.M{"title": v.Title, "tags": splitHan(v.Title + v.Introduction), "cover": v.Cover, "introduction": v.Introduction, "clips": v.Clips}})
	if e != nil {
		returnInfo(ws, "ERR", "修改失败:"+e.Error())
		return
	}
	returnInfo(ws, "OK", "")
}
func wsSearch(ws *websocket.Conn) {
	defer ws.Close()
	b := make([]byte, 2048)
	l, e := ws.Read(b)
	if testErr(e) {
		return
	}
	bi := BaseInfo{}
	e = json.Unmarshal(b[:l], &bi)
	if testErr(e) {
		return
	}
	pageNum64, e := strconv.ParseInt(bi.State, 10, 64)
	if testErr(e) {
		returnInfo(ws, "ERR", "String err")
		return
	}
	pageNum := int(pageNum64)
	s, e := mgo.Dial("127.0.0.1")
	checkErr(e)
	defer s.Close()
	cv := s.DB("theytube").C("videos")
	counter, e := cv.Find(bson.M{"$text": bson.M{"$search": splitHan(bi.Info)}}).Count()
	if e != nil {
		fmt.Println("counter err")
	}
	if counter == 0 {
		returnInfo(ws, "OK", "0个搜索结果")
		return
	}
	var maxPage = getPages(counter)
	if pageNum < 1 || pageNum > maxPage {
		returnInfo(ws, "ERR", "不存在的页面")
		return
	}
	vs := []Video{}
	e = cv.Find(bson.M{"$text": bson.M{"$search": splitHan(bi.Info)}}).Limit(20).Skip((pageNum - 1) * 20).All(&vs)
	if e != nil {
		returnInfo(ws, "ERR", e.Error())
		return
	}
	rdata := struct {
		BaseInfo
		Data []Video
	}{
		BaseInfo{"OK", strconv.FormatInt(int64(maxPage), 10)},
		vs,
	}
	data, e := json.Marshal(rdata)
	checkErr(e)
	ws.Write(data)
}
func wsDeleteVideo(ws *websocket.Conn) {
	defer ws.Close()
	b := make([]byte, 2048)
	l, e := ws.Read(b)
	if testErr(e) {
		return
	}
	bi := BaseInfo{}
	e = json.Unmarshal(b[:l], &bi)
	if testErr(e) {
		return
	}
	u, e := findUser(bson.M{"sessionid": bi.Info})
	if e != nil {
		returnInfo(ws, "ERR", "登录信息失效,请重新登陆")
		return
	}
	gv, e := findVideo(bson.M{"vid": bi.State})
	if e != nil {
		returnInfo(ws, "ERR", "视频已被删除")
		return
	}
	if gv.Owner != u.Email {
		returnInfo(ws, "ERR", "您没有权限修改")
		return
	}
	s, e := mgo.Dial("127.0.0.1")
	checkErr(e)
	e = s.DB("theytube").C("videos").Remove(bson.M{"vid": bi.State})
	if e != nil {
		returnInfo(ws, "ERR", "修改失败:"+e.Error())
		return
	}
	returnInfo(ws, "OK", "")
}
func wsGetVideo(ws *websocket.Conn) {
	defer ws.Close()
	b := make([]byte, 1024)
	l, e := ws.Read(b)
	if testErr(e) {
		return
	}
	bi := BaseInfo{}
	e = json.Unmarshal(b[:l], &bi)
	if testErr(e) {
		return
	}
	v, e := findVideo(bson.M{"vid": bi.Info})
	if testErr(e) {
		returnInfo(ws, "ERR", "该视频不存在")
		return
	}
	v.State = "OK"
	data, e := json.Marshal(v)
	if testErr(e) {
		returnInfo(ws, "ERR", e.Error())
		return
	}
	ws.Write(data)
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
		f, e1 := os.OpenFile("log.txt", os.O_APPEND|os.O_CREATE, 0666)
		if e1 != nil {
			fmt.Println("os.OpenFile failed : ", e1)
			return true
		}
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
func insertUser(u User) error {
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
func findVideo(m bson.M) (Video, error) {
	s, e := mgo.Dial("127.0.0.1")
	checkErr(e)
	defer s.Close()
	cv := s.DB("theytube").C("videos")
	v := Video{}
	e = cv.Find(m).One(&v)
	return v, e
}
func insertVideo(v Video) error {
	s, e := mgo.Dial("127.0.0.1")
	checkErr(e)
	defer s.Close()
	uv := s.DB("theytube").C("videos")
	e = uv.Insert(&v)
	return e
}
func splitHan(han string) string {
	bf := bytes.Buffer{}
	hz := regexp.MustCompile("[\\p{Han}]")
	for _, r := range han {
		str := string(r)
		if hz.MatchString(str) {
			bf.WriteString(str)
			bf.WriteString(" ")
			continue
		}
		bf.WriteString(str)
	}
	return bf.String()
}
func getPages(sum int) int {
	var pn = sum / 20
	var y = sum % 20
	if y == 0 {
		return pn
	}
	return pn + 1
}
