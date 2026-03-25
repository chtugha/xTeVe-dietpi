package src

const (
	// plexChannelLimit is the maximum number of channels exposed to Plex DVR.
	// Default value: 480.
	plexChannelLimit = 480

	// unfilteredChannelLimit is the maximum number of channels in the unfiltered
	// lineup. Default value: 480.
	unfilteredChannelLimit = 480

	// minCompatibilityVersion is the minimum xTeVe database version that this
	// build is compatible with. Default value: "1.4.4".
	minCompatibilityVersion = "1.4.4"

	// defaultPort is the TCP port xTeVe listens on when none is configured.
	// Default value: "34400".
	defaultPort = "34400"

	// defaultBackupKeep is the number of backup archives to retain before
	// pruning the oldest. Default value: 10.
	defaultBackupKeep = 10

	// defaultLogEntriesRAM is the number of log entries held in memory for
	// the web UI log view. Default value: 500.
	defaultLogEntriesRAM = 500

	// defaultBufferSizeKB is the read-buffer size used by the streaming proxy,
	// in kilobytes. Default value: 1024.
	defaultBufferSizeKB = 1024

	// defaultBufferTimeoutMS is the initial delay before the streaming proxy
	// begins forwarding data to clients, in milliseconds. Default value: 500.
	defaultBufferTimeoutMS = 500

	// defaultM3U8BandwidthMBPS is the assumed stream bandwidth used when
	// selecting an adaptive-bitrate HLS variant, in megabits per second.
	// Default value: 10.
	defaultM3U8BandwidthMBPS = 10

	// defaultMappingFirstChannel is the channel number assigned to the first
	// mapped XEPG channel. Default value: 1000.
	defaultMappingFirstChannel = 1000

	// defaultSessionTimeoutMin is the web UI session timeout, in minutes.
	// Default value: 60.
	defaultSessionTimeoutMin = 60

	// ssdpMaxAgeSec is the SSDP advertisement cache-control max-age, in seconds.
	// Default value: 1800.
	ssdpMaxAgeSec = 1800

	// ssdpAliveIntervalSec is the interval between SSDP alive announcements, in seconds.
	// Default value: 300.
	ssdpAliveIntervalSec = 300

	// websocketBufferSize is the read/write buffer size for WebSocket upgrades, in bytes.
	// Default value: 1024.
	websocketBufferSize = 1024

	// logLabelPadWidth is the column width used for label alignment in log output.
	// Default value: 23.
	logLabelPadWidth = 23
)
