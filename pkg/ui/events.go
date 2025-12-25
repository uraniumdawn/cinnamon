// Copyright (c) Sergey Petrovsky
// This source code is licensed under the MIT license found in the
// LICENSE file in the root directory of this source tree.

package ui

type EventType string

type Event struct {
	Type    EventType
	Payload Payload
}

type Payload struct {
	Data  any
	Force bool
}

func Publish(ch chan<- Event, eventType EventType, p Payload) {
	ch <- Event{Type: eventType, Payload: p}
}
