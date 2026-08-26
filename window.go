package main

import (
	"golang.org/x/sys/windows"
	"syscall"
	"unsafe"
	"strings"
	"fmt"
	"time"
	"log"
	"github.com/shirou/gopsutil/v3/process"
)
type WindowInfo struct {
	HWND      uintptr
	PID       uint32
	Title     string
	ClassName string
	Process   string
	Visible   bool
	Style     uintptr
	ExStyle   uintptr
}

type ChildWindow struct {
	HWND      uintptr
	ClassName string
	Title     string
}

var (
	user32 = windows.NewLazySystemDLL("user32.dll")
	kernel32 = windows.NewLazySystemDLL("kernel32.dll")
	enumeratedWindows []WindowInfo

	procGetForegroundWindow      = user32.NewProc("GetForegroundWindow")
	procGetWindowTextW           = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId = user32.NewProc("GetWindowThreadProcessId")

	procEnumWindows              = user32.NewProc("EnumWindows")
	procIsWindowVisible          = user32.NewProc("IsWindowVisible")
	procGetClassNameW            = user32.NewProc("GetClassNameW") // <-- Agrega esta línea

	procOpenProcess = kernel32.NewProc("OpenProcess")
	procCloseHandle = kernel32.NewProc("CloseHandle")

	procEnumChildWindows = user32.NewProc("EnumChildWindows")
	procGetWindowLongPtrW = user32.NewProc("GetWindowLongPtrW")
)

func getAllWindows() []WindowInfo {
	enumeratedWindows = nil

	callback := syscall.NewCallback(enumWindowsCallback)

	procEnumWindows.Call(
		callback,
		0,
	)

	return enumeratedWindows
}

func enumWindowsCallback(hwnd uintptr, lparam uintptr) uintptr {

	pid := getWindowProcessID(hwnd)
	title := getWindowTitle(hwnd)
	process := getProcessName(pid)

	window := WindowInfo{
		HWND:      hwnd,
		PID:       pid,
		Title:     title,
		ClassName: getWindowClass(hwnd),
		Process:   process,
		Visible:   isWindowVisible(hwnd),
	}

	enumeratedWindows = append(enumeratedWindows, window)

	return 1
}

func isWindowVisible(hwnd uintptr) bool {
	ret, _, _ := procIsWindowVisible.Call(hwnd)
	return ret != 0
}

// resumidamente
func getForegroundWindow() uintptr {
	hwnd, _, _ := procGetForegroundWindow.Call()
	return hwnd
}

func getWindowTitle(hwnd uintptr) string {
	buffer := make([]uint16, 255)

	procGetWindowTextW.Call(
		hwnd,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)

	return syscall.UTF16ToString(buffer)
}

func getWindowProcessID(hwnd uintptr) uint32 {
	var pid uint32

	procGetWindowThreadProcessId.Call(
		hwnd,
		uintptr(unsafe.Pointer(&pid)),
	)

	return pid
}
func isWindowFromPID(hwnd uintptr, pid uint32) bool {
	windowPID := getWindowProcessID(hwnd)
	return windowPID == pid
}

func getProcessName(pid uint32) string {
	p, err := process.NewProcess(int32(pid))
	if err != nil {
		return "desconocido"
	}

	name, err := p.Name()
	if err != nil {
		return "desconocido"
	}

	return name
}

func getWorkraveWindows() []WindowInfo {
	var result []WindowInfo

	for _, w := range getAllWindows() {
		if w.PID == 0 {
			continue
		}

		if strings.EqualFold(w.Process, "workrave.exe") {
			result = append(result, w)
		}
	}

	return result
}
func windowSignature(w WindowInfo) string {
	return fmt.Sprintf(
		"%d|%d|%t|%s",
		w.HWND,
		w.PID,
		w.Visible,
		w.Title,
	)
}
func monitorWindows() {

	known := make(map[string]bool)

	for {

		windows := getAllWindows()

		current := make(map[string]bool)

		for _, w := range windows {

			key := windowSignature(w)

			current[key] = true

			if !known[key] {

				log.Printf(
					"[NUEVA] PID:%d | Proceso:%s | Visible:%t | HWND:%d | Título:%q",
					w.PID,
					w.Process,
					w.Visible,
					w.HWND,
					w.Title,
				)
			}
		}

		known = current

		time.Sleep(500 * time.Millisecond)
	}
}

func isMicroBreakVisible() bool {

	for _, w := range getAllWindows() {

		if !strings.EqualFold(w.Process, "workrave.exe") {
			continue
		}

		if !w.Visible {
			continue
		}

		if strings.EqualFold(w.Title, "Micro-pausa") {
			return true
		}
	}

	return false
}

func monitorMicroBreak() {

	lastState := false

	for {

		currentState := isMicroBreakVisible()

		if currentState != lastState {

			if currentState {

				window := getActiveWorkraveWindow()

				if window != nil {
					log.Printf("DESCANSO INICIADO | Tipo: %s", window.Title)
				} else {
					log.Println("DESCANSO INICIADO")
				}

			} else {
				log.Println("DESCANSO TERMINADO")
			}

			lastState = currentState
		}

		time.Sleep(250 * time.Millisecond)
	}
}

func isInternalWorkraveWindow(title string) bool {

	switch title {

	case "":
		return true

	case "Harpoon64NotificationWindow":
		return true

	case "GDI+ Window (workrave.exe)":
		return true

	case "MSCTFIME UI":
		return true

	case "Default IME":
		return true
	}

	return false
}

func getActiveWorkraveWindow() *WindowInfo {

	for _, w := range getAllWindows() {

		if !strings.EqualFold(w.Process, "workrave.exe") {
			continue
		}

		if !w.Visible {
			continue
		}

		if isInternalWorkraveWindow(w.Title) {
			continue
		}

		return &w
	}

	return nil
}

func getWindowClass(hwnd uintptr) string {

	buffer := make([]uint16, 256)

	ret, _, _ := procGetClassNameW.Call(
		hwnd,
		uintptr(unsafe.Pointer(&buffer[0])),
		uintptr(len(buffer)),
	)

	if ret == 0 {
		return ""
	}

	return windows.UTF16ToString(buffer)
}

/*
const (
	GWL_STYLE   = -16
	GWL_EXSTYLE = -20
)

func getWindowStyle(hwnd uintptr) uintptr {
	style, _, _ := procGetWindowLongPtrW.Call(
		hwnd,
		uintptr(int(GWL_STYLE)),
	)
	return style
}

func getWindowExStyle(hwnd uintptr) uintptr {
	style, _, _ := procGetWindowLongPtrW.Call(
		hwnd,
		uintptr(int(GWL_EXSTYLE)),
	)
	return style
}
*/