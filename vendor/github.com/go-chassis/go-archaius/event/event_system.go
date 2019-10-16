/*
 * Copyright 2017 Huawei Technologies Co., Ltd
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

/*
 * Created by on 2017/6/22.
 */

//Package event provides the different Listeners
package event

import (
	"errors"
	"github.com/go-mesh/openlogging"
	"regexp"
)

// Event Constant
const (
	Update        = "UPDATE"
	Delete        = "DELETE"
	Create        = "CREATE"
	InvalidAction = "INVALID-ACTION"
)

// Event generated when any config changes
type Event struct {
	EventSource string
	EventType   string
	Key         string
	Value       interface{}
}

// Listener All Listener should implement this Interface
type Listener interface {
	Event(event *Event)
}

//Dispatcher is the observer
type Dispatcher struct {
	listeners map[string][]Listener
}

// NewDispatcher is a new Dispatcher for listeners
func NewDispatcher() *Dispatcher {
	dis := new(Dispatcher)
	dis.listeners = make(map[string][]Listener)
	return dis
}

// RegisterListener registers listener for particular configuration
func (dis *Dispatcher) RegisterListener(listenerObj Listener, keys ...string) error {
	if listenerObj == nil {
		err := errors.New("nil listener")
		openlogging.GetLogger().Error("nil listener supplied:" + err.Error())
		return errors.New("nil listener")
	}

	for _, key := range keys {
		listenerList, ok := dis.listeners[key]
		if !ok {
			listenerList = make([]Listener, 0)
		}

		// for duplicate registration
		for _, listener := range listenerList {
			if listener == listenerObj {
				return nil
			}
		}

		// append new listener
		listenerList = append(listenerList, listenerObj)

		// assign latest listener list
		dis.listeners[key] = listenerList
	}
	return nil
}

// UnRegisterListener un-register listener for a particular configuration
func (dis *Dispatcher) UnRegisterListener(listenerObj Listener, keys ...string) error {
	if listenerObj == nil {
		return errors.New("nil listener")
	}

	for _, key := range keys {
		listenerList, ok := dis.listeners[key]
		if !ok {
			continue
		}

		newListenerList := make([]Listener, 0)
		// remove listener
		for _, listener := range listenerList {
			if listener == listenerObj {
				continue
			}
			newListenerList = append(newListenerList, listener)
		}

		// assign latest listener list
		dis.listeners[key] = newListenerList
	}
	return nil
}

// DispatchEvent sends the action trigger for a particular event on a configuration
func (dis *Dispatcher) DispatchEvent(event *Event) error {
	if event == nil {
		return errors.New("empty event provided")
	}

	for regKey, listeners := range dis.listeners {
		matched, err := regexp.MatchString(regKey, event.Key)
		if err != nil {
			openlogging.GetLogger().Errorf("regular expresssion for key %s failed: %s", regKey, err)
			continue
		}
		if matched {
			for _, listener := range listeners {
				openlogging.GetLogger().Debugf("event generated for %s", regKey)
				go listener.Event(event)
			}
		}
	}

	return nil
}
