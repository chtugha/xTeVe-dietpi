package src

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"reflect"
	"strings"
	"time"

	up2date "xteve/src/internal/up2date/client"
)

// BinaryUpdate checks whether a newer version of the xTeVe binary is available
// and, if XteveAutoUpdate is enabled in settings, downloads and hot-swaps the
// running binary. Update sources are determined by the active Git branch:
//   - "master" / "beta": fetches release metadata and archives from GitHub.
//   - any other branch: contacts the custom update server URL in settings.
//
// On DietPi (DIETPI=1), auto-update defaults to disabled. If the user has
// explicitly opted in, a warning (6005) is logged before the update proceeds
// to indicate that the binary will be replaced outside of dietpi-software.
// fetchLatestReleaseInfo connects to GitHub or a custom server to get the latest version details
func fetchLatestReleaseInfo(updater *up2date.ClientInfo) (err error) {
	switch System.Branch {

	// Update von GitHub
	case "master", "beta":
		var apiURL = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", System.GitHub.User, System.GitHub.Repo)

		req, err := http.NewRequest("GET", apiURL, nil)
		if err != nil {
			return err
		}
		req.Header.Set("User-Agent", "xTeVe-Updater")

		client := &http.Client{Timeout: 15 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("%d: %s (%s)", resp.StatusCode, http.StatusText(resp.StatusCode), apiURL)
		}

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		type GitHubRelease struct {
			TagName string `json:"tag_name"`
			Assets  []struct {
				Name        string `json:"name"`
				DownloadURL string `json:"browser_download_url"`
			} `json:"assets"`
		}

		var release GitHubRelease
		err = json.Unmarshal(body, &release)
		if err != nil {
			return err
		}

		var downloadURL string
		var assetName string
		targetOS := strings.ToLower(System.OS)
		targetArch := strings.ToLower(System.ARCH)

		for _, asset := range release.Assets {
			name := strings.ToLower(asset.Name)
			if strings.HasPrefix(name, "xteve_") &&
				strings.Contains(name, targetOS) &&
				strings.Contains(name, targetArch) &&
				!strings.HasSuffix(name, ".zip") &&
				!strings.HasSuffix(name, ".tar.gz") &&
				!strings.HasSuffix(name, ".tgz") &&
				!strings.HasSuffix(name, ".md5") &&
				!strings.HasSuffix(name, ".sha256") {
				downloadURL = asset.DownloadURL
				assetName = asset.Name
				break
			}
		}

		if len(downloadURL) == 0 {
			return fmt.Errorf("no matching binary found in latest GitHub release assets for OS %s and ARCH %s", System.OS, System.ARCH)
		}

		updater.Response.Status = true
		updater.Response.UpdateBIN = downloadURL
		updater.Response.Version = strings.TrimPrefix(release.TagName, "v")
		updater.Response.Filename = assetName

	// Update vom eigenen Server
	default:
		updater.URL = Settings.UpdateURL

		if len(updater.URL) == 0 {
			return fmt.Errorf("no server URL specified in Settings.UpdateURL")
		}

		showInfo("Update URL:" + updater.URL)

		err = up2date.GetVersion()
		if err != nil {
			return err
		}

		if len(updater.Response.Reason) > 0 {
			return fmt.Errorf("update server: %s", updater.Response.Reason)
		}
	}

	return nil
}

// BinaryUpdate checks whether a newer version of the xTeVe binary is available
// and, if XteveAutoUpdate is enabled in settings, downloads and hot-swaps the
// running binary. Update sources are determined by the active Git branch:
//   - "master" / "beta": fetches release metadata and archives from GitHub.
//   - any other branch: contacts the custom update server URL in settings.
//
// On DietPi (DIETPI=1), auto-update defaults to disabled. If the user has
// explicitly opted in, a warning (6005) is logged before the update proceeds
// to indicate that the binary will be replaced outside of dietpi-software.
func BinaryUpdate() (err error) {

	if !System.GitHub.Update {
		showWarning(2099)
		return
	}

	var updater = &up2date.Updater
	updater.Name = System.Update.Name
	updater.Branch = System.Branch

	up2date.Init()

	err = fetchLatestReleaseInfo(updater)
	if err != nil {
		ShowError(err, 6003)
		return nil
	}

	var currentVersion = System.Version + "." + System.Build

	// Versionsnummer überprüfen
	if compareVersions(updater.Response.Version, currentVersion) > 0 && updater.Response.Status {

		if Settings.XteveAutoUpdate {

			if os.Getenv("DIETPI") == "1" {
				showWarning(6005)
			}

			var fileType, url string

			showInfo(fmt.Sprintf("Update Available:Version: %s", updater.Response.Version))

			switch System.Branch {

			// Update von GitHub
			case "master", "beta":
				showInfo("Update Server:GitHub")

			// Update vom eigenen Server
			default:
				showInfo(fmt.Sprintf("Update Server:%s", Settings.UpdateURL))

			}

			showInfo(fmt.Sprintf("Start Update:Branch: %s", updater.Branch))

			// Neue Version als BIN Datei herunterladen
			if len(updater.Response.UpdateBIN) > 0 {
				url = updater.Response.UpdateBIN
				fileType = "bin"
			}

			// Neue Version als ZIP Datei herunterladen
			if len(updater.Response.UpdateZIP) > 0 {
				url = updater.Response.UpdateZIP
				fileType = "zip"
			}

			if len(url) > 0 {

				err = up2date.DoUpdate(fileType, updater.Response.Filename)
				if err != nil {
					ShowError(err, 6002)
				}

			}

		} else {
			// Hinweis ausgeben
			showWarning(6004)
		}

	}

	return nil
}

func conditionalUpdateChanges() (err error) {

checkVersion:
	settingsMap, err := loadJSONFileToMap(System.File.Settings)
	if err != nil || len(settingsMap) == 0 {
		return
	}

	if settingsVersion, ok := settingsMap["version"].(string); ok {

		if settingsVersion > System.DBVersion {
			showInfo("Settings DB Version:" + settingsVersion)
			showInfo("System DB Version:" + System.DBVersion)
			err = errors.New(getErrMsg(1031))
			return
		}

		// Letzte Kompatible Version (1.4.4)
		if settingsVersion < System.Compatibility {
			err = errors.New(getErrMsg(1013))
			return
		}

		switch settingsVersion {

		case "1.4.4":
			// UUID Wert in xepg.json setzen
			err = setValueForUUID()
			if err != nil {
				return
			}

			// Neuer Filter (WebUI). Alte Filtereinstellungen werden konvertiert
			if oldFilter, ok := settingsMap["filter"].([]interface{}); ok {
				var newFilterMap = convertToNewFilter(oldFilter)
				settingsMap["filter"] = newFilterMap

				settingsMap["version"] = "2.0.0"

				err = saveMapToJSONFile(System.File.Settings, settingsMap)
				if err != nil {
					return
				}

				goto checkVersion

			} else {
				err = errors.New(getErrMsg(1030))
				return
			}

		case "2.0.0":

			if oldBuffer, ok := settingsMap["buffer"].(bool); ok {

				var newBuffer string
				switch oldBuffer {
				case true:
					newBuffer = "xteve"
				case false:
					newBuffer = "-"
				}

				settingsMap["buffer"] = newBuffer

				settingsMap["version"] = "2.1.0"

				err = saveMapToJSONFile(System.File.Settings, settingsMap)
				if err != nil {
					return
				}

				goto checkVersion

			} else {
				err = errors.New(getErrMsg(1030))
				return
			}

		case "2.1.0":
			// Falls es in einem späteren Update Änderungen an der Datenbank gibt, geht es hier weiter

			break
		}

	} else {
		// settings.json ist zu alt (älter als Version 1.4.4)
		err = errors.New(getErrMsg(1013))
	}

	return
}

func convertToNewFilter(oldFilter []interface{}) (newFilterMap map[int]interface{}) {

	newFilterMap = make(map[int]interface{})

	switch reflect.TypeOf(oldFilter).Kind() {

	case reflect.Slice:
		s := reflect.ValueOf(oldFilter)

		for i := 0; i < s.Len(); i++ {

			var newFilter FilterStruct
			newFilter.Active = true
			newFilter.Name = fmt.Sprintf("Custom filter %d", i+1)
			newFilter.Filter = s.Index(i).Interface().(string)
			newFilter.Type = "custom-filter"
			newFilter.CaseSensitive = false

			newFilterMap[i] = newFilter

		}

	}

	return
}

func setValueForUUID() (err error) {

	xepg, err := loadJSONFileToMap(System.File.XEPG)
	if err != nil {
		return
	}

	for _, c := range xepg {

		var xepgChannel = c.(map[string]interface{})

		if uuidKey, ok := xepgChannel["_uuid.key"].(string); ok {

			if value, ok := xepgChannel[uuidKey].(string); ok {

				if len(value) > 0 {
					xepgChannel["_uuid.value"] = value
				}

			}

		}

	}

	err = saveMapToJSONFile(System.File.XEPG, xepg)

	return
}
