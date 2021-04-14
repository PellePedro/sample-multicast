package network

// Differentiated Services Field Codepoints (DSCP), Updated: 2013-06-25
const (
	DiffServCS0        = 0x0  // CS0
	DiffServCS1        = 0x20 // CS1
	DiffServCS2        = 0x40 // CS2
	DiffServCS3        = 0x60 // CS3
	DiffServCS4        = 0x80 // CS4
	DiffServCS5        = 0xa0 // CS5
	DiffServCS6        = 0xc0 // CS6
	DiffServCS7        = 0xe0 // CS7
	DiffServAF11       = 0x28 // AF11
	DiffServAF12       = 0x30 // AF12
	DiffServAF13       = 0x38 // AF13
	DiffServAF21       = 0x48 // AF21
	DiffServAF22       = 0x50 // AF22
	DiffServAF23       = 0x58 // AF23
	DiffServAF31       = 0x68 // AF31
	DiffServAF32       = 0x70 // AF32
	DiffServAF33       = 0x78 // AF33
	DiffServAF41       = 0x88 // AF41
	DiffServAF42       = 0x90 // AF42
	DiffServAF43       = 0x98 // AF43
	DiffServEFPHB      = 0xb8 // EF PHB
	DiffServVOICEADMIT = 0xb0 // VOICE-ADMIT
)

// IPv4 TOS Byte and IPv6 Traffic Class Octet, Updated: 2001-09-06
const (
	NotECNTransport       = 0x0 // Not-ECT (Not ECN-Capable Transport)
	ECNTransport1         = 0x1 // ECT(1) (ECN-Capable Transport(1))
	ECNTransport0         = 0x2 // ECT(0) (ECN-Capable Transport(0))
	CongestionExperienced = 0x3 // CE (Congestion Experienced)
)
