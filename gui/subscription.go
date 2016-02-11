package gui

import (
	"fmt"
	"log"

	glib "github.com/gotk3/gotk3/glib/iface"
	gtk "github.com/gotk3/gotk3/gtk/iface"
	"github.com/twstrike/coyim/i18n"
)

func authorizePresenceSubscriptionDialog(parent gtk.Window, from string) gtk.MessageDialog {
	builder := builderForDefinition("AuthorizeSubscription")

	obj, _ := builder.GetObject("dialog")
	confirmDialog := obj.(gtk.MessageDialog)

	text := fmt.Sprintf(i18n.Local("%s wants to talk to you. Is that ok?"), from)
	confirmDialog.SetProperty("text", text)

	confirmDialog.SetTransientFor(parent)
	return confirmDialog
}

func presenceSubscriptionDialog(accounts []*account, sendSubscription func(accountID, peer string) error) gtk.Dialog {
	builder := builderForDefinition("AddContact")

	//TODO: move model to XML builder
	model, _ := g.gtk.ListStoreNew(
		glib.TYPE_STRING, // account name
		glib.TYPE_STRING, // account_id
	)

	for _, acc := range accounts {
		model.Set2(model.Append(), []int{0, 1}, []interface{}{acc.session.GetConfig().Account, acc.session.GetConfig().ID()})
	}

	accountsObj, _ := builder.GetObject("accounts")
	accountInput := accountsObj.(gtk.ComboBox)
	accountInput.SetModel(model.(gtk.TreeModel))

	accountObj, _ := builder.GetObject("address")
	contactInput := accountObj.(gtk.Entry)

	if len(accounts) > 0 {
		accountInput.SetActive(0)
	}

	renderer, _ := g.gtk.CellRendererTextNew()
	accountInput.(gtk.CellLayout).PackStart(renderer, true)
	accountInput.(gtk.CellLayout).AddAttribute(renderer, "text", 0)

	dialogObj, _ := builder.GetObject("AddContact")
	dialog := dialogObj.(gtk.Dialog)

	obj, _ := builder.GetObject("notification-area")
	notificationArea := obj.(gtk.Box)

	failures := 0
	var notification gtk.InfoBar

	builder.ConnectSignals(map[string]interface{}{
		"on_save_signal": func() {
			contact, _ := contactInput.GetText()
			isJid, errmsg := verifyXmppAddress(contact)

			if !isJid && failures > 0 {
				notificationArea.Remove(notification)
				notification = buildBadUsernameNotification(errmsg)
				notificationArea.Add(notification)
				notification.ShowAll()
				failures++
				log.Printf(errmsg)
				return
			}

			if !isJid {
				notification = buildBadUsernameNotification(errmsg)
				notificationArea.Add(notification)
				notification.ShowAll()
				failures++
				log.Printf(errmsg)
				return
			}

			iter, err := accountInput.GetActiveIter()
			if err != nil {
				log.Printf("Error encountered when getting account: %v", err)
				return
			}
			val, err := model.(gtk.TreeModel).GetValue(iter, 1)
			if err != nil {
				log.Printf("Error encountered when getting account: %v", err)
				return
			}
			accountID, err := val.GetString()
			if err != nil {
				log.Printf("Error encountered when getting account: %v", err)
				return
			}

			err = sendSubscription(accountID, contact)
			if err != nil {
				log.Printf("Error encountered when sending subscription: %v", err)
				return
			}

			dialog.Destroy()
		},
	})

	return dialog
}
