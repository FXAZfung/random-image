package utils

import "net/http"

func GetClientIp(request *http.Request) string {
	clientIP := request.Header.Get("X-Forwarded-For")
	if clientIP == "" {
		clientIP = request.RemoteAddr
	}
	return clientIP
}
