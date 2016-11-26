package utils

import (
	"net/http"
	"strings"
)

func GetClientIp( r *http.Request ) string {
	if xf := r.Header.Get("X-Forwarded-For");xf != "" {
		return strings.Split(xf,",")[0]
	}
	ip := r.RemoteAddr
	pos := strings.LastIndex(ip, ":")
	var clientIp string
	if pos > 0 {
		clientIp = ip[0:pos]
	} else {
		clientIp = ip
	}
	return clientIp
}
