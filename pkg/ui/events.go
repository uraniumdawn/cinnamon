// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

// EventType represents the type of event being published.
type EventType string

// Event represents an event with a type and payload.
type Event struct {
	Type    EventType
	Payload Payload
}

// Payload contains the data and force flag for an event.
type Payload struct {
	Data  any
	Force bool
}

// Publish sends an event to the specified channel.
func Publish(ch chan<- Event, eventType EventType, p Payload) {
	ch <- Event{Type: eventType, Payload: p}
}
