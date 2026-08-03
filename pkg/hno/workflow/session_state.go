package workflow

import (
	"encoding/json"
	"sync"
)

// SessionState provides thread-safe session state management for workflows
// SessionState 为工作流提供线程安全的会话状态管理
type SessionState struct {
	mu   sync.RWMutex
	data map[string]interface{}
}

// NewSessionState creates a new session state
// NewSessionState 创建新的会话状态
func NewSessionState() *SessionState {
	return &SessionState{
		data: make(map[string]interface{}),
	}
}

// Set stores a value in the session state
// Set 在会话状态中存储值
func (ss *SessionState) Set(key string, value interface{}) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.data[key] = value
}

// Get retrieves a value from the session state
// Get 从会话状态检索值
func (ss *SessionState) Get(key string) (interface{}, bool) {
	ss.mu.RLock()
	defer ss.mu.RUnlock()
	val, ok := ss.data[key]
	return val, ok
}

// GetAll returns a copy of all session state data
// GetAll 返回所有会话状态数据的副本
func (ss *SessionState) GetAll() map[string]interface{} {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	copy := make(map[string]interface{}, len(ss.data))
	for k, v := range ss.data {
		copy[k] = v
	}
	return copy
}

// Delete removes a key from the session state
// Delete 从会话状态中删除键
func (ss *SessionState) Delete(key string) {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	delete(ss.data, key)
}

// Clear removes all keys from the session state
// Clear 清空会话状态中的所有键
func (ss *SessionState) Clear() {
	ss.mu.Lock()
	defer ss.mu.Unlock()
	ss.data = make(map[string]interface{})
}

// Clone creates a deep copy of the session state
// Clone 创建会话状态的深拷贝
func (ss *SessionState) Clone() *SessionState {
	ss.mu.RLock()
	defer ss.mu.RUnlock()

	cloned := NewSessionState()
	for k, v := range ss.data {
		// Deep copy using JSON serialization
		// 使用JSON序列化进行深拷贝
		jsonBytes, err := json.Marshal(v)
		if err != nil {
			// Fallback to shallow copy if marshaling fails
			// 如果序列化失败则回退到浅拷贝
			cloned.data[k] = v
			continue
		}

		var clonedValue interface{}
		if err := json.Unmarshal(jsonBytes, &clonedValue); err != nil {
			cloned.data[k] = v
			continue
		}

		cloned.data[k] = clonedValue
	}

	return cloned
}

// Merge merges another session state into this one
// Merge 将另一个会话状态合并到当前状态
// Later values overwrite earlier ones (last-write-wins)
// 后面的值覆盖前面的值（最后写入获胜）
func (ss *SessionState) Merge(other *SessionState) {
	if other == nil {
		return
	}

	ss.mu.Lock()
	defer ss.mu.Unlock()

	other.mu.RLock()
	defer other.mu.RUnlock()

	for k, v := range other.data {
		ss.data[k] = v
	}
}

// MergeParallelSessionStates merges multiple session states from parallel execution
// MergeParallelSessionStates 合并来自并行执行的多个会话状态
// This function collects all changes from parallel branches and merges them
// 此函数收集并行分支的所有变更并合并它们
func MergeParallelSessionStates(original *SessionState, modified []*SessionState) *SessionState {
	if original == nil {
		original = NewSessionState()
	}

	// Create a new session state for the merged result
	// 为合并结果创建新的会话状态
	merged := original.Clone()

	// Merge each modified state
	// 合并每个修改的状态
	for _, modifiedState := range modified {
		if modifiedState == nil {
			continue
		}

		// Get all changes from this parallel branch
		// 获取此并行分支的所有变更
		modifiedState.mu.RLock()
		for key, value := range modifiedState.data {
			// Check if this is a new key or changed value
			// 检查这是否是新键或已更改的值
			originalValue, existsInOriginal := original.Get(key)

			// If key doesn't exist in original or value changed, apply it
			// 如果键在原始状态中不存在或值已更改，则应用它
			if !existsInOriginal || !deepEqual(originalValue, value) {
				merged.Set(key, value)
			}
		}
		modifiedState.mu.RUnlock()
	}

	return merged
}

// deepEqual checks if two values are equal (simple comparison)
// deepEqual 检查两个值是否相等（简单比较）
func deepEqual(a, b interface{}) bool {
	// For simple comparison, we use JSON marshaling
	// 对于简单比较，我们使用JSON序列化
	aBytes, aErr := json.Marshal(a)
	bBytes, bErr := json.Marshal(b)

	if aErr != nil || bErr != nil {
		return false
	}

	return string(aBytes) == string(bBytes)
}

// ToMap returns the session state as a regular map (not thread-safe)
// ToMap 将会话状态作为常规map返回（不是线程安全的）
func (ss *SessionState) ToMap() map[string]interface{} {
	return ss.GetAll()
}

// FromMap creates a session state from a map
// FromMap 从map创建会话状态
func FromMap(data map[string]interface{}) *SessionState {
	ss := NewSessionState()
	if data != nil {
		ss.data = make(map[string]interface{}, len(data))
		for k, v := range data {
			ss.data[k] = v
		}
	}
	return ss
}
