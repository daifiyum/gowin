## gowin

go/win32实现的最简空白窗口，附带托盘、托盘菜单和气泡通知

### 起源

在使用go写windows托盘应用时，目前的托盘包用不惯，半年不更新，使用也经常出问题，于是便自己研究一下，参考已有托盘包实现和c++/win32例子，写了一个最简空白窗口+托盘，目前没有封装成易用的托盘包，这样更灵活，别人看代码也更轻松

### 注意

**windows系统通知（气泡通知）：**

| 特性         | 传统通知 (Shell_NotifyIcon)                                  | 现代通知 (ToastNotification)             |
| ------------ | ------------------------------------------------------------ | ---------------------------------------- |
| **实现方式** | 使用 `Shell_NotifyIcon` Win32 API                            | 使用 `ToastNotification` WinRT API       |
| **APP图标**  | 来自注册窗口类时定义                                         | 通过注册表内定义的 AUMID 获取            |
| **APP名称**  | 从 `.rc` 文件的 `FileDescription` 获取，若无则使用编译后的文件名（如 `app.exe`） | 通过注册表内定义的 AUMID 获取            |

**.rc文件**

里面只写了`FileDescription`的值，为了让气泡通知显示软件名称而不带.exe后缀，可以自己修改，改后需使用windres重新编译为.syso文件，.syso文件必须与`app.go`一起，放在`main.go`处无效

**非阻塞运行**

```
// 标准方法
// 使用协程运行，务必在main函数下执行：runtime.UnlockOSThread()
// 因为在init里面进行了上锁，目的是为了阻塞运行时，绑定os线程，此时使用协程运行，就不需要在主线程里面上锁了，解锁一下更好
go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		app, _ := gowin.New("./go.ico")
		app.Run()
}()

// 错误方法1
app, _ := gowin.New("./go.ico")
go func() {
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		app.Run()
}()

// 错误方法2
app, _ := gowin.New("./go.ico")
go app.Run()

// 错误方法3
go func() {
		app, _ := gowin.New("./go.ico")
		app.Run()
}()
```



### 留言

但愿有屌大的人写出更好用的go托盘包，且代码简洁，实现优雅，然后推荐给我