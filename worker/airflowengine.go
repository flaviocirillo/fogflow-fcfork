package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	. "fogflow/common/config"
	. "fogflow/common/datamodel"
	. "fogflow/common/ngsi"
)

type AirflowEngine struct {
	remotePyModuleRepository string
	localPyModuleRepository  string
	dagTemplatePath          string
	airflowHome              string
}

func (airflowengine *AirflowEngine) Init(cfg *Config) bool {
	airflowengine.remotePyModuleRepository = "http://fogflow.io/pythonmodulerepository/"
	airflowengine.localPyModuleRepository = "local_pymodules_repository/"

	airflowengine.dagTemplatePath = "airflow_dag_template.py"
	airflowengine.airflowHome = "airflow_home/"

	return true
}

func (airflowengine *AirflowEngine) generateDAG(pythonModule string, pythonPackage string, taskId string, configuration string) {

	// importModule := fmt.Sprintf("sys.path.insert(0, '%s.zip')", pythonModule)
	// importModule := fmt.Sprintf("import %s", pythonModule)

	// importPath := strings.ReplaceAll(pythonPackage, "/", ".")
	importLine := fmt.Sprintf("from %s.%s import handleEntity", airflowengine.getLibFolderName(pythonModule, taskId), pythonPackage)

	configurationLine := fmt.Sprintf("configurations = %s", configuration)

	// Read the DAG template file
	inputFile, err := os.Open(airflowengine.dagTemplatePath)
	if err != nil {
		ERROR.Println("Error opening DAG template:", err)
		os.Exit(1)
	}
	defer inputFile.Close()

	// Read and modify the PLACEHOLDERS
	var lines []string
	scanner := bufio.NewScanner(inputFile)
	for scanner.Scan() {
		line := scanner.Text()
		lines = append(lines, line)

		// if strings.Contains(line, "MODULE PLACEHOLDER:") {
		// 	lines = append(lines, importModule)
		// }

		if strings.Contains(line, "FOGFUNCTION PLACEHOLDER:") {
			lines = append(lines, importLine)
		}

		if strings.Contains(line, "CONFIGURATION PLACEHOLDER:") {
			lines = append(lines, configurationLine)
		}
	}

	if err := scanner.Err(); err != nil {
		ERROR.Println("Error reading DAG template:", err)
		os.Exit(1)
	}

	dagName := fmt.Sprintf("%s_%s_%s_dag.py", pythonModule, pythonPackage, airflowengine.getModuleName(pythonModule, taskId))
	// Write back the modified content
	outputFile, err := os.Create(airflowengine.airflowHome + dagName)
	if err != nil {
		ERROR.Println("Error writing DAG template:", err)
		os.Exit(1)
	}
	defer outputFile.Close()

	for _, line := range lines {
		_, err := outputFile.WriteString(line + "\n")
		if err != nil {
			ERROR.Println("Error writing DAG template:", err)
			os.Exit(1)
		}
	}

	fmt.Println("Successfully inserted import line:", importLine)

}

// func addFileToZip(zipWriter *zip.Writer, path string, baseInZip string) error {
// 	info, err := os.Stat(path)
// 	if err != nil {
// 		return err
// 	}

// 	if info.IsDir() {
// 		entries, err := os.ReadDir(path)
// 		if err != nil {
// 			return err
// 		}
// 		for _, entry := range entries {
// 			entryPath := filepath.Join(path, entry.Name())
// 			entryBase := filepath.Join(baseInZip, entry.Name())
// 			if err := addFileToZip(zipWriter, entryPath, entryBase); err != nil {
// 				return err
// 			}
// 		}
// 	} else {
// 		fileToZip, err := os.Open(path)
// 		if err != nil {
// 			return err
// 		}
// 		defer fileToZip.Close()

// 		writer, err := zipWriter.Create(baseInZip)
// 		if err != nil {
// 			return err
// 		}
// 		_, err = io.Copy(writer, fileToZip)
// 		if err != nil {
// 			return err
// 		}
// 	}
// 	return nil
// }

// func (airflowengine *AirflowEngine) importModule(moduleName string) error {

// 	localRepo := airflowengine.localPyModuleRepository
// 	remoteRepo := airflowengine.remotePyModuleRepository
// 	destDir := airflowengine.airflowHome

// 	folderPath := filepath.Join(localRepo, moduleName)
// 	zipName := moduleName + ".zip"
// 	zipPath := filepath.Join(localRepo, zipName)
// 	destZipPath := filepath.Join(destDir, zipName)

// 	// 1. Check if folder exists
// 	if info, err := os.Stat(folderPath); err == nil && info.IsDir() {
// 		INFO.Println("Found folder:", folderPath)

// 		// Zip the folder contents (not the folder itself)
// 		files, err := os.ReadDir(folderPath)
// 		if err != nil {
// 			return fmt.Errorf("failed to read folder contents: %w", err)
// 		}

// 		zipFile, err := os.Create(destZipPath)
// 		if err != nil {
// 			return fmt.Errorf("failed to create zip file: %w", err)
// 		}
// 		defer zipFile.Close()

// 		zipWriter := zip.NewWriter(zipFile)
// 		defer zipWriter.Close()

// 		for _, file := range files {
// 			filePath := filepath.Join(folderPath, file.Name())
// 			err = addFileToZip(zipWriter, filePath, file.Name())
// 			if err != nil {
// 				return fmt.Errorf("failed to add file to zip: %w", err)
// 			}
// 		}

// 		// // Zip the folder
// 		// err := airflowengine.zipFolder(folderPath, destZipPath)
// 		// if err != nil {
// 		// 	ERROR.Println("failed to zip folder: %w", err)
// 		// 	return fmt.Errorf("failed to zip folder: %w", err)
// 		// }
// 		INFO.Println("Zipped and copied to:", destZipPath)
// 		return nil
// 	}

// 	// 2. If not a folder, check if .zip file exists
// 	if _, err := os.Stat(zipPath); err == nil {
// 		INFO.Println("Found zip file:", zipPath)
// 		// Copy to destination
// 		err := airflowengine.copyFile(zipPath, destZipPath)
// 		if err != nil {
// 			ERROR.Println("failed to copy zip: %w", err)
// 			return fmt.Errorf("failed to copy zip: %w", err)
// 		}
// 		INFO.Println("Copied zip to:", destZipPath)
// 		return nil
// 	}

// 	// 3. If neither found, download
// 	INFO.Println("Module not found locally, downloading from external repo:", airflowengine.remotePyModuleRepository)
// 	url := remoteRepo + "/" + moduleName
// 	airflowengine.importPythonModuleFromRemote(url, airflowengine.airflowHome, moduleName)

// 	return nil
// }

// func (airflowengine *AirflowEngine) zipFolder(srcDir, destZip string) error {
// 	zipfile, err := os.Create(destZip)
// 	if err != nil {
// 		return err
// 	}
// 	defer zipfile.Close()

// 	archive := zip.NewWriter(zipfile)
// 	defer archive.Close()

// 	return filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
// 		if err != nil {
// 			return err
// 		}
// 		relPath, err := filepath.Rel(filepath.Dir(srcDir), path)
// 		if err != nil {
// 			return err
// 		}
// 		if info.IsDir() {
// 			return nil // skip folders, added automatically
// 		}
// 		file, err := os.Open(path)
// 		if err != nil {
// 			return err
// 		}
// 		defer file.Close()

// 		writer, err := archive.Create(relPath)
// 		if err != nil {
// 			return err
// 		}
// 		_, err = io.Copy(writer, file)
// 		return err
// 	})
// }

// func (airflowengine *AirflowEngine) copyFile(src, dst string) error {
// 	in, err := os.Open(src)
// 	if err != nil {
// 		return err
// 	}
// 	defer in.Close()

// 	out, err := os.Create(dst)
// 	if err != nil {
// 		return err
// 	}
// 	defer func() {
// 		if cerr := out.Close(); cerr != nil && err == nil {
// 			err = cerr
// 		}
// 	}()

// 	_, err = io.Copy(out, in)
// 	return err
// }

// Helper to copy individual files
func copyFile(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return os.Chmod(dst, perm)
}

// Helper to copy individual files
func copyFolder(srcFolder, dstFolder string) error {
	// Check if source folder exists
	srcInfo, err := os.Stat(srcFolder)
	if err != nil {
		return fmt.Errorf("source folder not found: %w", err)
	}
	if !srcInfo.IsDir() {
		return fmt.Errorf("source path is not a directory: %s", srcFolder)
	}

	// Ensure dstFolder exists
	if err := os.MkdirAll(dstFolder, 0755); err != nil {
		return fmt.Errorf("failed to create dstFolder: %w", err)
	}

	// Walk through source and copy each file/dir
	err = filepath.Walk(srcFolder, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(srcFolder, srcPath)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dstFolder, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Copy file
		return copyFile(srcPath, destPath, info.Mode())
	})
	if err != nil {
		return fmt.Errorf("failed to copy folder: %w", err)
	}

	return nil
}

// Example:
// libFolderName = taskId_lib
// libFolderPath = airflow-home/taskId_lib/
// libFolder = taskId_lib/

// Example:
// sanitizedModuleName = Task_TestPython_Main_2542520784
func (airflowengine *AirflowEngine) getModuleName(pythonModule string, taskId string) string {
	module := ""
	if taskId == "" {
		module = pythonModule
	} else {
		module = taskId
	}
	sanitizedModuleName := strings.ReplaceAll(module, ".", "_")
	sanitizedModuleName = strings.ReplaceAll(sanitizedModuleName, "-", "_")
	if LoggerIsEnabled(DEBUG) {
		DEBUG.Println("ModuleName: ", sanitizedModuleName)
	}
	return sanitizedModuleName
}

// Return:
// libFolderName = Task_TestPython_Main_2542520784_lib
func (airflowengine *AirflowEngine) getLibFolderName(pythonModule string, taskId string) string {
	folderName := airflowengine.getModuleName(pythonModule, taskId) + "_lib"
	if LoggerIsEnabled(DEBUG) {
		DEBUG.Println("LibFolderName: ", folderName)
	}
	return folderName
}

// Return:
// libFolderName = Task_TestPython_Main_2542520784_lib/
func (airflowengine *AirflowEngine) getLibFolder(pythonModule string, taskId string) string {
	libFolder := airflowengine.getLibFolderName(pythonModule, taskId) + "/"
	if LoggerIsEnabled(DEBUG) {
		DEBUG.Println("LibFolder: ", libFolder)
	}
	return libFolder

}

// Return:
// libFolderPath = airflow-home/Task_TestPython_Main_2542520784_lib/
func (airflowengine *AirflowEngine) getLibFolderPath(pythonModule string, taskId string) string {
	libFolderPath := filepath.Join(airflowengine.airflowHome, airflowengine.getLibFolder(pythonModule, taskId))
	if LoggerIsEnabled(DEBUG) {
		DEBUG.Println("LibFolderPath: ", libFolderPath)
	}
	return libFolderPath
}

// // return the new libdirectory name and full path
// func (airflowengine *AirflowEngine) getLibDir(folder string) (string, string) {
// 	sanitizedFolderName := strings.ReplaceAll(folder, ".", "_") + "_lib"
// 	sanitizedFolderName = strings.ReplaceAll(sanitizedFolderName, "-", "_")
// 	libDir := filepath.Join(airflowengine.airflowHome, sanitizedFolderName)
// 	return sanitizedFolderName, libDir
// }

func (airflowengine *AirflowEngine) importModuleAsFolder(moduleName string, taskId string) error {

	localRepo := airflowengine.localPyModuleRepository
	destDir := airflowengine.getLibFolderPath(moduleName, taskId)
	// if taskId == "" {
	// 	_, destDir = airflowengine.getLibDir(moduleName)
	// } else {
	// 	_, destDir = airflowengine.getLibDir(taskId)
	// }
	// destDir := airflowengine.airflowHome

	srcFolder := filepath.Join(localRepo, moduleName)

	copyFolder(srcFolder, destDir)

	fmt.Printf("[INFO] Module %s imported into %s\n", moduleName, destDir)
	return nil

}

// downloadZip downloads a ZIP file from a URL and saves it in targetFolder with the given filename.
func (airflowengine *AirflowEngine) importPythonModuleFromRemote(url string, targetFolder string, filename string) error {

	// Build the full path
	filePath := filepath.Join(targetFolder, filename)

	// Send GET request
	resp, err := http.Get(url)
	if err != nil {
		ERROR.Println("failed to download file: %w", err)
		return fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// Check if response is OK
	if resp.StatusCode != http.StatusOK {
		ERROR.Printf("bad status code: %s", resp.Status)
		return fmt.Errorf("bad status code: %s", resp.Status)
	}

	// Create the destination file
	outFile, err := os.Create(filePath)
	if err != nil {
		ERROR.Println("failed to create file: %w", err)
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer outFile.Close()

	// Copy the content
	_, err = io.Copy(outFile, resp.Body)
	if err != nil {
		ERROR.Println("failed to write file content: %w", err)
		return fmt.Errorf("failed to write file content: %w", err)
	}

	INFO.Printf("Downloaded file to: %s", filePath)
	// fmt.Printf("Downloaded file to: %s\n", filePath)
	return nil
}

// ensureIgnore adds a line to .airflowignore to ignore folder
func (airflowengine *AirflowEngine) airflowIgnore(dagsDir string, pathToIgnore string) error {

	ignoreFile := filepath.Join(dagsDir, ".airflowignore")

	// Create file if it doesn't exist
	if _, err := os.Stat(ignoreFile); os.IsNotExist(err) {
		file, err := os.Create(ignoreFile)
		if err != nil {
			ERROR.Println("error creating .airflowignore: %w", err)
			return fmt.Errorf("error creating .airflowignore: %w", err)
		}
		defer file.Close()
	}

	// Check if pathToIgnore is already ignored
	file, err := os.Open(ignoreFile)
	if err != nil {
		ERROR.Println("error opening .airflowignore: %w", err)
		return fmt.Errorf("error opening .airflowignore: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	alreadyPresent := false
	for scanner.Scan() {
		if strings.TrimSpace(scanner.Text()) == pathToIgnore {
			alreadyPresent = true
			break
		}
	}
	if err := scanner.Err(); err != nil {
		ERROR.Println("error reading .airflowignore: %w", err)
		return fmt.Errorf("error reading .airflowignore: %w", err)
	}

	// If not present, append it
	if !alreadyPresent {
		f, err := os.OpenFile(ignoreFile, os.O_APPEND|os.O_WRONLY, 0644)
		if err != nil {
			ERROR.Println("error appending .airflowignore: %w", err)
			return fmt.Errorf("error appending to .airflowignore: %w", err)
		}
		defer f.Close()

		if _, err := f.WriteString(pathToIgnore + "\n"); err != nil {
			ERROR.Println("error writing .airflowignore: %w", err)
			return fmt.Errorf("error writing to .airflowignore: %w", err)
		}
		// fmt.Printf("Added %s to .airflowignore\n", zipFilename)
		INFO.Printf("Added %s to .airflowignore\n", pathToIgnore)

	} else {
		// fmt.Printf("%s is already in .airflowignore\n", zipFilename)
		INFO.Printf("%s is already in .airflowignore\n", pathToIgnore)

	}

	return nil
}

// func (airflowengine *AirflowEngine) generateCommandsList(task *ScheduledTaskInstance, brokerURL string) string {

// 	// configure the task with its output streams via its admin interface
// 	commands := make([]interface{}, 0)

// 	// set broker URL
// 	setBrokerCmd := make(map[string]interface{})
// 	setBrokerCmd["command"] = "CONNECT_BROKER"
// 	setBrokerCmd["brokerURL"] = brokerURL
// 	commands = append(commands, setBrokerCmd)

// 	// set CorrelatorID
// 	setCorrelatorCmd := make(map[string]interface{})
// 	setCorrelatorCmd["command"] = "SET_CORRELATORID"
// 	setCorrelatorCmd["correlatorID"] = task.ID
// 	commands = append(commands, setCorrelatorCmd)

// 	// set input stream
// 	for _, inputStream := range task.Inputs {
// 		setInputCmd := make(map[string]interface{})
// 		setInputCmd["command"] = "SET_INPUTS"
// 		setInputCmd["type"] = inputStream.Type
// 		setInputCmd["id"] = inputStream.ID
// 		setInputCmd["attributes"] = inputStream.AttributeList
// 		commands = append(commands, setInputCmd)
// 	}

// 	// set output stream
// 	for _, outStream := range task.Outputs {
// 		setOutputCmd := make(map[string]interface{})
// 		setOutputCmd["command"] = "SET_OUTPUTS"
// 		setOutputCmd["type"] = outStream.Type
// 		setOutputCmd["id"] = outStream.StreamID
// 		commands = append(commands, setOutputCmd)
// 	}

// 	for _, parameter := range task.Parameters {

// 		setParameterCmd := make(map[string]interface{})
// 		setParameterCmd["name"] = parameter.Name
// 		setParameterCmd["value"] = parameter.Value
// 		commands = append(commands, setParameterCmd)

// 	}

// 	// pass the initial configuration as the environmental variable
// 	jsonBytes, _ := json.Marshal(commands)

// 	return string(jsonBytes)
// }

func (airflowengine *AirflowEngine) PullImage(moduleName string, ephemeralId ...string) (string, error) {

	taskId := ""
	if len(ephemeralId) > 0 {
		taskId = ephemeralId[0]
	}

	// module := moduleName + ".zip"
	// airflowengine.airflowIgnore(airflowengine.airflowHome, moduleName, taskId)
	// moduleFolderName := ""
	// if taskId == "" {
	// 	moduleFolderName, _ = airflowengine.getLibDir(moduleName)
	// } else {
	// 	moduleFolderName, _ = airflowengine.getLibDir(taskId)
	// }
	// if !strings.HasSuffix(moduleFolderName, "/") {
	// 	moduleFolderName += "/"
	// }
	//
	// airflowengine.airflowIgnore(airflowengine.airflowHome, moduleFolderName)

	airflowengine.getLibFolder(moduleName, taskId)

	// url := airflowengine.remotePyModuleRepository + "/" + module
	// airflowengine.importPythonModuleFromRemote(url, airflowengine.airflowHome, moduleName)

	airflowengine.importModuleAsFolder(moduleName, taskId)

	return airflowengine.getLibFolderName(moduleName, taskId), nil
}

// functionCode string, taskID string, adminCfg []interface{}, servicePorts []string)
func (airflowengine *AirflowEngine) StartTask(task *ScheduledTaskInstance, brokerURL string, taskCommands []interface{}) (string, string, error) {

	INFO.Printf(
		"TaskInstance => ID: %s | TopologyName: %s | TaskName: %s | ServiceIntentID: %s\n",
		task.ID, task.TopologyName, task.TaskName, task.ServiceIntentID,
	)

	airflowengine.PullImage(task.PythonModule, task.ID)

	jsonBytes, _ := json.Marshal(taskCommands)
	jsonConfiguration := string(jsonBytes)

	// airflowengine.generateDAG(task.PythonModule, task.PythonPackage, airflowengine.generateCommandsList(task, brokerURL))
	airflowengine.generateDAG(task.PythonModule, task.PythonPackage, task.ID, jsonConfiguration)

	//return container.ID, refURL, nil
	return "", "", nil

}

func (airflowengine *AirflowEngine) StopTask(containerID string) {
	//go dockerengine.client.StopContainer(containerID, 1)
}
