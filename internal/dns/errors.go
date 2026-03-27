package dns

import "fmt"

// BindError 表示 UDP 监听绑定失败（如端口已被占用）。
type BindError struct {
	Bind string
	Err  error
}

func (e *BindError) Error() string {
	return fmt.Sprintf("udp bind %s: %v", e.Bind, e.Err)
}

func (e *BindError) Unwrap() error {
	return e.Err
}
