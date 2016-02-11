package gui

import gtk "github.com/gotk3/gotk3/gtk/iface"

func (u *gtkUI) increaseFontSize(w gtk.Window) {
	u.displaySettings.increaseFontSize()
}

func (u *gtkUI) decreaseFontSize(w gtk.Window) {
	u.displaySettings.decreaseFontSize()
}

func (u *gtkUI) closeApplication(w gtk.Window) {
	u.quit()
}

func (u *gtkUI) closeWindow(w gtk.Window) {
	w.Hide()
}

func connectShortcut(accel string, w gtk.Window, action func(gtk.Window)) {
	gx, _ := g.gtk.AccelGroupNew()
	key, mod := g.gtk.AcceleratorParse(accel)

	// Do not remove the closure here - there is a limitation
	// in gtk that makes it necessary to have different functions for different accelerator groups
	gx.Connect2(key, mod, gtk.ACCEL_VISIBLE, func() {
		action(w)
	})

	w.AddAccelGroup(gx)
}

func (u *gtkUI) connectShortcutsMainWindow(w gtk.Window) {
	// <Primary> maps to Command on OS X, but Control on other platforms
	connectShortcut("<Primary>q", w, u.closeApplication)
	connectShortcut("<Primary>w", w, u.closeApplication)
	connectShortcut("<Alt>F4", w, u.closeApplication)
}

func (u *gtkUI) connectShortcutsChildWindow(w gtk.Window) {
	// <Primary> maps to Command on OS X, but Control on other platforms
	connectShortcut("<Primary>q", w, u.closeApplication)
	connectShortcut("<Primary>w", w, u.closeWindow)
	connectShortcut("<Primary>F4", w, u.closeWindow)
	connectShortcut("<Alt>F4", w, u.closeApplication)
	connectShortcut("Escape", w, u.closeWindow)
}

func (u *gtkUI) connectShortcutsConversationWindow(c *conversationWindow) {
	// <Primary> maps to Command on OS X, but Control on other platforms
	connectShortcut("<Primary>plus", c.win, u.increaseFontSize)
	connectShortcut("<Primary>minus", c.win, u.decreaseFontSize)
}
