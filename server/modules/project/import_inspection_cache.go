package project

import (
	"sync"
	"time"
)

type importInspectionCache struct {
	mu       sync.Mutex
	sessions map[string]importInspectionSession
}

// newImportInspectionCache 创建并返回一个新的导入检查缓存，并初始化空的会话映射。
func newImportInspectionCache() *importInspectionCache {
	return &importInspectionCache{sessions: make(map[string]importInspectionSession)}
}

func (c *importInspectionCache) storeSession(session importInspectionSession) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pruneLocked(time.Now().UTC())
	c.sessions[session.ID] = session
}

func (c *importInspectionCache) lookupSession(id string) (importInspectionSession, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.pruneLocked(time.Now().UTC())
	session, ok := c.sessions[id]
	return session, ok
}

func (c *importInspectionCache) pruneLocked(now time.Time) {
	for key, session := range c.sessions {
		if now.After(session.ExpiresAt) {
			next := make(map[string]importInspectionSession, len(c.sessions)-1)
			for existingKey, existingSession := range c.sessions {
				if existingKey == key {
					continue
				}
				next[existingKey] = existingSession
			}
			c.sessions = next
		}
	}
}
