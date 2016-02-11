// +build !cli

package main

import (
	"github.com/gotk3/gotk3/gdk"
	"github.com/gotk3/gotk3/glib"
	"github.com/gotk3/gotk3/gtk"
	"github.com/gotk3/gotk3/pango"
	"github.com/twstrike/coyim/gui"
	"github.com/twstrike/coyim/session"
	"github.com/twstrike/coyim/xmpp"
)

func runClient() {
	g := gui.CreateGraphics(
		gtk.RealSince310,
		glib.Real,
		gdk.Real,
		pango.Real,
	)
	gui.NewGTK(coyimVersion, session.Factory, xmpp.DialerFactory, g).Loop()
}
