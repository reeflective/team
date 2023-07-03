package assets

// Wiregost - Post-Exploitation & Implant Framework
// Copyright © 2020 Para
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

// const (
// 	settingsFileName = "settings.json"
// )
//
// // ClientSettings - Client JSON config
// type ClientSettings struct {
// 	ConsoleLogs bool `json:"console_logs"`
// }

// // LoadSettings - Load the client settings from disk
// func LoadSettings() (*ClientSettings, error) {
// 	rootDir, _ := filepath.Abs(GetRootAppDir())
// 	data, err := os.ReadFile(filepath.Join(rootDir, settingsFileName))
// 	if err != nil {
// 		return defaultSettings(), err
// 	}
// 	settings := defaultSettings()
// 	err = json.Unmarshal(data, settings)
// 	if err != nil {
// 		return defaultSettings(), err
// 	}
// 	return settings, nil
// }
//
// func defaultSettings() *ClientSettings {
// 	return &ClientSettings{
// 		ConsoleLogs: true,
// 	}
// }
//
// // SaveSettings - Save the current settings to disk
// func SaveSettings(settings *ClientSettings) error {
// 	rootDir, _ := filepath.Abs(GetRootAppDir())
// 	if settings == nil {
// 		settings = defaultSettings()
// 	}
// 	data, err := json.MarshalIndent(settings, "", "  ")
// 	if err != nil {
// 		return err
// 	}
// 	err = os.WriteFile(filepath.Join(rootDir, settingsFileName), data, 0o600)
// 	return err
// }
