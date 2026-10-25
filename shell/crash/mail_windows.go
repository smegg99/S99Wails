// shell/crash/mail_windows.go

//go:build windows

package crash

import (
	"fmt"
	"path/filepath"
	"runtime"
	"unsafe"

	"golang.org/x/sys/windows"
)

// Simple MAPI, which Outlook, Thunderbird and other desktop clients register; its structs as in mapi.h.
type mapiRecip struct {
	reserved   uint32
	recipClass uint32
	name       *uint16
	address    *uint16
	eidSize    uint32
	entryID    uintptr
}

type mapiFile struct {
	reserved uint32
	flags    uint32
	position uint32
	pathName *uint16
	fileName *uint16
	fileType uintptr
}

type mapiMessage struct {
	reserved       uint32
	subject        *uint16
	noteText       *uint16
	messageType    *uint16
	dateReceived   *uint16
	conversationID *uint16
	flags          uint32
	originator     uintptr
	recipCount     uint32
	recips         *mapiRecip
	fileCount      uint32
	files          *mapiFile
}

const (
	mapiTo        = 1
	mapiLogonUI   = 0x1
	mapiDialog    = 0x8
	mapiUserAbort = 1
	// mapiNoPosition leaves where the attachment goes to the client.
	mapiNoPosition = 0xFFFFFFFF
)

var sendMailW = windows.NewLazySystemDLL("mapi32.dll").NewProc("MAPISendMailW")

func sendMail(m mail) error {
	if sendMailW.Find() != nil {
		return errNoAttach
	}
	text := func(s string) *uint16 {
		p, _ := windows.UTF16PtrFromString(s)
		return p
	}
	recip := mapiRecip{recipClass: mapiTo, name: text(m.to), address: text("SMTP:" + m.to)}
	files := make([]mapiFile, len(m.files))
	for i, path := range m.files {
		files[i] = mapiFile{position: mapiNoPosition, pathName: text(path), fileName: text(filepath.Base(path))}
	}
	msg := mapiMessage{subject: text(m.subject), noteText: text(m.body), recipCount: 1, recips: &recip, fileCount: uint32(len(files))}
	if len(files) > 0 {
		msg.files = &files[0]
	}
	return within(func() error {
		// The compose dialog is modal to the call, so it runs on its own goroutine and thread.
		runtime.LockOSThread()
		defer runtime.UnlockOSThread()
		code, _, _ := sendMailW.Call(0, 0, uintptr(unsafe.Pointer(&msg)), mapiLogonUI|mapiDialog, 0)
		runtime.KeepAlive(&msg)
		runtime.KeepAlive(files)
		if code != 0 && code != mapiUserAbort {
			return fmt.Errorf("MAPISendMailW: %d", code)
		}
		return nil
	})
}
