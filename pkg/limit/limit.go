package limit

import (
	"github.com/FXAZfung/random-image/internal/config"
	"golang.org/x/time/rate"
	"sync"
)

var ipLimiters sync.Map // 存储每个 IP 的限流器

func IsIpLimited(ip string) bool {
	if !config.MainConfig.Limit.Required {
		return false
	}
	limiter, ok := ipLimiters.Load(ip)
	if !ok {
		limiter = rate.NewLimiter(rate.Limit(config.MainConfig.Limit.Rate), config.MainConfig.Limit.Bucket) // 每秒允许 5 个请求，最多存储 2 个令牌
		ipLimiters.Store(ip, limiter)
		return false
	}
	return !limiter.(*rate.Limiter).Allow()
}
