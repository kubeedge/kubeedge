package session

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/go-chassis/go-chassis/core/common"
	"github.com/go-chassis/go-chassis/core/invocation"
	"github.com/go-chassis/go-chassis/pkg/util/httputil"
	"github.com/go-mesh/openlogging"
	"github.com/patrickmn/go-cache"
)

// ErrResponseNil used for to represent the error response, when it is nil
var ErrResponseNil = errors.New("can not set session, resp is nil")

// Cache session cache variable
var Cache *cache.Cache

// SessionStickinessCache key: go-chassisLB , value is cookie
var SessionStickinessCache *cache.Cache

func init() {
	Cache = initCache()
	SessionStickinessCache = initCache()
	cookieMap = make(map[string]string)
}
func initCache() *cache.Cache {
	var value *cache.Cache

	value = cache.New(3e+10, time.Second*30)
	return value
}

var cookieMap map[string]string

// getLBCookie gets cookie from local map
func getLBCookie(key string) string {
	return cookieMap[key]
}

// setLBCookie sets cookie to local map
func setLBCookie(key, value string) {
	cookieMap[key] = value
}

// GetContextMetadata gets data from context
func GetContextMetadata(ctx context.Context, key string) string {
	md, ok := ctx.Value(common.ContextHeaderKey{}).(map[string]string)
	if ok {
		return md[key]
	}
	return ""
}

// SetContextMetadata sets data to context
func SetContextMetadata(ctx context.Context, key string, value string) context.Context {
	md, ok := ctx.Value(common.ContextHeaderKey{}).(map[string]string)
	if !ok {
		md = make(map[string]string)
	}

	if md[key] == value {
		return ctx
	}

	md[key] = value
	return context.WithValue(ctx, common.ContextHeaderKey{}, md)
}

//GetSessionFromResp return session uuid in resp if there is
func GetSessionFromResp(cookieKey string, resp *http.Response) string {
	bytes := httputil.GetRespCookie(resp, cookieKey)
	if bytes != nil {
		return string(bytes)
	}
	return ""
}

// SaveSessionIDFromContext check session id in response ctx and save it to session storage
func SaveSessionIDFromContext(ctx context.Context, ep string, autoTimeout int) context.Context {

	timeValue := time.Duration(autoTimeout) * time.Second

	sessionIDStr := GetContextMetadata(ctx, common.LBSessionID)
	if sessionIDStr != "" {
		cookieKey := strings.Split(sessionIDStr, "=")
		if len(cookieKey) > 1 {
			sessionIDStr = cookieKey[1]
		}
	}

	ClearExpired()
	var sessBool bool
	if sessionIDStr != "" {
		_, sessBool = Cache.Get(sessionIDStr)
	}

	if sessionIDStr != "" && sessBool {
		cookie := common.LBSessionID + "=" + sessionIDStr
		setLBCookie(common.LBSessionID, cookie)
		Save(sessionIDStr, ep, timeValue)
		return ctx
	}

	sessionIDValue, err := GenerateSessionID()
	if err != nil {
		openlogging.Warn("session id generate fail, it is impossible", openlogging.WithTags(
			openlogging.Tags{
				"err": err.Error(),
			}))
	}
	cookie := common.LBSessionID + "=" + sessionIDValue
	setLBCookie(common.LBSessionID, cookie)
	Save(sessionIDValue, ep, timeValue)
	return SetContextMetadata(ctx, common.LBSessionID, cookie)
}

//Temporary responsewriter for SetCookie
type cookieResponseWriter http.Header

// Header implements ResponseWriter Header interface
func (c cookieResponseWriter) Header() http.Header {
	return http.Header(c)
}

//Write is a dummy function
func (c cookieResponseWriter) Write([]byte) (int, error) {
	panic("ERROR")
}

//WriteHeader is a dummy function
func (c cookieResponseWriter) WriteHeader(int) {
	panic("ERROR")
}

//setCookie appends cookie with already present cookie with ';' in between
func setCookie(resp *http.Response, value string) {

	newCookie := common.LBSessionID + "=" + value
	oldCookie := string(httputil.GetRespCookie(resp, common.LBSessionID))

	if oldCookie != "" {
		//If cookie is already set, append it with ';'
		newCookie = newCookie + ";" + oldCookie
	}

	c1 := http.Cookie{Name: common.LBSessionID, Value: newCookie}

	w := cookieResponseWriter(resp.Header)
	http.SetCookie(w, &c1)
}

// SaveSessionIDFromHTTP check session id
func SaveSessionIDFromHTTP(ep string, autoTimeout int, resp *http.Response, req *http.Request) {
	if resp == nil {
		openlogging.GetLogger().Warnf("", ErrResponseNil)
		return
	}

	timeValue := time.Duration(autoTimeout) * time.Second

	var sessionIDStr string

	if c, err := req.Cookie(common.LBSessionID); err != http.ErrNoCookie && c != nil {
		sessionIDStr = c.Value
	}

	ClearExpired()
	var sessBool bool
	if sessionIDStr != "" {
		_, sessBool = Cache.Get(sessionIDStr)
	}

	valueChassisLb := GetSessionFromResp(common.LBSessionID, resp)
	//if session is in resp, then just save it
	if valueChassisLb != "" {
		Save(valueChassisLb, ep, timeValue)
	} else if sessionIDStr != "" && sessBool {
		setCookie(resp, sessionIDStr)
		Save(sessionIDStr, ep, timeValue)
	} else {
		sessionIDValue, err := GenerateSessionID()
		if err != nil {
			openlogging.Warn("session id generate fail, it is impossible", openlogging.WithTags(
				openlogging.Tags{
					"err": err.Error(),
				}))
		}
		setCookie(resp, sessionIDValue)
		Save(sessionIDValue, ep, timeValue)
	}

}

//GenerateSessionID generate a session id
func GenerateSessionID() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// DeletingKeySuccessiveFailure deleting key successes and failures
func DeletingKeySuccessiveFailure(resp *http.Response) {
	Cache.DeleteExpired()
	if resp == nil {
		valueChassisLb := getLBCookie(common.LBSessionID)
		if valueChassisLb != "" {
			cookieKey := strings.Split(valueChassisLb, "=")
			if len(cookieKey) > 1 {
				Delete(cookieKey[1])
				setLBCookie(common.LBSessionID, "")
			}
		}
		return
	}

	valueChassisLb := GetSessionFromResp(common.LBSessionID, resp)
	if valueChassisLb != "" {
		cookieKey := strings.Split(valueChassisLb, "=")
		if len(cookieKey) > 1 {
			Delete(cookieKey[1])
		}
	}
}

// GetSessionCookie getting session cookie
func GetSessionCookie(ctx context.Context, resp *http.Response) string {
	if ctx != nil {
		return GetContextMetadata(ctx, common.LBSessionID)
	}

	if resp == nil {
		openlogging.GetLogger().Warnf("", ErrResponseNil)
		return ""
	}

	valueChassisLb := GetSessionFromResp(common.LBSessionID, resp)
	if valueChassisLb != "" {
		return valueChassisLb
	}

	return ""
}

// AddSessionStickinessToCache add new cookie or refresh old cookie
func AddSessionStickinessToCache(cookie, namespace string) {
	key := getSessionStickinessCacheKey(namespace)
	value, ok := SessionStickinessCache.Get(key)
	if !ok || value == nil {
		SessionStickinessCache.Set(key, cookie, 0)
		return
	}
	s, ok := value.(string)
	if !ok {
		SessionStickinessCache.Set(key, cookie, 0)
		return
	}
	if cookie != "" && s != cookie {
		SessionStickinessCache.Set(key, cookie, 0)
	}
}

// GetSessionID get sessionID from cache
func GetSessionID(namespace string) string {

	value, ok := SessionStickinessCache.Get(getSessionStickinessCacheKey(namespace))
	if !ok || value == nil {
		openlogging.GetLogger().Warn("not sessionID in cache")
		return ""
	}
	s, ok := value.(string)
	if !ok {
		openlogging.GetLogger().Warn("get sessionID from cache failed")
		return ""
	}
	return s
}
func getSessionStickinessCacheKey(namespace string) string {
	if namespace == "" {
		namespace = common.SessionNameSpaceDefaultValue
	}
	return strings.Join([]string{common.LBSessionID, namespace}, "|")
}

// GetSessionIDFromInv when use  SessionStickiness , get session id from inv
func GetSessionIDFromInv(inv invocation.Invocation, key string) string {
	var metadata interface{}
	switch inv.Reply.(type) {
	case *http.Response:
		resp := inv.Reply.(*http.Response)
		value := httputil.GetRespCookie(resp, key)
		if string(value) != "" {
			metadata = string(value)
		}
	default:
		value := GetContextMetadata(inv.Ctx, key)
		if value != "" {
			metadata = value
		}
	}
	if metadata == nil {
		metadata = ""
	}
	return metadata.(string)
}
