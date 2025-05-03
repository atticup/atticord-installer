package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/eiannone/keyboard"
	"github.com/schollz/progressbar/v3"
)

var venP = filepath.Join(os.Getenv("APPDATA"), "Vencord")
var distP = filepath.Join(venP, "dist")
var asarFN = "atticord.asar"
var configF = filepath.Join(venP, "config.json")
var possibleClients = []string{"Discord", "DiscordCanary", "DiscordPTB"}

type Config struct {
	Hash string `json:"hash"`
	Key  string `json:"key"`
}
type LV struct {
	Hash string `json:"hash"`
}

func main() {
	fmt.Println("BTW YOU NEED NODEJS TO RUN THIS (IF U HAVEN'T DOWNLOADED ALREADY PLEASE DO)")
	fmt.Println("(if u see this right before u opened discord it means atticord has a new update)")
	if _, err := os.Stat(venP); os.IsNotExist(err) {
		err := os.MkdirAll(venP, os.ModePerm)
		if err != nil {
			fmt.Println("ERROR creating vencord folder (please report this to atticus):", err)
		} else {
			fmt.Println("Successfully replicated the vencord folder")
		}
	}
	key, err := getkey()
	if err != nil {
		fmt.Printf("Key setup failed (most likely your fault report this to atticus anyway) %v\n", err)
		waitForExit()
		return
	}
	latestV, err := getLV(fmt.Sprintf("http://ro-premium.pylex.xyz:9304/latest.json?key=%s", key))
	if err != nil {
		fmt.Printf("ERROR getting the version (please check and make sure u put the correct key) %v\n", err)
		waitForExit()
		return
	}
	_ = updateConfig(Config{Hash: latestV.Hash, Key: key})
	_ = keyboard.Open()
	defer keyboard.Close()
	fmt.Println("Installing 'asar' via npm. Press K now to change your key before it finishes.")
	done := make(chan struct{})
	go func() {
		_ = exec.Command("npm", "install", "-g", "asar").Run()
		close(done)
	}()
	keypress := make(chan rune)
	go func() {
		for {
			char, _, err := keyboard.GetKey()
			if err == nil {
				keypress <- char
			}
		}
	}()
waitLoop:
	for {
		select {
		case <-done:
			break waitLoop
		case char := <-keypress:
			if char == 'k' || char == 'K' {
				fmt.Print("Enter new key: ")
				newInput, _ := bufio.NewReader(os.Stdin).ReadString('\n')
				key = strings.TrimSpace(newInput)
				_ = updateConfig(Config{Hash: latestV.Hash, Key: key})
				fmt.Println("Key updated. Continuing...")
				break waitLoop
			}
		case <-time.After(100 * time.Millisecond):
		}
	}

	_ = killdiscordproc()
	_ = os.RemoveAll(distP)
	_ = os.MkdirAll(distP, os.ModePerm)
	asarPath := filepath.Join(distP, asarFN)
	_ = downloadF(fmt.Sprintf("http://ro-premium.pylex.xyz:9304/atticord.asar?key=%s", key), asarPath)
	_ = extractasarF(asarPath, distP)
	_ = os.Remove(asarPath)
	selectedClients := selectInstalledClients()
	fmt.Println("You're almost done!!! all u have to do is PRESS ENTER now to continue and then just wait till it closes itself")
	injectasar2discord(key, selectedClients)
	finalcleanups(key, selectedClients)
	fmt.Println("you can start atticord now (yippeee right?)")
	waitForExit()
}

func waitForExit() {
	fmt.Print("Press Enter to exit... (or stay here i don't really judge)")
	bufio.NewReader(os.Stdin).ReadString('\n')
}

func getkey() (string, error) {
	if _, err := os.Stat(configF); os.IsNotExist(err) {
		fmt.Print("Enter your key: ")
		input, _ := bufio.NewReader(os.Stdin).ReadString('\n')
		key := strings.TrimSpace(input)
		_ = os.MkdirAll(venP, os.ModePerm)
		data, _ := json.MarshalIndent(Config{Key: key}, "", "  ")
		_ = os.WriteFile(configF, data, 0644)
		return key, nil
	}
	data, _ := os.ReadFile(configF)
	var cfg Config
	_ = json.Unmarshal(data, &cfg)
	return cfg.Key, nil
}

func getLV(url string) (*LV, error) {
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	var latest LV
	_ = json.Unmarshal(body, &latest)
	return &latest, nil
}

func updateConfig(cfg Config) error {
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(configF, data, 0644)
}

func killdiscordproc() error {
	for _, name := range possibleClients {
		_ = exec.Command("taskkill", "/F", "/IM", name+".exe").Run()
		fmt.Printf("Took care of (terminated): %s\n", name)
	}
	return nil
}

func extractasarF(asarPath, destPath string) error {
	return exec.Command("asar", "extract", asarPath, destPath).Run()
}

func downloadF(url, destPath string) error {
	resp, _ := http.Get(url)
	defer resp.Body.Close()
	out, _ := os.Create(destPath)
	defer out.Close()
	bar := progressbar.NewOptions(int(resp.ContentLength),
		progressbar.OptionSetWidth(20),
		progressbar.OptionSetDescription("Downloading atticord.asar (may take a bit depending on your wifi speed)"),
		progressbar.OptionSetRenderBlankState(true),
	)
	buf := make([]byte, 1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			_, _ = out.Write(buf[:n])
			_ = bar.Add(n)
		}
		if err == io.EOF || bar.IsFinished() {
			break
		}
	}
	_ = bar.Finish()
	return nil
}

// (some of the next parts are partly codded by chatgpt (sadly i'm a beginner when it comes to go) so thanks chatty please let me know if u run into any errors in those parts since i don't fully know how they function yet they may not be perfect)
func selectInstalledClients() []string {
	appData := os.Getenv("LOCALAPPDATA")
	installed := []string{}
	for _, client := range possibleClients {
		if _, err := os.Stat(filepath.Join(appData, client)); err == nil {
			installed = append(installed, client)
		}
	}
	selected := make([]bool, len(installed))
	for i := range selected {
		selected[i] = true
	}
	_ = keyboard.Open()
	defer keyboard.Close()
	current := 0
	printInstructions := func() {
		fmt.Println("Use arrow keys to navigate, Space to toggle, X means selected, Enter to confirm (2x)")
	}
	render := func() {
		fmt.Print("\033[H\033[2J")
		printInstructions()
		for i, client := range installed {
			cursor := "  "
			if i == current {
				cursor = "->"
			}
			mark := "[ ]"
			if selected[i] {
				mark = "[x]"
			}
			fmt.Printf("%s %s %s\n", cursor, mark, client)
		}
	}
	render()
	screenUpdated := false
	for {
		char, key, _ := keyboard.GetKey()
		if key == keyboard.KeyArrowDown {
			current = (current + 1) % len(installed)
			screenUpdated = true
		} else if key == keyboard.KeyArrowUp {
			current = (current - 1 + len(installed)) % len(installed)
			screenUpdated = true
		} else if key == keyboard.KeySpace {
			selected[current] = !selected[current]
			screenUpdated = true
		} else if key == keyboard.KeyEnter || char == '\r' {
			break
		}
		if screenUpdated {
			render()
			screenUpdated = false
		}
	}
	final := []string{}
	for i, sel := range selected {
		if sel {
			final = append(final, installed[i])
		}
	}
	return final
}

func injectasar2discord(key string, clients []string) {
	appData := os.Getenv("LOCALAPPDATA")
	for _, client := range clients {
		clientPath := filepath.Join(appData, client)
		entries, _ := os.ReadDir(clientPath)
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "app-") {
				continue
			}
			resourcePath := filepath.Join(clientPath, entry.Name(), "resources")
			if _, err := os.Stat(resourcePath); os.IsNotExist(err) {
				continue
			}
			asarFile := filepath.Join(resourcePath, "app.asar")
			backupAsar := filepath.Join(resourcePath, "_app.asar")
			useAsar := backupAsar

			if _, err := os.Stat(backupAsar); os.IsNotExist(err) {
				if _, err := os.Stat(asarFile); os.IsNotExist(err) {
					fmt.Printf("SKIPPED app.asar nor _app.asar exists at %s\n", resourcePath)
					continue
				}
				err = os.Rename(asarFile, backupAsar)
				if err != nil {
					fmt.Printf("ERROR renaming app.asar %v\n", err)
					continue
				}
				fmt.Printf("Successfully injected atticord into a nonvencord installation %s\n", resourcePath)
			}

			rand.Seed(time.Now().UnixNano())
			atticorddir := filepath.Join(resourcePath, fmt.Sprintf("atticord%d", rand.Intn(1000000)))
			err := os.MkdirAll(atticorddir, os.ModePerm)
			if err != nil {
				fmt.Printf("ERROR creating atticord dir (REPORT TO ATTICUS) %v\n", err)
				continue
			}

			cmd := exec.Command("asar", "extract", useAsar, atticorddir)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				fmt.Printf("ERROR extracting asar (REPORT TO ATTICUS) %s: %v\n", resourcePath, err)
				continue
			}

			videosPath := filepath.Join(atticorddir, "app_bootstrap", "videos")
			os.MkdirAll(videosPath, os.ModePerm)
			connectingPath := filepath.Join(videosPath, "connecting.webm")
			os.Remove(connectingPath)
			videoResp, err := http.Get("https://github.com/atticup/atticord-installer/raw/refs/heads/main/connecting.webm")
			if err != nil {
				fmt.Printf("ERROR downloading splashscreen (REPORT TO ATTICUS) %v\n", err)
				continue
			}
			defer videoResp.Body.Close()
			outFile, err := os.Create(connectingPath)
			if err != nil {
				fmt.Printf("ERROR creating splashscreen (REPORT TO ATTICUS) %v\n", err)
				continue
			}
			_, err = io.Copy(outFile, videoResp.Body)
			outFile.Close()
			if err != nil {
				fmt.Printf("ERROR saving splashscreen (REPORT TO ATTICUS) %v\n", err)
				continue
			}

			autoStartPath := filepath.Join(atticorddir, "app_bootstrap", "autoStart")
			win32JSPath := filepath.Join(autoStartPath, "win32.js")
			if _, err := os.Stat(win32JSPath); err == nil {
				err := os.Remove(win32JSPath)
				if err != nil {
					fmt.Printf("ERROR removing existing win32.js %v\n", err)
					continue
				}
			}

			win32JSURL := "https://github.com/atticup/atticord-installer/raw/refs/heads/main/win32.js"
			win32JSResp, err := http.Get(win32JSURL)
			if err != nil {
				fmt.Printf("ERROR downloading win32.js %v\n", err)
				continue
			}
			defer win32JSResp.Body.Close()
			win32JSFile, err := os.Create(win32JSPath)
			if err != nil {
				fmt.Printf("ERROR creating win32.js %v\n", err)
				continue
			}
			_, err = io.Copy(win32JSFile, win32JSResp.Body)
			win32JSFile.Close()
			if err != nil {
				fmt.Printf("ERROR saving win32.js %v\n", err)
				continue
			}

			// Clean up and finalize
			os.Remove(backupAsar)
			cmd = exec.Command("asar", "pack", atticorddir, backupAsar)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			err = cmd.Run()
			if err != nil {
				fmt.Printf("ERROR repacking asar (REPORT TO ATTICUS) %v\n", err)
				continue
			}
			os.RemoveAll(atticorddir)
			files, _ := os.ReadDir(resourcePath)
			for _, f := range files {
				if strings.HasSuffix(f.Name(), ".asar") && f.Name() != "_app.asar" {
					os.Remove(filepath.Join(resourcePath, f.Name()))
				}
			}
			finalAsar := filepath.Join(resourcePath, "app.asar")
			url := fmt.Sprintf("http://ro-premium.pylex.xyz:9304/app.asar?key=%s", key)
			err = downloadF(url, finalAsar)
			if err != nil {
				fmt.Printf("ERROR failed to download app.asar (REPORT TO ATTICUS) %v\n", err)
				continue
			}

			updaterP := filepath.Join(clientPath, entry.Name(), "atticordupdater.exe")
			updaterResp, err := http.Get("https://github.com/atticup/atticord-installer/raw/refs/heads/main/atticordupdater.exe")
			if err != nil {
				fmt.Printf("ERROR downloading atticordupdater.exe (REPORT TO ATTICUS) %v\n", err)
				continue
			}
			defer updaterResp.Body.Close()
			updaterFile, err := os.Create(updaterP)
			if err != nil {
				fmt.Printf("ERROR creating atticordupdater.exe (REPORT TO ATTICUS) %v\n", err)
				continue
			}
			_, err = io.Copy(updaterFile, updaterResp.Body)
			updaterFile.Close()
			if err != nil {
				fmt.Printf("ERROR saving atticordupdater.exe (REPORT TO ATTICUS) %v\n", err)
				continue
			}

			fmt.Printf("Completed injection for %s\n", resourcePath)
		}
	}
}

func finalcleanups(key string, clients []string) {
	appData := os.Getenv("LOCALAPPDATA")
	for _, client := range clients {
		clientPath := filepath.Join(appData, client)
		entries, _ := os.ReadDir(clientPath)
		for _, entry := range entries {
			if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "app-") {
				continue
			}
			resourcePath := filepath.Join(clientPath, entry.Name(), "resources")
			asarPath := filepath.Join(resourcePath, "app.asar")
			info, err := os.Stat(asarPath)
			if err == nil && info.IsDir() {
				_ = os.RemoveAll(asarPath)
				url := fmt.Sprintf("http://ro-premium.pylex.xyz:9304/app.asar?key=%s", key)
				_ = downloadF(url, asarPath)
				fmt.Printf("Successfully saved possible crash in folder: %s\n", resourcePath)
			}
		}
	}
}
