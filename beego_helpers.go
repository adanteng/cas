package cas

import (
	"fmt"
	"net/http"

	"github.com/golang/glog"
)

// 这里仅封装了userName, 如果需要其他信息可以后续添加
type BeegoCASData struct {
	userName string
}

func (bcd *BeegoCASData) GetUserName() string {
	return bcd.userName
}

// http.Handler的ServeHTTP方法封装在beego内部，cas.v1仅对外部开放ServeHTTP(上面的方法)，不满足需求
func ServeBeego(w http.ResponseWriter, r *http.Request, c *Client) *BeegoCASData {
	if glog.V(2) {
		glog.Infof("cas: handling %v request for %v", r.Method, r.URL)
	}

	// 参考clientHandler中的ServeHTTP方法
	setClient(r, c)
	defer clear(r)

	// 这块不是很理解，但是考虑到支持single logout，还是抄过来，后续补充说明
	if isSingleLogoutRequest(r) {
		performSingleLogout(w, r, c)
		return nil
	}

	c.getSession(w, r)

	// 下面逻辑本来是作者提供给cas库外部调用的，原因是上面会在方法返回后clear掉r，导致下面的方法nil异常
	// 原来的逻辑没有问题的原因是作者clientHandler封装http.Handler的ServeHTTP方法，在同一个方法内
	if !IsAuthenticated(r) {
		RedirectToLogin(w, r)
		return nil
	}

	if r.URL.Path == "/logout" {
		RedirectToLogout(w, r)
		return nil
	}

	return &BeegoCASData{
		userName: Username(r),
	}
}

func performSingleLogout(w http.ResponseWriter, r *http.Request, c *Client) {
	rawXML := r.FormValue("logoutRequest")
	logoutRequest, err := parseLogoutRequest([]byte(rawXML))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := c.tickets.Delete(logoutRequest.SessionIndex); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	c.deleteSession(logoutRequest.SessionIndex)

	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "OK")
}
