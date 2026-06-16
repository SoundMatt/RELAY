// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package relay

import "errors"

// Common error sentinels. Every RELAY-conformant implementation must define
// these errors and wrap them so errors.Is returns true (see spec §5).
var (
	ErrClosed          = errors.New("relay: closed")
	ErrNotConnected    = errors.New("relay: not connected")
	ErrTimeout         = errors.New("relay: timeout")
	ErrPayloadTooLarge = errors.New("relay: payload too large")
)
