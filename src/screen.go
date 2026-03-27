package src

import (
	"fmt"
	"log"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var screenLogMutex sync.RWMutex

func showInfo(str string) {

	if System.Flag.Info {
		return
	}

	var msg = strings.SplitN(str, ":", 2)
	var length = len(msg[0])

	if len(msg) == 2 {

		space := strings.Repeat(" ", logLabelPadWidth-length)
		msg[0] = msg[0] + ":" + space

		var logMsg = fmt.Sprintf("[%s] %s%s", System.Name, msg[0], msg[1])

		printLogOnScreen(logMsg, "info")

		screenLogMutex.Lock()
		logMsg = strings.Replace(logMsg, " ", "&nbsp;", -1)
		WebScreenLog.Log = append(WebScreenLog.Log, time.Now().Format("2006-01-02 15:04:05")+" "+logMsg)
		logCleanUp()
		screenLogMutex.Unlock()

	}

	return
}

func showDebug(str string, level int) {

	if System.Flag.Debug < level {
		return
	}

	var msg = strings.SplitN(str, ":", 2)
	var length = len(msg[0])

	if len(msg) == 2 {

		space := strings.Repeat(" ", logLabelPadWidth-length)
		msg[0] = msg[0] + ":" + space

		var logMsg = fmt.Sprintf("[DEBUG] %s%s", msg[0], msg[1])

		printLogOnScreen(logMsg, "debug")

		screenLogMutex.Lock()
		logMsg = strings.Replace(logMsg, " ", "&nbsp;", -1)
		WebScreenLog.Log = append(WebScreenLog.Log, time.Now().Format("2006-01-02 15:04:05")+" "+logMsg)
		logCleanUp()
		screenLogMutex.Unlock()

	}

	return
}

func showHighlight(str string) {

	var msg = strings.SplitN(str, ":", 2)
	var length = len(msg[0])

	var notification Notification
	notification.Type = "info"

	if len(msg) == 2 {

		space := strings.Repeat(" ", logLabelPadWidth-length)
		msg[0] = msg[0] + ":" + space

		var logMsg = fmt.Sprintf("[%s] %s%s", System.Name, msg[0], msg[1])

		printLogOnScreen(logMsg, "highlight")

	}

	notification.Message = msg[1]

	addNotification(notification)

	return
}

func showWarning(errCode int) {

	var errMsg = getErrMsg(errCode)
	var logMsg = fmt.Sprintf("[%s] [WARNING] %s", System.Name, errMsg)

	printLogOnScreen(logMsg, "warning")

	screenLogMutex.Lock()
	WebScreenLog.Log = append(WebScreenLog.Log, time.Now().Format("2006-01-02 15:04:05")+" "+logMsg)
	WebScreenLog.Warnings++
	screenLogMutex.Unlock()

	return
}

// ShowError : Zeigt die Fehlermeldungen in der Konsole
func ShowError(err error, errCode int) {

	var errMsg = getErrMsg(errCode)
	var logMsg = fmt.Sprintf("[%s] [ERROR] %s (%s) - EC: %d", System.Name, err, errMsg, errCode)

	printLogOnScreen(logMsg, "error")

	screenLogMutex.Lock()
	WebScreenLog.Log = append(WebScreenLog.Log, time.Now().Format("2006-01-02 15:04:05")+" "+logMsg)
	WebScreenLog.Errors++
	screenLogMutex.Unlock()

	return
}

func printLogOnScreen(logMsg string, logType string) {

	var color string

	switch logType {

	case "info":
		color = "\033[0m"

	case "debug":
		color = "\033[35m"

	case "highlight":
		color = "\033[32m"

	case "warning":
		color = "\033[33m"

	case "error":
		color = "\033[31m"

	}

	switch runtime.GOOS {

	case "windows":
		log.Println(logMsg)

	default:
		fmt.Print(color)
		log.Println(logMsg)
		fmt.Print("\033[0m")

	}

}

func logCleanUp() {

	var logEntriesRAM = Settings.LogEntriesRAM
	var logs = WebScreenLog.Log

	if len(logs) > logEntriesRAM {
		removed := logs[:len(logs)-logEntriesRAM]
		for _, entry := range removed {
			if strings.Contains(entry, "WARNING") {
				WebScreenLog.Warnings--
			}
			if strings.Contains(entry, "ERROR") {
				WebScreenLog.Errors--
			}
		}
		logs = logs[len(logs)-logEntriesRAM:]
	}

	WebScreenLog.Log = logs

	return
}

// Fehlercodes
func getErrMsg(errCode int) (errMsg string) {

	switch errCode {

	case 0:
		return

	// Errors
	case 1001:
		errMsg = "Web server could not be started."
	case 1002:
		errMsg = "No local IP address found."
	case 1003:
		errMsg = "Invalid xml"
	case 1004:
		errMsg = "File not found"
	case 1005:
		errMsg = "Invalid M3U file, an extended M3U file is required."
	case 1006:
		errMsg = "No playlist!"
	case 1007:
		errMsg = "XEPG requires an XMLTV file."
	case 1010:
		errMsg = "Invalid file compression"
	case 1011:
		errMsg = fmt.Sprintf("Data is corrupt or unavailable, %s now uses an older version of this file", System.Name)
	case 1012:
		errMsg = "Invalid formatting of the time"
	case 1013:
		errMsg = fmt.Sprintf("Invalid settings file (settings.json), file must be at least version %s", System.Compatibility)
	case 1014:
		errMsg = "Invalid filter rule"

	case 1020:
		errMsg = "Data could not be saved, invalid keyword"

	case 1030:
		errMsg = fmt.Sprintf("Invalid settings file (%s)", System.File.Settings)
	case 1031:
		errMsg = "Database error. The database version of your settings is not compatible with this version."

	case 1050:
		errMsg = "Invalid duration specification in the M3U8 playlist."

	case 1060:
		errMsg = "Invalid characters found in the tvg parameters, streams with invalid parameters were skipped."

	case 1070:
		errMsg = "Folder could not be created."
	case 1071:
		errMsg = "File could not be created"
	case 1072:
		errMsg = "File not found"

	case 1090:
		errMsg = "Automatic backup failed"

	case 1100:
		errMsg = "WebUI build error"
	case 1101:
		errMsg = "WebUI request error"
	case 1102:
		errMsg = "WebUI response error"

	case 1200:
		errMsg = "Could not create file"

	case 1201:
		errMsg = "Plex stream error"
	case 1202:
		errMsg = "Streaming URL could not be found in any playlist"
	case 1203:
		errMsg = "Streaming URL could not be found in any playlist"
	case 1204:
		errMsg = "Streaming was stopped by third party transcoder (FFmpeg / VLC)"

	case 2000:
		errMsg = fmt.Sprintf("Plex can not handle more than %d streams. If you do not use Plex, you can ignore this warning.", System.PlexChannelLimit)
	case 2001:
		errMsg = fmt.Sprintf("%s has loaded more than %d streams. Use the filter to reduce the number of streams.", System.Name, System.UnfilteredChannelLimit)
	case 2002:
		errMsg = "PMS can not play m3u8 streams"
	case 2003:
		errMsg = "PMS can not play streams over RTSP."
	case 2004:
		errMsg = "Buffer is disabled for this stream."
	case 2005:
		errMsg = "There are no channels mapped, use the mapping menu to assign EPG data to the channels."
	case 2010:
		errMsg = "No valid streaming URL"
	case 2020:
		errMsg = "FFmpeg binary was not found. Check the FFmpeg binary path in the xTeVe settings."
	case 2021:
		errMsg = "VLC binary was not found. Check the VLC path binary in the xTeVe settings."

	case 2099:
		errMsg = "Updates have been disabled by the developer"

	case 2105:
		errMsg = "The number of tuners has changed, you have to delete " + System.Name + " in Plex / Emby HDHR and set it up again."
	case 2106:
		errMsg = "This function is only available with XEPG as EPG source"

	case 2110:
		errMsg = "Don't run this as Root!"

	case 2300:
		errMsg = "No channel logo found in the XMLTV or M3U file."
	case 2301:
		errMsg = "XMLTV file no longer available, channel has been deactivated."
	case 2302:
		errMsg = "Channel ID in the XMLTV file has changed. Channel has been deactivated."

	case 3000:
		errMsg = "Database for user authentication could not be initialized."
	case 3001:
		errMsg = "The user has no authorization to load the channels."

	case 4000:
		errMsg = "Connection to streaming source was interrupted."
	case 4001:
		errMsg = "Too many errors connecting to the provider. Streaming is canceled."
	case 4002:
		errMsg = "New URL for the redirect to the streaming server is missing"
	case 4003:
		errMsg = "Server sends an incompatible content-type"
	case 4004:
		errMsg = "This error message comes from the provider"
	case 4005:
		errMsg = "Temporary buffer files could not be deleted"
	case 4006:
		errMsg = "Server connection timeout"

	case 4050:
		errMsg = "Invalid M3U8 file"
	case 4051:
		errMsg = "#EXTM3U header is missing"

	case 4100:
		errMsg = "Unknown content type for downloaded image"
	case 4101:
		errMsg = "Invalid URL, original URL is used for this image"

	case 5000:
		errMsg = "Invalid API command"

	case 6001:
		errMsg = "Invalid key"
	case 6002:
		errMsg = "Update failed"
	case 6003:
		errMsg = "Update server not available"
	case 6004:
		errMsg = "xTeVe update available"

	case 6005:
		errMsg = "XteveAutoUpdate is enabled on DietPi. The binary will be replaced outside of dietpi-software. To manage xTeVe via DietPi, disable XteveAutoUpdate in Settings."

	default:
		errMsg = fmt.Sprintf("Unknown error / warning (%d)", errCode)
	}

	return errMsg
}

func addNotification(notification Notification) (err error) {

	var t = time.Now().UnixNano() / (int64(time.Millisecond) / int64(time.Nanosecond))
	notification.Time = strconv.FormatInt(t, 10)
	notification.New = true

	if len(notification.Headline) == 0 {
		notification.Headline = strings.ToUpper(notification.Type)
	}

	if len(System.Notification) == 0 {
		System.Notification = make(map[string]Notification)
	}

	System.Notification[notification.Time] = notification

	if len(System.Notification) > 10 {
		keys := make([]string, 0, len(System.Notification))
		for key := range System.Notification {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys[:len(keys)-10] {
			delete(System.Notification, key)
		}
	}

	return
}
