// Package autofill provides the Chrome DevTools Protocol
// commands, types, and events for the Autofill domain.
//
// Defines commands and events for Autofill.
//
// Generated by the cdproto-gen command.
package autofill

// Code generated by cdproto-gen. DO NOT EDIT.

import (
	"context"

	"github.com/chromedp/cdproto/cdp"
)

// TriggerParams trigger autofill on a form identified by the fieldId. If the
// field and related form cannot be autofilled, returns an error.
type TriggerParams struct {
	FieldID cdp.BackendNodeID `json:"fieldId"`                    // Identifies a field that serves as an anchor for autofill.
	FrameID cdp.FrameID       `json:"frameId,omitempty,omitzero"` // Identifies the frame that field belongs to.
	Card    *CreditCard       `json:"card"`                       // Credit card information to fill out the form. Credit card data is not saved.
}

// Trigger trigger autofill on a form identified by the fieldId. If the field
// and related form cannot be autofilled, returns an error.
//
// See: https://chromedevtools.github.io/devtools-protocol/tot/Autofill#method-trigger
//
// parameters:
//
//	fieldID - Identifies a field that serves as an anchor for autofill.
//	card - Credit card information to fill out the form. Credit card data is not saved.
func Trigger(fieldID cdp.BackendNodeID, card *CreditCard) *TriggerParams {
	return &TriggerParams{
		FieldID: fieldID,
		Card:    card,
	}
}

// WithFrameID identifies the frame that field belongs to.
func (p TriggerParams) WithFrameID(frameID cdp.FrameID) *TriggerParams {
	p.FrameID = frameID
	return &p
}

// Do executes Autofill.trigger against the provided context.
func (p *TriggerParams) Do(ctx context.Context) (err error) {
	return cdp.Execute(ctx, CommandTrigger, p, nil)
}

// SetAddressesParams set addresses so that developers can verify their forms
// implementation.
type SetAddressesParams struct {
	Addresses []*Address `json:"addresses"`
}

// SetAddresses set addresses so that developers can verify their forms
// implementation.
//
// See: https://chromedevtools.github.io/devtools-protocol/tot/Autofill#method-setAddresses
//
// parameters:
//
//	addresses
func SetAddresses(addresses []*Address) *SetAddressesParams {
	return &SetAddressesParams{
		Addresses: addresses,
	}
}

// Do executes Autofill.setAddresses against the provided context.
func (p *SetAddressesParams) Do(ctx context.Context) (err error) {
	return cdp.Execute(ctx, CommandSetAddresses, p, nil)
}

// DisableParams disables autofill domain notifications.
type DisableParams struct{}

// Disable disables autofill domain notifications.
//
// See: https://chromedevtools.github.io/devtools-protocol/tot/Autofill#method-disable
func Disable() *DisableParams {
	return &DisableParams{}
}

// Do executes Autofill.disable against the provided context.
func (p *DisableParams) Do(ctx context.Context) (err error) {
	return cdp.Execute(ctx, CommandDisable, nil, nil)
}

// EnableParams enables autofill domain notifications.
type EnableParams struct{}

// Enable enables autofill domain notifications.
//
// See: https://chromedevtools.github.io/devtools-protocol/tot/Autofill#method-enable
func Enable() *EnableParams {
	return &EnableParams{}
}

// Do executes Autofill.enable against the provided context.
func (p *EnableParams) Do(ctx context.Context) (err error) {
	return cdp.Execute(ctx, CommandEnable, nil, nil)
}

// Command names.
const (
	CommandTrigger      = "Autofill.trigger"
	CommandSetAddresses = "Autofill.setAddresses"
	CommandDisable      = "Autofill.disable"
	CommandEnable       = "Autofill.enable"
)
