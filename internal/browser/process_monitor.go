package browser

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/rennaisance-jomt/axon/pkg/logger"
)

// ProcessMonitor helps detect and clean up orphaned Chromium processes
type ProcessMonitor struct {
	monitoredProcesses []string
}

// NewProcessMonitor creates a new process monitor
func NewProcessMonitor() *ProcessMonitor {
	return &ProcessMonitor{
		monitoredProcesses: []string{"chrome", "chromium", "chromium-browser"},
	}
}

// GetChromiumProcesses returns a list of running Chromium processes
func (pm *ProcessMonitor) GetChromiumProcesses() ([]ProcessInfo, error) {
	var processes []ProcessInfo
	
	switch runtime.GOOS {
	case "windows":
		processes = pm.getWindowsProcesses()
	case "darwin":
		processes = pm.getDarwinProcesses()
	case "linux":
		processes = pm.getLinuxProcesses()
	default:
		return nil, fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	
	return processes, nil
}

// getWindowsProcesses gets Chromium processes on Windows
func (pm *ProcessMonitor) getWindowsProcesses() []ProcessInfo {
	cmd := exec.Command("tasklist", "/fo", "csv", "/nh")
	output, err := cmd.Output()
	if err != nil {
		logger.Warn("Failed to get Windows process list: %v", err)
		return nil
	}
	
	var processes []ProcessInfo
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		// Parse CSV format: "Image Name","PID","Session Name","Session#","Mem Usage"
		parts := strings.Split(line, ",")
		if len(parts) >= 2 {
			imageName := strings.Trim(parts[0], "\"")
			pid := strings.Trim(parts[1], "\"")
			
			for _, monitored := range pm.monitoredProcesses {
				if strings.Contains(strings.ToLower(imageName), monitored) {
					processes = append(processes, ProcessInfo{
						Name: imageName,
						PID:  pid,
						OS:   "windows",
					})
					break
				}
			}
		}
	}
	
	return processes
}

// getDarwinProcesses gets Chromium processes on macOS
func (pm *ProcessMonitor) getDarwinProcesses() []ProcessInfo {
	cmd := exec.Command("ps", "-eo", "pid,command")
	output, err := cmd.Output()
	if err != nil {
		logger.Warn("Failed to get macOS process list: %v", err)
		return nil
	}
	
	var processes []ProcessInfo
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			pid := parts[0]
			command := strings.Join(parts[1:], " ")
			
			for _, monitored := range pm.monitoredProcesses {
				if strings.Contains(strings.ToLower(command), monitored) {
					processes = append(processes, ProcessInfo{
						Name: command,
						PID:  pid,
						OS:   "darwin",
					})
					break
				}
			}
		}
	}
	
	return processes
}

// getLinuxProcesses gets Chromium processes on Linux
func (pm *ProcessMonitor) getLinuxProcesses() []ProcessInfo {
	cmd := exec.Command("ps", "-eo", "pid,command")
	output, err := cmd.Output()
	if err != nil {
		logger.Warn("Failed to get Linux process list: %v", err)
		return nil
	}
	
	var processes []ProcessInfo
	lines := strings.Split(string(output), "\n")
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			pid := parts[0]
			command := strings.Join(parts[1:], " ")
			
			for _, monitored := range pm.monitoredProcesses {
				if strings.Contains(strings.ToLower(command), monitored) {
					processes = append(processes, ProcessInfo{
						Name: command,
						PID:  pid,
						OS:   "linux",
					})
					break
				}
			}
		}
	}
	
	return processes
}

// KillProcess kills a process by PID
func (pm *ProcessMonitor) KillProcess(pid string) error {
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("taskkill", "/F", "/PID", pid)
	case "darwin", "linux":
		cmd = exec.Command("kill", "-9", pid)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to kill process %s: %w", pid, err)
	}
	
	logger.Info("Successfully killed process %s", pid)
	return nil
}

// CleanupOrphanedProcesses kills any Chromium processes that shouldn't be running
func (pm *ProcessMonitor) CleanupOrphanedProcesses() error {
	processes, err := pm.GetChromiumProcesses()
	if err != nil {
		return err
	}
	
	if len(processes) == 0 {
		logger.Info("No Chromium processes found")
		return nil
	}
	
	logger.Warn("Found %d Chromium processes that may be orphaned:", len(processes))
	for _, proc := range processes {
		logger.Warn("  - %s (PID: %s)", proc.Name, proc.PID)
	}
	
	// Wait a shorter moment to see if they clean up themselves
	logger.Info("Waiting 1 second to see if processes clean up naturally...")
	time.Sleep(1 * time.Second)
	
	// Check again
	processes, err = pm.GetChromiumProcesses()
	if err != nil {
		return err
	}
	
	if len(processes) == 0 {
		logger.Info("Processes cleaned up naturally")
		return nil
	}
	
	logger.Warn("Force killing %d orphaned Chromium processes...", len(processes))
	killCount := 0
	for _, proc := range processes {
		// Use a more aggressive approach to ensure process termination
		if err := pm.ForceKillProcess(proc.PID); err != nil {
			logger.Error("Failed to kill process %s: %v", proc.PID, err)
		} else {
			killCount++
		}
	}
	
	// Verify processes were actually terminated
	time.Sleep(500 * time.Millisecond)
	remainingProcesses, _ := pm.GetChromiumProcesses()
	if len(remainingProcesses) > 0 {
		logger.Warn("%d Chrome processes still remain after cleanup attempt", len(remainingProcesses))
		
		// Use system-specific cleanup for really stubborn processes
		if killCount == 0 && len(remainingProcesses) > 0 {
			logger.Warn("Using system-specific cleanup methods for stubborn processes...")
			pm.systemSpecificCleanup()
		}
	} else {
		logger.Success("All Chrome processes successfully terminated")
	}
	
	return nil
}

// ForceKillProcess uses stronger methods to ensure a process is killed
func (pm *ProcessMonitor) ForceKillProcess(pid string) error {
	var cmd *exec.Cmd
	
	switch runtime.GOOS {
	case "windows":
		// Use /F flag to forcefully terminate
		cmd = exec.Command("taskkill", "/F", "/T", "/PID", pid)
	case "darwin", "linux":
		// Use SIGKILL which cannot be caught or ignored
		cmd = exec.Command("kill", "-9", pid)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to force-kill process %s: %w", pid, err)
	}
	
	logger.Success("Successfully killed process %s", pid)
	return nil
}

// systemSpecificCleanup uses more drastic platform-specific methods
func (pm *ProcessMonitor) systemSpecificCleanup() {
	switch runtime.GOOS {
	case "windows":
		// On Windows, try a multi-pronged approach
		cmd1 := exec.Command("taskkill", "/F", "/IM", "chrome.exe", "/T")
		cmd1.Run() // Ignore errors
		cmd2 := exec.Command("taskkill", "/F", "/IM", "chromium.exe", "/T")
		cmd2.Run() // Ignore errors
		cmd3 := exec.Command("wmic", "process", "where", "name like '%chrome%'", "delete")
		cmd3.Run() // Ignore errors
	case "darwin":
		// On macOS, try the pkill command
		cmd := exec.Command("pkill", "-9", "Chrome")
		cmd.Run() // Ignore errors
	case "linux":
		// On Linux, try pkill with wildcard
		cmd := exec.Command("pkill", "-9", "-f", "chrom")
		cmd.Run() // Ignore errors
	}
}

// ProcessInfo represents information about a running process
type ProcessInfo struct {
	Name string
	PID  string
	OS   string
}

// MonitorChromiumCleanup monitors for orphaned Chromium processes after shutdown
func MonitorChromiumCleanup() {
	monitor := NewProcessMonitor()
	
	// Wait a moment after shutdown
	time.Sleep(1 * time.Second)
	
	// First attempt
	if err := monitor.CleanupOrphanedProcesses(); err != nil {
		logger.Error("Failed to cleanup orphaned processes: %v", err)
	}
	
	// Check if there are still processes remaining and try again with more aggressive options
	remainingProcesses, _ := monitor.GetChromiumProcesses()
	if len(remainingProcesses) > 0 {
		logger.Warn("Chromium processes still remain after first cleanup, trying again with more force...")
		time.Sleep(500 * time.Millisecond)
		
		// Second attempt with system specific commands
		monitor.systemSpecificCleanup()
	}
}
