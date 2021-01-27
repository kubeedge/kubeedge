package watch

import (
	"github.com/fsnotify/fsnotify"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// FileWatcher is an interface we use to watch changes in files
type FileWatcher interface {
	Close() error
}

// OSFileWatcher defines a watch over a file
type OSFileWatcher struct {
	file    string
	watcher *fsnotify.Watcher
	// onEvent callback to be invoked after the file being watched changes
	onEvent func()
}

// NewFileWatcher creates a new FileWatcher
func NewFileWatcher(file string, onEvent func()) (FileWatcher, error) {
	fw := OSFileWatcher{
		file:    file,
		onEvent: onEvent,
	}
	err := fw.watch()
	return fw, err
}

// Close ends the watch
func (f OSFileWatcher) Close() error {
	return f.watcher.Close()
}

// watch creates a fsnotify watcher for a file and create of write events
func (f *OSFileWatcher) watch() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	f.watcher = watcher

	realFile, err := filepath.EvalSymlinks(f.file)
	if err != nil {
		return err
	}

	dir, file := path.Split(f.file)
	go func(file string) {
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Create == fsnotify.Create ||
					event.Op&fsnotify.Write == fsnotify.Write {
					if finfo, err := os.Lstat(event.Name); err != nil {
						log.Printf("can not lstat file: %v\n", err)
					} else if finfo.Mode()&os.ModeSymlink != 0 {
						if currentRealFile, err := filepath.EvalSymlinks(f.file); err == nil &&
							currentRealFile != realFile {
							f.onEvent()
							realFile = currentRealFile
						}
						continue
					}
					if strings.HasSuffix(event.Name, file) {
						f.onEvent()
					}
				}
			case err := <-watcher.Errors:
				if err != nil {
					log.Printf("error watching file: %v\n", err)
				}
			}
		}
	}(file)
	return watcher.Add(dir)
}
