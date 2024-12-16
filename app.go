// 这是使用go/win32实现的最简空白窗口，附带托盘、托盘菜单和气泡通知
package gowin

import (
	"errors"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

func init() {
	// windows GUI 编程需要在主线程上
	// 注意：锁定到os线程并没有高枕无忧，如果在协程中运行还是会卡死，不信自己测试，有解决办法告诉我一下
	// 此处只能保证在主线程中正常运行
	runtime.LockOSThread()
}

type App struct {
	Hwnd      syscall.Handle  // 窗口句柄
	Hinstance syscall.Handle  // 应用程序句柄
	HIcon     syscall.Handle  // 应用图标
	ClassName *uint16         // 窗口类名
	WinName   *uint16         // 窗口名称
	Nid       NOTIFYICONDATAW // 托盘实例
	Hmenu     syscall.Handle  // 菜单句柄
}

// 初始化一切
func New(icon string) (*App, error) {
	hIcon, _ := LoadIconFromFile(icon)

	app := &App{HIcon: hIcon}
	app.SetProcessDPIAware()
	app.registerWindowClass()
	app.createWindow()
	app.initTrayIcon()
	app.AddMenu()

	return app, nil
}

// Run
func (t *App) Run() error {
	return t.messageLoop()
}

// 消息循环
func (t *App) messageLoop() error {
	var msg MSG
	for {
		ret, _, _ := GetMessage.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if ret <= 0 {
			return errors.New("GetMessage failed")
		}
		TranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		DispatchMessage.Call(uintptr(unsafe.Pointer(&msg)))
	}
}

// 设置进程 DPI 感知，让界面看起来清晰
func (t *App) SetProcessDPIAware() error {
	status, _, err := SetProcessDPIAware.Call()
	if status == 0 {
		return fmt.Errorf("SetProcessDPIAware failed: %v", err)
	}
	return nil
}

// 注册窗口类
func (t *App) registerWindowClass() error {
	hinstance, _, _ := GetModuleHandle.Call(0)

	cursor, _, _ := LoadCursor.Call(0, uintptr(IDC_ARROW))

	className, _ := syscall.UTF16PtrFromString("DemoWindowClass")
	t.ClassName = className
	windowName, _ := syscall.UTF16PtrFromString("go/win32 demo")
	t.WinName = windowName

	var wcex WNDCLASSEX
	wcex.Size = uint32(unsafe.Sizeof(wcex))
	wcex.Style = 0
	wcex.WndProc = syscall.NewCallback(t.windowProc)
	wcex.ClsExtra = 0
	wcex.WndExtra = 0
	wcex.Instance = syscall.Handle(hinstance)
	wcex.Icon = syscall.Handle(t.HIcon)
	wcex.Cursor = syscall.Handle(cursor)
	wcex.Background = syscall.Handle(COLOR_WINDOW + 1)
	wcex.MenuName = nil
	wcex.ClassName = className
	wcex.IconSm = 0

	ret, _, err := RegisterClassEx.Call(uintptr(unsafe.Pointer(&wcex)))
	if ret == 0 {
		return fmt.Errorf("RegisterClassEx failed: %w", err)
	}

	t.Hinstance = syscall.Handle(hinstance)
	return nil
}

// 注销窗口
func (t *App) unregister() error {
	res, _, err := UnregisterClass.Call(
		uintptr(unsafe.Pointer(t.ClassName)),
		uintptr(t.Hinstance),
	)
	if res == 0 {
		return err
	}
	return nil
}

// 创建窗口
func (t *App) createWindow() error {
	hwnd, _, err := CreateWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(t.ClassName)),
		uintptr(unsafe.Pointer(t.WinName)),
		uintptr(WS_OVERLAPPEDWINDOW),
		uintptr(CW_USEDEFAULT),
		0,
		uintptr(CW_USEDEFAULT),
		0,
		0,
		0,
		uintptr(t.Hinstance),
		0,
	)
	if hwnd == 0 {
		return fmt.Errorf("CreateWindowEx failed: %w", err)
	}

	t.Hwnd = syscall.Handle(hwnd)

	ShowWindow.Call(
		uintptr(hwnd),
		uintptr(1), // 窗口状态：0隐藏，1显示，微软文档搜ShowWindow查看更多
	)

	UpdateWindow.Call(
		uintptr(hwnd),
	)

	return nil
}

// 初始化托盘
func (t *App) initTrayIcon() error {
	var nid NOTIFYICONDATAW
	nid.CbSize = uint32(unsafe.Sizeof(nid))
	nid.HWnd = t.Hwnd
	nid.UID = 1
	nid.UFlags = NIF_ICON | NIF_MESSAGE | NIF_TIP
	nid.HIcon = t.HIcon
	nid.UCallbackMessage = WM_TRAY_NOTIFYICON
	nid.SzTip = TipFromStr("go win32 app")

	t.Nid = nid

	ret, _, err := ShellNotifyIcon.Call(NIM_ADD, uintptr(unsafe.Pointer(&t.Nid)))
	if ret == 0 {
		fmt.Println(ret, err)
		return fmt.Errorf("failed to add tray icon: %w", err)
	}

	return nil
}

// 更新托盘图标
func (t *App) SetIcon(iconPath string) error {
	hIcon, err := LoadIconFromFile(iconPath)
	if err != nil {
		return fmt.Errorf("failed to load icon: %w", err)
	}

	t.Nid.HIcon = hIcon
	ret, _, err := ShellNotifyIcon.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&t.Nid)))
	if ret == 0 {
		return fmt.Errorf("failed to update tray icon: %w", err)
	}

	return nil
}

// 弹出一条系统通知
func (t *App) ShowTrayNotification(title, msg string) error {
	t.Nid.CbSize = uint32(unsafe.Sizeof(t.Nid))
	t.Nid.HWnd = t.Hwnd
	t.Nid.UFlags = NIF_INFO

	SetUTF16String(&t.Nid.SzInfoTitle, title)
	SetUTF16String(&t.Nid.SzInfo, msg)

	ret, _, err := ShellNotifyIcon.Call(NIM_MODIFY, uintptr(unsafe.Pointer(&t.Nid)))
	if ret == 0 {
		return fmt.Errorf("Shell_NotifyIcon failed: %w", err)
	}

	// 重置默认标识，不然气泡通知会不断弹出，微软BUG？
	t.Nid.UFlags = NIF_ICON | NIF_TIP | NIF_MESSAGE

	return nil
}

// 添加托盘菜单项
func (t *App) AddMenu() {
	// 创建菜单
	hmenu, _, _ := CreatePopupMenu.Call(0, 0, 0, 0)

	// 创建菜单项
	menuItems := []map[string]any{
		{"id": 1001, "label": "菜单项1", "type": MF_STRING},
		{"id": 1002, "label": "菜单项2", "type": MF_STRING},
		{"id": 1003, "label": "复选菜单项", "type": MF_CHECKED},
		{"id": 1004, "type": MF_SEPARATOR},
		{"id": 1005, "label": "退出", "type": MF_STRING},
	}

	for i := range menuItems {
		l := menuItems[i]["label"]
		f := menuItems[i]["type"]
		id := menuItems[i]["id"].(int)
		if l != nil {
			label, _ := syscall.UTF16PtrFromString(l.(string))
			AppendMenu.Call(hmenu, uintptr(f.(int)), uintptr(id), uintptr(unsafe.Pointer(label)))
		} else {
			AppendMenu.Call(hmenu, uintptr(f.(int)), uintptr(id))
		}
	}

	t.Hmenu = syscall.Handle(hmenu)
}

// 弹出托盘菜单
func (t *App) ShowMenu() {
	pt := POINT{}
	// 获取点击时的鼠标坐标
	GetCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	x, y := int(pt.X), int(pt.Y)

	// 设置位前台窗口，这样鼠标从菜单上失焦就会自动关闭
	SetForegroundWindow.Call(uintptr(t.Hwnd))

	// 弹出菜单
	TrackPopupMenu.Call(
		uintptr(t.Hmenu),
		uintptr(TPM_LEFTBUTTON), // 菜单项只可左键点击
		uintptr(x),
		uintptr(y),
		0,
		uintptr(t.Hwnd),
		0,
	)
}

// 托盘菜单项回调
func (t *App) MenuCallback(wp uint32) {
	switch wp {
	case 1001:
		fmt.Println("菜单项1")
	case 1002:
		fmt.Println("菜单项2")
	case 1003:
		if !CheckItem(t.Hmenu, 1003) {
			SetCheckItem(t.Hmenu, 1003, MF_CHECKED)
		} else {
			SetCheckItem(t.Hmenu, 1003, MF_UNCHECKED)
		}
		fmt.Println("菜单项3")
	case 1005:
		DestroyWindow.Call(uintptr(t.Hwnd))
	}
}

// 消息处理函数
func (t *App) windowProc(hwnd syscall.Handle, msg uint32, wparam, lparam uintptr) uintptr {
	switch msg {
	case WM_CREATE: // 窗口创建消息
		fmt.Println("窗口即将创建")
		return 0
	case WM_TRAY_NOTIFYICON: // 自定义托盘图标消息
		// 托盘左右键
		switch lparam {
		case WM_LBUTTONUP:
			fmt.Println("左键点击")
		case WM_RBUTTONUP:
			fmt.Println("右键点击")
			t.ShowMenu() // 右键弹出托盘菜单
		}
		return 0
	case WM_COMMAND: // 菜单项等消息
		// 菜单项回调
		if HIWORD(uint64(wparam)) == 0 {
			t.MenuCallback(LOWORD(uint64(wparam)))
		}
		return 0
	case WM_CLOSE: // 关闭窗口消息，即通知窗口关闭
		DestroyWindow.Call(uintptr(t.Hwnd))
		return 0
	case WM_DESTROY: // 窗口关闭时的消息
		t.unregister() // 退出前注销窗口
		PostQuitMessage.Call(0)
		return 0
	default: // 其他消息默认处理
		ret, _, _ := DefWindowProc.Call(uintptr(hwnd), uintptr(msg), wparam, lparam)
		return ret
	}

}
