package gui

import (
	"log"
	"path/filepath"

	"github.com/coyim/coyim/i18n"
	"github.com/coyim/coyim/session/data"
	"github.com/coyim/coyim/session/events"
	"github.com/coyim/coyim/xmpp/utils"
	"github.com/coyim/gotk3adapter/gtki"
)

// OK, so from the user interface, for now, we need a few things:
//  - A way of choosing where the file should be put
//  - A way of displaying errors that happened during the transfer
//  - A way for the user to cancel the transfer
//  - A way to notify the user when the transfer is done
//  - A way to update the user interface about progress
//  In general, hopefully these methods are completely independent of transport.
// Once we get to encrypted transfer we might want to highlight that (and say
// something about the file being transmitted in the clear otherwise)

// Actual user interface:
//   First - ask if you want the file
//   Second - choose where to put the file using standard file chooser/saver
//   Third - one status bar per file with percentages etc.
//       This will get a checkbox and a message when done
//       Or it will get an error message when failed
//       There will be a cancel button there, that will cancel the file receipt

func (u *gtkUI) startAllListenersForFileReceiving(ev events.FileTransfer, cv conversationView, file *fileNotification) {
	go ev.Control.WaitForFinish(func() {
		doInUIThread(func() {
			cv.successFileTransfer(file, "receive")
		})
		log.Printf("Receiving file transfer of file %s finished with success", ev.Name)
	})

	go ev.Control.WaitForUpdate(func(upd, total int64) {
		file.progress = float64((upd*100)/total) / 100

		doInUIThread(func() {
			cv.startFileTransfer(file)
		})
		log.Printf("Receiving file transfer of file %s: %d/%d done", ev.Name, upd, total)

		// TODO: behaves weirdly on bytestreams
		if file.canceled {
			doInUIThread(func() {
				cv.cancelFileTransfer(file)
			})
			ev.Control.Cancel()
		} else if cv.isFileTransferNotifCanceled() {
			log.Printf("Receiving file transfer of file canceled")
			ev.Control.Cancel()
		}
	})

	go ev.Control.WaitForError(func(err error) {
		doInUIThread(func() {
			cv.failFileTransfer(file)
		})
		log.Printf("Receiving file transfer of file %s failed with %v", ev.Name, err)
	})
}

func (u *gtkUI) handleFileTransfer(ev events.FileTransfer) {
	dialogID := "FileTransferAskToReceive"
	builder := newBuilder(dialogID)
	dialogOb := builder.getObj(dialogID)
	account := u.findAccountForSession(ev.Session)

	d := dialogOb.(gtki.MessageDialog)
	d.SetDefaultResponse(gtki.RESPONSE_YES)
	d.SetTransientFor(u.window)

	var message, secondary string

	if ev.IsDirectory {
		message = i18n.Localf("%s wants to send you a directory: do you want to receive it?", utils.RemoveResourceFromJid(ev.Peer))
		secondary = i18n.Localf("Directory name: %s", ev.Name)
	} else {
		message = i18n.Localf("%s wants to send you a file: do you want to receive it?", utils.RemoveResourceFromJid(ev.Peer))
		secondary = i18n.Localf("File name: %s", ev.Name)
	}

	if ev.Description != "" {
		secondary = i18n.Localf("%s\nDescription: %s", secondary, ev.Description)
	}
	if ev.DateLastModified != "" {
		secondary = i18n.Localf("%s\nLast modified: %s", secondary, ev.DateLastModified)
	}
	if ev.Size != 0 {
		secondary = i18n.Localf("%s\nSize: %d bytes", secondary, ev.Size)
	}

	d.SetProperty("text", message)
	d.SetProperty("secondary_text", secondary)

	responseType := gtki.ResponseType(d.Run())
	result := responseType == gtki.RESPONSE_YES
	d.Destroy()

	var name string

	if result {
		label := "Choose where to save file"
		action := gtki.FILE_CHOOSER_ACTION_SAVE
		if ev.IsDirectory {
			label = "Choose where to save folder"
			action = gtki.FILE_CHOOSER_ACTION_CREATE_FOLDER
		}

		fdialog, _ := g.gtk.FileChooserDialogNewWith2Buttons(
			i18n.Local(label),
			u.window,
			action,
			i18n.Local("_Cancel"),
			gtki.RESPONSE_CANCEL,
			i18n.Local("_Save"),
			gtki.RESPONSE_OK,
		)

		fdialog.SetCurrentName(ev.Name)

		if gtki.ResponseType(fdialog.Run()) == gtki.RESPONSE_OK {
			name = fdialog.GetFilename()
		}
		fdialog.Destroy()
	}

	if result && name != "" {
		fileName := resizeFileName(ev.Name)
		purpose := "receive"

		cv := u.roster.openConversationView(account, utils.RemoveResourceFromJid(ev.Peer), true, "")

		var currentFile *fileNotification
		if !cv.getFileTransferNotification() {
			currentFile = cv.showFileTransferNotification(fileName, purpose, ev.IsDirectory)
			u.startAllListenersForFileReceiving(ev, cv, currentFile)
			ev.Answer <- &name
		} else {
			currentFile = cv.showFileTransferInfo(fileName, purpose, ev.IsDirectory)
			u.startAllListenersForFileReceiving(ev, cv, currentFile)
			ev.Answer <- &name
		}
	} else {
		ev.Answer <- nil
	}
}

func (u *gtkUI) startAllListenersForFileSending(ctl *data.FileTransferControl, cv conversationView, file *fileNotification, name string) {
	// TODO: this updates slowly
	go ctl.WaitForError(func(err error) {
		doInUIThread(func() {
			cv.failFileTransfer(file)
		})
		log.Printf("Sending file transfer of file %s failed with %v", name, err)
	})

	go ctl.WaitForFinish(func() {
		doInUIThread(func() {
			cv.successFileTransfer(file, "send")
		})
		log.Printf("Sending file transfer of file %s finished with success", name)
	})

	go ctl.WaitForUpdate(func(upd, total int64) {
		file.progress = float64((upd*100)/total) / 100
		doInUIThread(func() {
			cv.startFileTransfer(file)
		})
		log.Printf("Sending file transfer of file %s: %d/%d done", name, upd, total)

		if file.canceled {
			doInUIThread(func() {
				cv.cancelFileTransfer(file)
			})
			ctl.Cancel()
			return
		} else if cv.isFileTransferNotifCanceled() {
			log.Printf("Sending file transfer of file canceled")
			ctl.Cancel()
			return
		}
	})
}

func (account *account) sendDirectoryTo(peer string, u *gtkUI) {
	if dir, ok := chooseDirToSend(u.window); ok {
		ctl := account.session.SendDirTo(peer, dir)

		base := filepath.Base(dir)
		dirName := resizeFileName(base)

		cv := u.roster.openConversationView(account, utils.RemoveResourceFromJid(peer), true, "")

		var currentFile *fileNotification
		if !cv.getFileTransferNotification() {
			currentFile = cv.showFileTransferNotification(dirName, "send", true)
			u.startAllListenersForFileSending(ctl, cv, currentFile, dir)
		} else {
			currentFile = cv.showFileTransferInfo(dirName, "send", true)
			u.startAllListenersForFileSending(ctl, cv, currentFile, dir)
		}
	}
}

func (account *account) sendFileTo(peer string, u *gtkUI) {
	if file, ok := chooseFileToSend(u.window); ok {
		ctl := account.session.SendFileTo(peer, file)

		base := filepath.Base(file)
		fileName := resizeFileName(base)
		purpose := "send"

		cv := u.roster.openConversationView(account, utils.RemoveResourceFromJid(peer), true, "")

		var currentFile *fileNotification
		if !cv.getFileTransferNotification() {
			currentFile = cv.showFileTransferNotification(fileName, purpose, false)
			u.startAllListenersForFileSending(ctl, cv, currentFile, file)
		} else {
			currentFile = cv.showFileTransferInfo(fileName, purpose, false)
			u.startAllListenersForFileSending(ctl, cv, currentFile, file)
		}
	}
}

func chooseItemToSend(w gtki.Window, action gtki.FileChooserAction, title string) (string, bool) {
	dialog, _ := g.gtk.FileChooserDialogNewWith2Buttons(
		i18n.Local(title),
		w,
		action,
		i18n.Local("_Cancel"),
		gtki.RESPONSE_CANCEL,
		i18n.Local("Send"),
		gtki.RESPONSE_OK,
	)
	defer dialog.Destroy()

	if gtki.ResponseType(dialog.Run()) == gtki.RESPONSE_OK {
		return dialog.GetFilename(), true
	}
	return "", false
}

func chooseFileToSend(w gtki.Window) (string, bool) {
	return chooseItemToSend(w, gtki.FILE_CHOOSER_ACTION_OPEN, "Chose file to send")
}

func chooseDirToSend(w gtki.Window) (string, bool) {
	return chooseItemToSend(w, gtki.FILE_CHOOSER_ACTION_SELECT_FOLDER, "Chose directory to send")
}
