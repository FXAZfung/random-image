// /home/user/random-image/internal/limiter/limiter.go
package limiter

import (
	"sync"
	"time"
)

// visitor 单个 IP 的访问记录
type visitor struct {
	// 令牌桶
	tokens     float64
	maxTokens  float64
	refillRate float64 // 每秒补充的令牌数
	lastRefill time.Time

	// 滑动窗口计数（用于封禁判断）
	hitCount    int
	windowStart time.Time

	// 封禁状态
	banned      bool
	bannedUntil time.Time
}

// Limiter IP 限流器
// 双重机制：令牌桶平滑限流 + 窗口计数自动封禁
type Limiter struct {
	mu              sync.Mutex
	visitors        map[string]*visitor
	rate            int           // 每分钟允许请求数
	burst           int           // 令牌桶容量（突发上限）
	banThreshold    int           // 每分钟超过此数触发封禁
	banDuration     time.Duration // 封禁时长
	cleanupInterval time.Duration // 清理过期记录间隔
}

// New 创建限流器
func New(rate, burst, banThreshold int, banDuration, cleanupInterval time.Duration) *Limiter {
	l := &Limiter{
		visitors:        make(map[string]*visitor),
		rate:            rate,
		burst:           burst,
		banThreshold:    banThreshold,
		banDuration:     banDuration,
		cleanupInterval: cleanupInterval,
	}

	go l.cleanupLoop()
	return l
}

// Allow 检查指定 IP 是否允许本次请求
// 返回 true 表示放行，false 表示拒绝
func (l *Limiter) Allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()

	v, ok := l.visitors[ip]
	if !ok {
		v = &visitor{
			tokens:      float64(l.burst),
			maxTokens:   float64(l.burst),
			refillRate:  float64(l.rate) / 60.0,
			lastRefill:  now,
			hitCount:    0,
			windowStart: now,
		}
		l.visitors[ip] = v
	}

	// 1. 检查封禁状态
	if v.banned {
		if now.After(v.bannedUntil) {
			// 封禁到期，解除
			v.banned = false
			v.hitCount = 0
			v.windowStart = now
			v.tokens = float64(l.burst)
			v.lastRefill = now
		} else {
			return false
		}
	}

	// 2. 滑动窗口计数（1 分钟窗口）
	if now.Sub(v.windowStart) > time.Minute {
		v.hitCount = 0
		v.windowStart = now
	}
	v.hitCount++

	// 超过封禁阈值，自动封禁
	if l.banThreshold > 0 && v.hitCount > l.banThreshold {
		v.banned = true
		v.bannedUntil = now.Add(l.banDuration)
		return false
	}

	// 3. 令牌桶：补充令牌
	elapsed := now.Sub(v.lastRefill).Seconds()
	v.tokens += elapsed * v.refillRate
	if v.tokens > v.maxTokens {
		v.tokens = v.maxTokens
	}
	v.lastRefill = now

	// 消耗一个令牌
	if v.tokens < 1.0 {
		return false
	}
	v.tokens -= 1.0
	return true
}

// IsBanned 检查指定 IP 当前是否处于封禁状态
func (l *Limiter) IsBanned(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	v, ok := l.visitors[ip]
	if !ok {
		return false
	}
	return v.banned && time.Now().Before(v.bannedUntil)
}

// Stats 返回统计：总访客数、当前被封禁数
func (l *Limiter) Stats() (total int, banned int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for _, v := range l.visitors {
		total++
		if v.banned && now.Before(v.bannedUntil) {
			banned++
		}
	}
	return
}

// cleanupLoop 定期清理不活跃的访客记录，防止内存泄漏
func (l *Limiter) cleanupLoop() {
	ticker := time.NewTicker(l.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		l.cleanup()
	}
}

func (l *Limiter) cleanup() {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	for ip, v := range l.visitors {
		if v.banned {
			// 封禁已过期，清除
			if now.After(v.bannedUntil) {
				delete(l.visitors, ip)
			}
			continue
		}
		// 未封禁且超过 5 分钟没有活动，清除
		if now.Sub(v.lastRefill) > 5*time.Minute {
			delete(l.visitors, ip)
		}
	}
}
