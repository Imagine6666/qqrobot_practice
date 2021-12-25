package qqbot

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"regexp"
	"strconv"
	"time"
)

type User struct {
	Client                       http.Client `json:"-"`
	Captcha                      image.Image `json:"-"`
	Vfwebqq, Ptwebqq, Pssesionid string
	Uin                          int
}

func NewUser() *User {
	jar, _ := cookiejar.New(nil)
	client := http.Client{Jar: jar, Timeout: 20 * time.Second}
	return &User{Client: client}
}

func (user *User) update() {
	user.Client.Jar, _ = cookiejar.New(nil)
	user.updateCookie()
	user.updateCaptcha()
}

func (this *User) updateCookie() {
	req, _ := http.NewRequest("GET", "https://ui.ptlogin2.qq.com/cgi-bin/login?daid=164&target=self&style=16&mibao_css=m_webqq&appid=501004106&enable_qlogin=0&no_verifyimg=1&s_url=http%3A%2F%2Fw.qq.com/proxy.html&f_url=loginerroralert&strong_login=1&login_state=10&t=20130723001&f_qr=0", nil)
	res, _ := this.Client.Do(req)
	defer res.Body.Close()
}

func (this *User) updateCaptcha() {
	req, _ := http.NewRequest("GET", "https://ssl.ptlogin2.qq.com/ptqrshow?appid=501004106&e=0&l=M&s=5&d=72&v=4", nil)
	req.Header.Add("Referer", "https://ui.ptlogin2.qq.com/cgi-bin/login?daid=164&target=self&style=16&mibao_css=m_webqq&appid=501004106&enable_qlogin=0&no_verifyimg=1&s_url=http%3A%2F%2Fw.qq.com/proxy.html&f_url=loginerroralert&strong_login=1&login_state=10&t=20130723001&f_qr=0")
	res, _ := this.Client.Do(req)
	defer res.Body.Close()
	img, _ := png.Decode(res.Body)
	this.Captcha = img
}

func (this *User) checkVerify() (int, string, string, string) {
	req, _ := http.NewRequest("GET", "https://ssl.ptlogin2.qq.com/ptqrlogin?webqq_type=10&remember_uin=1&login2qq=1&aid=501004106&u1=http%3A%2F%2Fw.qq.com%2Fproxy.html%3Flogin2qq%3D1%26webqq_type%3D10&ptredirect=0&ptlang=2052&daid=164&from_ui=1&pttype=1&dumy=&fp=loginerroralert&action=0-0-112024&mibao_css=m_webqq&t=undefined&g=1&js_type=0&js_ver=10175&login_sig=&pt_randsalt=0", nil)
	req.Header.Add("Referer", "https://ui.ptlogin2.qq.com/cgi-bin/login?daid=164&target=self&style=16&mibao_css=m_webqq&appid=501004106&enable_qlogin=0&no_verifyimg=1&s_url=http%3A%2F%2Fw.qq.com%2Fproxy.html&f_url=loginerroralert&strong_login=1&login_state=10&t=20131024001")
	res, _ := this.Client.Do(req)
	defer res.Body.Close()
	if res.StatusCode != 200 {
		return -1, "", "", ""
	}
	data, _ := ioutil.ReadAll(res.Body)
	reg := regexp.MustCompile(`'(.*?)'`)
	strs := reg.FindAllStringSubmatch(string(data), -1)
	status, _ := strconv.Atoi(strs[0][1])
	link, info, name := strs[2][1], strs[4][1], strs[5][1]
	return status, link, info, name
}

func (this *User) updatePtwebqq(u string) {
	req, _ := http.NewRequest("GET", u, nil)
	res, _ := this.Client.Do(req)
	res.Body.Close()
}

type txResult struct {
	Retcode int
	Result  struct {
		Vfwebqq    string
		Uin        int
		Psessionid string
	}
}

func (this *User) updateVfwebqq() txResult {
	u, _ := url.Parse("http://s.web2.qq.com/api/getvfwebqq")
	var ptwebqq string
	for _, it := range this.Client.Jar.Cookies(u) {
		if it.Name == "ptwebqq" {
			ptwebqq = it.Value
		}
	}
	req, _ := http.NewRequest("GET", "http://s.web2.qq.com/api/getvfwebqq?ptwebqq="+ptwebqq+"&clientid=53999199&psessionid=&t=1473584468629", nil)
	req.Header.Add("Referer", "http://s.web2.qq.com/proxy.html?v=20130916001&callback=1&id=1")
	res, _ := this.Client.Do(req)
	var result txResult
	data, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()
	json.Unmarshal(data, &result)
	if result.Retcode == 0 {
		this.Vfwebqq = result.Result.Vfwebqq
		this.Ptwebqq = ptwebqq
	}
	return result
}

func (this *User) updateUin() txResult {
	req, _ := http.NewRequest("POST", "http://d1.web2.qq.com/channel/login2", bytes.NewReader([]byte("r=%7B%22ptwebqq%22%3A%22"+this.Ptwebqq+"%22%2C%22clientid%22%3A53999199%2C%22psessionid%22%3A%22%22%2C%22status%22%3A%22online%22%7D")))
	req.Header.Add("Referer", "http://s.web2.qq.com/proxy.html?v=20130916001&callback=1&id=1")
	res, _ := this.Client.Do(req)
	data, _ := ioutil.ReadAll(res.Body)
	res.Body.Close()
	var result txResult
	json.Unmarshal(data, &result)
	if result.Retcode == 0 {
		this.Uin = result.Result.Uin
		this.Pssesionid = result.Result.Psessionid
	}
	return result
}

func (this *User) Login() error {
	if res := this.updateVfwebqq(); res.Retcode != 0 {
		fmt.Println(res)
		return errors.New("登录失败")
	}
	if res := this.updateUin(); res.Retcode != 0 {
		fmt.Println(res)
		return errors.New("登录失败")
	}
	return nil
}

func (this *User) WaitVerify() chan image.Image {
	c := make(chan image.Image)
	go func() {
		for {
			status, rawurl, info, _ := this.checkVerify()
			fmt.Println(info)
			if status == 0 {
				this.updatePtwebqq(rawurl)
				close(c)
				break
			}
			if status == 65 || status == -1 {
				this.update()
				c <- this.Captcha
			}
			time.Sleep(1 * time.Second)
		}
	}()
	return c
}
