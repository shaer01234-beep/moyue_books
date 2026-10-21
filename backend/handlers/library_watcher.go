package handlers

import (
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"my-books/backend/config"
	"my-books/backend/models"

	"github.com/fsnotify/fsnotify"
)

// libraryWatchManager 用 fsnotify 监听各媒体库目录：
// 当目录（含子目录）内的文件发生 create / write / remove / rename 时，
// 经过防抖（默认 10s 的安静期）后对该媒体库触发一次扫描（runLibraryScan），
// 从而替代原来的「定时轮询全量扫描」。
type libraryWatchManager struct {
	fsw          *fsnotify.Watcher
	mu           sync.Mutex
	libPaths     map[uint]map[string]bool // libID -> 已监听的路径集合
	pathLib      map[string]uint          // 监听路径 -> libID
	libRecursive map[uint]bool            // libID -> 是否递归
	timers       map[uint]*time.Timer     // libID -> 防抖定时器
	stop         chan struct{}
}

// libWatch 全局单例，供 CRUD 在媒体库增删改时动态维护监听。
var libWatch *libraryWatchManager

// watchDebounce 文件变动后的安静期：期间若持续有变动则不断顺延，
// 直到连续 watchDebounce 内无新事件才真正扫描，避免批量操作时反复扫描。
const watchDebounce = 10 * time.Second

// StartLibraryWatcher 启动文件夹监听（替代原来的定时轮询扫描）。
// 仅对 auto_scan = true 的媒体库生效；recursive 时递归监听子目录。
// 若 fsnotify 初始化失败（如某些网络文件系统不支持），自动降级为原来的定时轮询。
func StartLibraryWatcher() {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		log.Printf("media library watcher disabled (fsnotify init failed: %v); folder watching unavailable", err)
		return
	}
	libWatch = &libraryWatchManager{
		fsw:          fsw,
		libPaths:     make(map[uint]map[string]bool),
		pathLib:      make(map[string]uint),
		libRecursive: make(map[uint]bool),
		timers:       make(map[uint]*time.Timer),
		stop:         make(chan struct{}),
	}
	var libs []models.Library
	config.DB.Where("auto_scan = ?", true).Find(&libs)
	for i := range libs {
		libWatch.addLibrary(&libs[i])
	}
	log.Printf("media library watcher started, watching %d libraries", len(libs))
	go libWatch.run()
}

// StopLibraryWatcher 优雅停止监听（进程退出时可选调用）。
func StopLibraryWatcher() {
	if libWatch == nil {
		return
	}
	close(libWatch.stop)
	_ = libWatch.fsw.Close()
}

// ===================== 对外接口（供 CRUD 调用） =====================

// WatchLibrary 媒体库创建 / 启用自动扫描时调用，开始监听其目录。
func WatchLibrary(lib *models.Library) {
	if libWatch == nil || !lib.AutoScan {
		return
	}
	libWatch.addLibrary(lib)
}

// UnwatchLibrary 媒体库删除 / 关闭自动扫描时调用，停止监听。
func UnwatchLibrary(libID uint) {
	if libWatch == nil {
		return
	}
	libWatch.removeLibrary(libID)
}

// ===================== 监听管理 =====================

// addLibrary 注册一个媒体库的所有监听目录（调用方无需持锁）。
func (m *libraryWatchManager) addLibrary(lib *models.Library) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.libPaths[lib.ID]; !ok {
		m.libPaths[lib.ID] = make(map[string]bool)
	}
	m.libRecursive[lib.ID] = lib.Recursive
	m.watchLocked(lib.Path, lib.ID, lib.Recursive)
}

// removeLibrary 停止某媒体库的全部监听（调用方无需持锁）。
func (m *libraryWatchManager) removeLibrary(libID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if paths, ok := m.libPaths[libID]; ok {
		for p := range paths {
			_ = m.fsw.Remove(p)
			delete(m.pathLib, p)
		}
		delete(m.libPaths, libID)
	}
	if t, ok := m.timers[libID]; ok {
		t.Stop()
		delete(m.timers, libID)
	}
	delete(m.libRecursive, libID)
}

// watchLocked 在指定目录上添加 fsnotify 监听（调用方必须持有 m.mu）。
// recursive 为 true 时递归向下监听各级子目录（跳过点开头的隐藏目录）。
func (m *libraryWatchManager) watchLocked(dir string, libID uint, recursive bool) {
	if m.libPaths[libID][dir] {
		return
	}
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		return
	}
	if err := m.fsw.Add(dir); err != nil {
		log.Printf("watch add failed %s: %v", dir, err)
		return
	}
	m.libPaths[libID][dir] = true
	m.pathLib[dir] = libID
	if recursive {
		entries, err := os.ReadDir(dir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() && !strings.HasPrefix(e.Name(), ".") {
					m.watchLocked(filepath.Join(dir, e.Name()), libID, recursive)
				}
			}
		}
	}
}

// libOfDirLocked 由事件路径向上查找已注册的监听目录，返回对应媒体库 ID（调用方必须持有 m.mu）。
func (m *libraryWatchManager) libOfDirLocked(p string) uint {
	dir := filepath.Dir(p)
	for {
		if libID, ok := m.pathLib[dir]; ok {
			return libID
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return 0
}

// ===================== 事件循环 =====================

func (m *libraryWatchManager) run() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-m.stop:
			return
		case ev, ok := <-m.fsw.Events:
			if !ok {
				return
			}
			m.handleEvent(ev)
		case err, ok := <-m.fsw.Errors:
			if !ok {
				return
			}
			log.Printf("media library watcher error: %v", err)
		case <-ticker.C:
			// 周期对账：补加新建/恢复目录的监听，清理已禁用/删除的库。
			m.reconcile()
		}
	}
}

func (m *libraryWatchManager) handleEvent(ev fsnotify.Event) {
	// 忽略纯权限变更（如 chmod），避免无意义扫描。
	if ev.Op == fsnotify.Chmod {
		return
	}
	m.mu.Lock()
	libID := m.pathLib[ev.Name]
	if libID == 0 {
		libID = m.libOfDirLocked(ev.Name)
	}
	// 新增子目录：递归纳入监听（仅当该媒体库为 recursive）。
	if libID != 0 && ev.Op&fsnotify.Create != 0 {
		if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
			m.watchLocked(ev.Name, libID, m.libRecursive[libID])
		}
	}
	m.mu.Unlock()
	if libID == 0 {
		return
	}
	m.scheduleScan(libID)
}

// scheduleScan 以 libID 为单位做防抖：连续变动只会在安静期结束后扫描一次。
func (m *libraryWatchManager) scheduleScan(libID uint) {
	m.mu.Lock()
	if t, ok := m.timers[libID]; ok {
		t.Stop()
	}
	m.timers[libID] = time.AfterFunc(watchDebounce, func() {
		m.mu.Lock()
		delete(m.timers, libID)
		m.mu.Unlock()
		m.scanLibrary(libID)
	})
	m.mu.Unlock()
}

// scanLibrary 防抖到期后的实际扫描，复用手动扫描同一套并发控制与去重逻辑。
func (m *libraryWatchManager) scanLibrary(libID uint) {
	if _, loaded := libraryScanning.LoadOrStore(libID, true); loaded {
		return // 正在扫描，跳过
	}
	defer libraryScanning.Delete(libID)

	var lib models.Library
	if err := config.DB.First(&lib, libID).Error; err != nil {
		return
	}
	if !lib.AutoScan {
		return
	}
	config.DB.Model(&lib).Update("status", "scanning")
	if _, err := runLibraryScan(&lib, nil); err != nil {
		config.DB.Model(&lib).Updates(map[string]interface{}{"status": "error", "last_result": err.Error()})
		return
	}
}

// reconcile 周期对账：保证 auto_scan 库始终被监听，已禁用/删除的库被清理。
func (m *libraryWatchManager) reconcile() {
	var libs []models.Library
	config.DB.Where("auto_scan = ?", true).Find(&libs)
	want := make(map[uint]bool, len(libs))
	for i := range libs {
		want[libs[i].ID] = true
	}
	// 停止已不在监听范围内的库
	m.mu.Lock()
	ids := make([]uint, 0, len(m.libPaths))
	for id := range m.libPaths {
		ids = append(ids, id)
	}
	m.mu.Unlock()
	for _, id := range ids {
		if !want[id] {
			m.removeLibrary(id)
		}
	}
	// 补加缺失的监听（新建库、目录重建、首次监听失败等）
	for i := range libs {
		m.mu.Lock()
		_, watched := m.libPaths[libs[i].ID]
		m.mu.Unlock()
		if !watched {
			m.addLibrary(&libs[i])
		}
	}
}
