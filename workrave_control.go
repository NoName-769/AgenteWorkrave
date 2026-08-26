package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"gopkg.in/ini.v1"
)

func getWorkravePath() (string, error) {

	configPath := getConfigPath()

	cfg, err := ini.Load(configPath)

	if err == nil {

		section := cfg.Section("agent")

		configuredPath := strings.TrimSpace(
			section.Key("workrave_path").String(),
		)

		if configuredPath != "" {

			if _, err := os.Stat(configuredPath); err == nil {
				return configuredPath, nil
			}

			log.Println(
				"La ruta guardada de Workrave ya no existe:",
				configuredPath,
			)
		}
	}

	if path, err := exec.LookPath("workrave.exe"); err == nil {

		log.Println("Workrave encontrado en PATH:", path)

		saveWorkravePath(path)

		return path, nil
	}

	userHome, _ := os.UserHomeDir()

	candidates := []string{
		`C:\Program Files\Workrave\bin\Workrave.exe`,
		`C:\Program Files\Workrave\lib\Workrave.exe`,
		`C:\Program Files (x86)\Workrave\bin\Workrave.exe`,
		`C:\Program Files (x86)\Workrave\lib\Workrave.exe`,
	}

	if userHome != "" {

		candidates = append(
			candidates,

			filepath.Join(
				userHome,
				"AppData",
				"Local",
				"Programs",
				"Workrave",
				"Workrave.exe",
			),

			filepath.Join(
				userHome,
				"AppData",
				"Local",
				"Programs",
				"Workrave",
				"bin",
				"Workrave.exe",
			),

			filepath.Join(
				userHome,
				"AppData",
				"Local",
				"Programs",
				"Workrave",
				"lib",
				"Workrave.exe",
			),
		)
	}

	for _, path := range candidates {

		if _, err := os.Stat(path); err == nil {

			log.Println(
				"Workrave encontrado:",
				path,
			)

			saveWorkravePath(path)

			return path, nil
		}
	}

	searchRoots := []string{
		`C:\Program Files`,
		`C:\Program Files (x86)`,
	}

	if userHome != "" {

		searchRoots = append(
			searchRoots,
			filepath.Join(
				userHome,
				"AppData",
				"Local",
				"Programs",
			),
		)
	}

	for _, root := range searchRoots {

		if _, err := os.Stat(root); err != nil {
			continue
		}

		log.Println(
			"Buscando Workrave en:",
			root,
		)

		found := ""

		err := filepath.Walk(
			root,
			func(
				path string,
				info os.FileInfo,
				err error,
			) error {

				if err != nil || info == nil {
					return nil
				}

				if info.IsDir() {
					return nil
				}

				if strings.EqualFold(
					info.Name(),
					"Workrave.exe",
				) {

					found = path

					return filepath.SkipDir
				}

				return nil
			},
		)

		if err == nil && found != "" {

			log.Println(
				"Workrave encontrado automáticamente:",
				found,
			)

			saveWorkravePath(found)

			return found, nil
		}
	}

	return "",
		fmt.Errorf(
			"no se encontró Workrave.exe automáticamente",
		)
}

func saveWorkravePath(path string) {

	configPath := getConfigPath()

	cfg, err := ini.Load(configPath)

	if err != nil {

		log.Println(
			"Error cargando agent.ini:",
			err,
		)

		return
	}

	section := cfg.Section("agent")

	section.Key(
		"workrave_path",
	).SetValue(path)

	if err := cfg.SaveTo(configPath); err != nil {

		log.Println(
			"Error guardando workrave_path:",
			err,
		)

		return
	}

	log.Println(
		"Ruta de Workrave guardada:",
		path,
	)
}

func WorkraveStatus() bool {

	_, err := getWorkraveProcess()

	return err == nil
}

func OpenWorkrave() error {

	path, err := getWorkravePath()

	if err != nil {
		return err
	}

	cmd := exec.Command(path)

	return cmd.Start()
}

func CloseWorkrave() error {

	process, err := getWorkraveProcess()

	if err != nil {
		return err
	}

	return process.Terminate()
}
