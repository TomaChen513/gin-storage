package controller

import (
	"encoding/json"
	"file-store/lib"
	"file-store/model"

	"file-store/util"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

type PrivateInfo struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    string `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenId       string `json:"openid"`
}

type GUserInfo struct {
	Id         int `json:"id"`
	Avatar_url string `json:"avatar_url"`
	Login      string `json:"login"`
}

// 登录页
func Login(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", nil)
}

// 处理登录
func HandlerLogin(c *gin.Context) {
	conf := lib.LoadServerConfig()
	state := "xxxxxxx"
	url := "https://github.com/login/oauth/authorize?client_id=" + conf.AppId + "&redirect_uri=" + conf.RedirectURI + "&state=" + state
	c.Redirect(http.StatusMovedPermanently, url)
}

// 获取access_token
func GetGithubToken(c *gin.Context) {
	conf := lib.LoadServerConfig()
	code := c.Query("code")

	loginUrl := "https://github.com/login/oauth/access_token?client_id=" + conf.AppId + "&client_secret=" + conf.AppKey + "&redirect_uri=" + conf.RedirectURI + "&code=" + code

	response, err := http.PostForm(loginUrl, url.Values{
		"client_id":     {conf.AppId},
		"client_secret": {conf.AppKey},
		"redirect_uri":  {conf.RedirectURI},
		"code":          {code},
	})

	if err != nil {
		fmt.Println("请求错误", err.Error())
		return
	}
	defer response.Body.Close()

	bs, _ := ioutil.ReadAll(response.Body)
	body := string(bs)
	resultMap := util.ConvertToMap(body)


	info := &PrivateInfo{}
	info.AccessToken = resultMap["access_token"]

	GetOpenId(info, c)
}

func GetOpenId(info *PrivateInfo, c *gin.Context) {

	client := &http.Client{}
	reqest, err := http.NewRequest("GET", "https://api.github.com/user", nil)

	if err != nil {
		panic(err)
	}

	reqest.Header.Add("Authorization", "token "+info.AccessToken)
	resp, err := client.Do(reqest)

	if err != nil {
		fmt.Println("GetMessage Err", err.Error())
		return
	}

	defer resp.Body.Close()

	gUser:=GUserInfo{}
	
	err = json.NewDecoder(resp.Body).Decode(&gUser)
	if err != nil {
		panic(err)
	}
	id:=strconv.Itoa(gUser.Id)

	//创建一个token
	hashToken := util.EncodeMd5("token" + strconv.FormatInt(time.Now().Unix(),10) + id)

	//存入redis
	if err := lib.SetKey(hashToken, strconv.Itoa(gUser.Id), 24*3600); err != nil {
		fmt.Println("Redis Set Err:", err.Error())
		return
	}
	//设置cookie
	c.SetCookie("Token", hashToken, 3600*24, "", "", false,true)

	if ok := model.QueryUserExists(id); ok { //用户存在直接登录
		//登录成功重定向到首页
		c.Redirect(http.StatusMovedPermanently, "/cloud/index")
	} else {
		model.CreateUser(id, gUser.Login, gUser.Avatar_url)
		//登录成功重定向到首页
		c.Redirect(http.StatusMovedPermanently, "/cloud/index")
	}
}

// 退出登录
func Logout(c *gin.Context) {
	token, err := c.Cookie("Token")
	if err != nil {
		fmt.Println("cookie", err.Error())
	}

	if err := lib.DelKey(token); err != nil {
		fmt.Println("Del Redis Err:", err.Error())
	}

	c.SetCookie("Token", "", 0, "/", "localhost", false, false)
	c.Redirect(http.StatusFound, "/")
}
