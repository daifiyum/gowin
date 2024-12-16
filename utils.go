// 一些工具函数
package gowin

import (
	"syscall"
	"unicode/utf16"
	"unsafe"
)

// 加载本地.ico图标
func LoadIconFromFile(iconPath string) (syscall.Handle, error) {
	iconPathPtr, _ := syscall.UTF16PtrFromString(iconPath)

	ret, _, err := LoadImage.Call(
		0,
		uintptr(unsafe.Pointer(iconPathPtr)),
		uintptr(IMAGE_ICON),
		0,
		0,
		uintptr(LR_LOADFROMFILE|LR_DEFAULTSIZE),
	)

	if ret == 0 {
		return 0, err
	}

	return syscall.Handle(ret), nil
}

// LOWORD
func LOWORD(l uint64) uint32 {
	return uint32(l & 0xFFFFFFFF)
}

// HIWORD
func HIWORD(l uint64) uint32 {
	return uint32((l >> 32) & 0xFFFFFFFF)
}

// 将字符串转换为 UTF-16 编码
func SetUTF16String(dst interface{}, src string) {
	utf16Slice := utf16.Encode([]rune(src))
	switch d := dst.(type) {
	case *[64]uint16:
		copy(d[:], utf16Slice)
	case *[256]uint16:
		copy(d[:], utf16Slice)
	default:
		panic("unsupported array type")
	}
}

// 获取复选框状态
func CheckItem(hMenu syscall.Handle, uIDCheckItem uint32) bool {
	ret, _, _ := CheckMenuItem.Call(
		uintptr(hMenu),
		uintptr(uIDCheckItem),
		uintptr(MF_BYCOMMAND),
	)

	return ret == MF_CHECKED
}

// 修改复选框状态
func SetCheckItem(hMenu syscall.Handle, uIDCheckItem uint32, uCheck uint32) {
	CheckMenuItem.Call(
		uintptr(hMenu),
		uintptr(uIDCheckItem),
		uintptr(uCheck),
	)
}

// 托盘提示字符处理
func TipFromStr(s string) [128]uint16 {
	utf16Tip, _ := syscall.UTF16FromString(s)
	var szTip [128]uint16
	copy(szTip[:], utf16Tip)
	return szTip
}
