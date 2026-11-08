package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"zoo-go/internal/domain"
)

func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req domain.LoginRequest
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	openID, err := s.resolveOpenID(req)
	if err != nil {
		writeError(w, err)
		return
	}
	session, err := s.store.Login(openID, req.Nickname, req.Avatar, "", s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, session)
}

func (s *Server) resolveOpenID(req domain.LoginRequest) (string, error) {
	if s.config.WeChatAppID != "" && s.config.WeChatAppSecret != "" {
		if strings.TrimSpace(req.Code) == "" {
			return "", &httpError{status: http.StatusBadRequest, code: "WECHAT_CODE_REQUIRED", message: "微信登录凭证不能为空"}
		}
		return s.wechatOpenID(req.Code)
	}
	if !s.config.DevAuth {
		return "", &httpError{status: http.StatusServiceUnavailable, code: "AUTH_NOT_CONFIGURED", message: "微信登录尚未配置"}
	}
	device := strings.TrimSpace(req.DeviceID)
	if device == "" {
		device = strings.TrimSpace(req.Code)
	}

	if device == "" {
		return "", &httpError{status: http.StatusBadRequest, code: "DEVICE_ID_REQUIRED", message: "开发登录需要设备标识"}
	}
	return "dev:" + device, nil
}

type httpError struct {
	status  int
	code    string
	message string
}

func (e *httpError) Error() string {
	return e.code + ": " + e.message
}

func (s *Server) wechatOpenID(code string) (string, error) {
	endpoint := "https://api.weixin.qq.com/sns/jscode2session?" + url.Values{
		"appid":      {s.config.WeChatAppID},
		"secret":     {s.config.WeChatAppSecret},
		"js_code":    {code},
		"grant_type": {"authorization_code"},
	}.Encode()
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return "", &httpError{status: http.StatusBadGateway, code: "WECHAT_LOGIN_FAILED", message: "微信登录服务暂时不可用"}
	}
	defer resp.Body.Close()
	var payload struct {
		OpenID  string `json:"openid"`
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", &httpError{status: http.StatusBadGateway, code: "WECHAT_LOGIN_FAILED", message: "微信登录响应无效"}
	}
	if payload.OpenID == "" {
		return "", &httpError{
			status: http.StatusUnauthorized, code: "WECHAT_LOGIN_FAILED",
			message: fmt.Sprintf("微信登录失败：%s", payload.ErrMsg),
		}
	}
	return payload.OpenID, nil
}

func (s *Server) authMe(w http.ResponseWriter, r *http.Request) {
	writeData(w, currentUser(r))
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request) {
	var req domain.UpdateProfileRequest
	if err := decode(r, &req); err != nil {
		writeError(w, err)
		return
	}
	profile, err := s.store.UpdateProfile(currentUser(r).ID, req, s.now())
	if err != nil {
		writeError(w, err)
		return
	}
	writeData(w, profile)
}

func (s *Server) logout(w http.ResponseWriter, r *http.Request) {
	if err := s.store.Logout(bearerToken(r)); err != nil {
		writeError(w, err)
		return
	}
	writeData(w, map[string]bool{"ok": true})
}
