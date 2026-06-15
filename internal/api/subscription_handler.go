package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/phoenix-panel/phoenix/internal/links"
	"github.com/phoenix-panel/phoenix/internal/service"
)

// SubscriptionHandler serves the public subscription endpoint consumed by
// proxy clients. It requires no admin auth — the unguessable token IS the auth.
type SubscriptionHandler struct {
	subs *service.SubscriptionService
}

// NewSubscriptionHandler constructs a SubscriptionHandler.
func NewSubscriptionHandler(subs *service.SubscriptionService) *SubscriptionHandler {
	return &SubscriptionHandler{subs: subs}
}

// Get handles GET /sub/:token. Output format is chosen by ?format=base64|plain
// (default base64). Usage and expiry are advertised via the standard
// "Subscription-Userinfo" header that v2ray-family clients understand.
func (h *SubscriptionHandler) Get(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		fail(c, http.StatusBadRequest, "missing token")
		return
	}

	data, err := h.subs.Build(c.Request.Context(), token)
	if err != nil {
		// Avoid distinguishing "bad token" from other errors to clients.
		failErr(c, err)
		return
	}

	format := links.SubFormat(c.DefaultQuery("format", string(links.FormatBase64)))
	body := links.Render(data.URIs, format)

	h.setUserinfoHeader(c, data)
	c.Header("Profile-Update-Interval", "12")
	c.Header("Content-Disposition", `inline; filename="`+data.User.Username+`"`)

	if format == links.FormatPlain {
		c.String(http.StatusOK, body)
		return
	}
	c.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(body))
}

// setUserinfoHeader emits the Subscription-Userinfo header:
//   upload=<bytes>; download=<bytes>; total=<bytes>; expire=<unix>
func (h *SubscriptionHandler) setUserinfoHeader(c *gin.Context, data *service.SubscriptionData) {
	u := data.User
	var expire int64 // 0 means no expiry per the de-facto spec
	if u.ExpireAt != nil {
		expire = u.ExpireAt.Unix()
	}
	info := "upload=" + strconv.FormatInt(u.UsedUp, 10) +
		"; download=" + strconv.FormatInt(u.UsedDown, 10) +
		"; total=" + strconv.FormatInt(u.DataLimit, 10) +
		"; expire=" + strconv.FormatInt(expire, 10)
	c.Header("Subscription-Userinfo", info)
}
