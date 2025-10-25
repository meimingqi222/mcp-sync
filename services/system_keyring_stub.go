//go:build !windows

package services

// WindowsKeyring 是Windows平台的密钥环实现
// 在非Windows平台上这是一个stub实现，永远不会被使用
type WindowsKeyring struct{}

func (wk *WindowsKeyring) SetKey(service, keyName string, keyData []byte) error {
	// 这个函数永远不会被调用，因为在非Windows平台上不会创建WindowsKeyring实例
	panic("WindowsKeyring should not be used on non-Windows platforms")
}

func (wk *WindowsKeyring) GetKey(service, keyName string) ([]byte, error) {
	// 这个函数永远不会被调用，因为在非Windows平台上不会创建WindowsKeyring实例
	panic("WindowsKeyring should not be used on non-Windows platforms")
}

func (wk *WindowsKeyring) DeleteKey(service, keyName string) error {
	// 这个函数永远不会被调用，因为在非Windows平台上不会创建WindowsKeyring实例
	panic("WindowsKeyring should not be used on non-Windows platforms")
}
