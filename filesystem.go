package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileInfo represents information about a file
type FileInfo struct {
	Path    string `json:"path"`
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"is_dir"`
	ModTime string `json:"mod_time"`
}

// InspectDirectoryRecursive recursively lists all files in a directory
func InspectDirectoryRecursive(path string, maxDepth int) ([]FileInfo, error) {
	var files []FileInfo

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't read
		}

		// Calculate depth
		relPath, _ := filepath.Rel(path, filePath)
		depth := strings.Count(relPath, string(os.PathSeparator))

		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		files = append(files, FileInfo{
			Path:    filePath,
			Name:    info.Name(),
			Size:    info.Size(),
			IsDir:   info.IsDir(),
			ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
		})

		return nil
	})

	return files, err
}

// SearchFiles searches for files matching a pattern
func SearchFiles(rootPath string, pattern string, maxDepth int) ([]FileInfo, error) {
	var matches []FileInfo

	err := filepath.Walk(rootPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		// Calculate depth
		relPath, _ := filepath.Rel(rootPath, filePath)
		depth := strings.Count(relPath, string(os.PathSeparator))

		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if name matches pattern
		matched, _ := filepath.Match(pattern, info.Name())
		if matched || strings.Contains(strings.ToLower(info.Name()), strings.ToLower(pattern)) {
			matches = append(matches, FileInfo{
				Path:    filePath,
				Name:    info.Name(),
				Size:    info.Size(),
				IsDir:   info.IsDir(),
				ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
			})
		}

		return nil
	})

	return matches, err
}

// GetCommonDirectories returns common user directories
func GetCommonDirectories() []string {
	home, _ := os.UserHomeDir()

	return []string{
		filepath.Join(home, "Desktop"),
		filepath.Join(home, "Documents"),
		filepath.Join(home, "Downloads"),
		filepath.Join(home, "Pictures"),
		filepath.Join(home, "Videos"),
		filepath.Join(home, "Music"),
		filepath.Join(home, "OneDrive"),
		filepath.Join(home, "Projects"),
		filepath.Join(home, "Code"),
		filepath.Join(home, "Development"),
	}
}

// FindFilesByExtension finds all files with a specific extension
func FindFilesByExtension(rootPath string, extension string, maxDepth int) ([]FileInfo, error) {
	var matches []FileInfo

	if !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}

	err := filepath.Walk(rootPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		relPath, _ := filepath.Rel(rootPath, filePath)
		depth := strings.Count(relPath, string(os.PathSeparator))

		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), strings.ToLower(extension)) {
			matches = append(matches, FileInfo{
				Path:    filePath,
				Name:    info.Name(),
				Size:    info.Size(),
				IsDir:   info.IsDir(),
				ModTime: info.ModTime().Format("2006-01-02 15:04:05"),
			})
		}

		return nil
	})

	return matches, err
}

// ReadFileContent reads the content of a file (with size limit for safety)
func ReadFileContent(filePath string, maxSize int64) (string, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return "", err
	}

	if info.Size() > maxSize {
		return "", fmt.Errorf("file too large: %d bytes (max: %d bytes)", info.Size(), maxSize)
	}

	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// ListDirectoryTree creates a tree structure of a directory
func ListDirectoryTree(rootPath string, maxDepth int) (string, error) {
	var output strings.Builder

	output.WriteString(rootPath + "\n")

	err := filepath.Walk(rootPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if filePath == rootPath {
			return nil
		}

		relPath, _ := filepath.Rel(rootPath, filePath)
		depth := strings.Count(relPath, string(os.PathSeparator))

		if depth > maxDepth {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Create indentation
		indent := strings.Repeat("  ", depth)

		if info.IsDir() {
			output.WriteString(fmt.Sprintf("%s📁 %s/\n", indent, info.Name()))
		} else {
			size := formatFileSize(info.Size())
			output.WriteString(fmt.Sprintf("%s📄 %s (%s)\n", indent, info.Name(), size))
		}

		return nil
	})

	return output.String(), err
}

func formatFileSize(bytes int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	switch {
	case bytes >= GB:
		return fmt.Sprintf("%.2f GB", float64(bytes)/float64(GB))
	case bytes >= MB:
		return fmt.Sprintf("%.2f MB", float64(bytes)/float64(MB))
	case bytes >= KB:
		return fmt.Sprintf("%.2f KB", float64(bytes)/float64(KB))
	default:
		return fmt.Sprintf("%d B", bytes)
	}
}
