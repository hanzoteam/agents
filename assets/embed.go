// Copyright (c) 2015-present Mattermost, Inc. All Rights Reserved.
// See LICENSE.txt for license information.

package assets

import _ "embed"

//go:embed bot_icon.png
var DefaultAgentProfilePicture []byte

// Role-matched profile icons for the built-in agents, one emoji per role
// (e.g. dev is the computer-programmer emoji). Source: Noto Color Emoji
// (SIL Open Font License 1.1), 128x128 PNG. See NOTICE.txt for attribution.

//go:embed agents/vi.png
var iconVi []byte

//go:embed agents/dev.png
var iconDev []byte

//go:embed agents/des.png
var iconDes []byte

//go:embed agents/opera.png
var iconOpera []byte

//go:embed agents/db.png
var iconDB []byte

//go:embed agents/sec.png
var iconSec []byte

//go:embed agents/core.png
var iconCore []byte

//go:embed agents/algo.png
var iconAlgo []byte

//go:embed agents/mark.png
var iconMark []byte

//go:embed agents/su.png
var iconSu []byte

//go:embed agents/fin.png
var iconFin []byte

//go:embed agents/cal.png
var iconCal []byte

//go:embed agents/art.png
var iconArt []byte

//go:embed agents/mu.png
var iconMu []byte

//go:embed agents/data.png
var iconData []byte

//go:embed agents/chat.png
var iconChat []byte

// agentProfilePictures maps each built-in agent name to the icon matching
// its role. Agents not listed here fall back to DefaultAgentProfilePicture.
var agentProfilePictures = map[string][]byte{
	"vi":    iconVi,    // Visionary Leader — compass
	"dev":   iconDev,   // Software Engineer — computer programmer
	"des":   iconDes,   // Designer — triangular ruler
	"opera": iconOpera, // Operations Engineer — gear
	"db":    iconDB,    // Database Expert — file cabinet
	"sec":   iconSec,   // Security Expert — shield
	"core":  iconCore,  // Core Engineer — brain
	"algo":  iconAlgo,  // Algorithm Expert — puzzle piece
	"mark":  iconMark,  // Marketing Director — megaphone
	"su":    iconSu,    // Support Engineer — headphone
	"fin":   iconFin,   // Financial Expert — money bag
	"cal":   iconCal,   // Calculator — abacus
	"art":   iconArt,   // Artist — artist palette
	"mu":    iconMu,    // Musician — musical note
	"data":  iconData,  // Data Scientist — bar chart
	"chat":  iconChat,  // Conversation Expert — speech balloon
}

// AgentProfilePicture returns the profile image for a built-in agent by
// name, falling back to DefaultAgentProfilePicture when name has no
// role-matched icon.
func AgentProfilePicture(name string) []byte {
	if icon, ok := agentProfilePictures[name]; ok {
		return icon
	}
	return DefaultAgentProfilePicture
}
