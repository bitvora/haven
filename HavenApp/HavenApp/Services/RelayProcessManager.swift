import Foundation
import Combine

class RelayProcessManager: ObservableObject {
    @Published var isRunning = false
    @Published var isBooting = false
    @Published var bootStatusMessage: String = ""
    @Published var isImporting = false
    @Published var importStatusMessage: String = ""
    @Published var importProgress: Double = 0.0
    @Published var isLocked = false
    
    @Published var logs: [LogEntry] = []
    
    // Metrics
    @Published var memoryUsage: Double = 0
    @Published var cpuUsage: Double = 0
    @Published var activeConnections: Int = 0
    @Published var eventsStored: Int = 0
    
    private var process: Process?
    private var outputPipe: Pipe?
    
    private var pendingImportConfig: HavenConfig?
    @Published var startDate: Date?
    
    private var metricsTimer: Timer?
    
    struct LogEntry: Identifiable {
        let id = UUID()
        let timestamp: Date
        let level: String
        let message: String
        
        static func parse(_ line: String) -> LogEntry {
            // Simplified parsing
            let level = line.contains("ERROR") ? "ERROR" :
                       line.contains("WARN") ? "WARN" : "INFO"
            return LogEntry(timestamp: Date(), level: level, message: line)
        }
    }
    
    func startRelay(config: HavenConfig) {
        if let importConfig = pendingImportConfig {
            importNotes(config: importConfig)
            return
        }
        
        guard !isRunning else { return }
        
        let relayDataDir = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("haven_relay")
        try? FileManager.default.createDirectory(at: relayDataDir, withIntermediateDirectories: true)
        
        // Copy templates directory
        if let templatesPath = Bundle.main.path(forResource: "templates", ofType: "") {
             let destURL = relayDataDir.appendingPathComponent("templates")
             // Always update templates on start (clean up old first)
             if FileManager.default.fileExists(atPath: destURL.path) {
                 try? FileManager.default.removeItem(at: destURL)
             }
             try? FileManager.default.copyItem(at: URL(fileURLWithPath: templatesPath), to: destURL)
             logs.append(LogEntry(timestamp: Date(), level: "INFO", message: "Copied templates to \(destURL.path)"))
        } else {
             logs.append(LogEntry(timestamp: Date(), level: "WARN", message: "Templates folder not found in Bundle"))
        }

        // Ensure relay configuration files exist
        let encoder = JSONEncoder()
        encoder.outputFormatting = .prettyPrinted
        
        let importRelaysURL = relayDataDir.appendingPathComponent(config.importSeedRelaysFile)
        if let data = try? encoder.encode(config.importSeedRelays) {
            try? data.write(to: importRelaysURL)
        }
        
        let blastrRelaysURL = relayDataDir.appendingPathComponent(config.blastrRelaysFile)
        if let data = try? encoder.encode(config.blastrRelays) {
            try? data.write(to: blastrRelaysURL)
            logs.append(LogEntry(timestamp: Date(), level: "INFO", message: "Wrote \(config.blastrRelays.count) blastr relays to \(config.blastrRelaysFile)"))
        }

        // Check bundle first
        var executablePath = Bundle.main.path(forResource: "haven", ofType: "")
        
        // Fallback or dev environment
        if executablePath == nil {
             executablePath = "/usr/local/bin/haven" // Fallback
             // For dev, you might want to point to the project dir but sandbox prevents it.
             logs.append(LogEntry(timestamp: Date(), level: "WARN", message: "haven binary not found in bundle, trying /usr/local/bin"))
        }

        guard let executablePath = executablePath, FileManager.default.fileExists(atPath: executablePath) else {
            logs.append(LogEntry(timestamp: Date(), level: "ERROR", message: "haven binary not found at \(executablePath ?? "unknown")"))
            return
        }
        
        logs.append(LogEntry(timestamp: Date(), level: "INFO", message: "Starting relay from: \(executablePath)"))
        
        // Ensure executable permissions
        do {
            let attributes = try FileManager.default.attributesOfItem(atPath: executablePath)
            if let perms = attributes[.posixPermissions] as? Int {
                if perms != 0o755 {
                    try FileManager.default.setAttributes([.posixPermissions: 0o755], ofItemAtPath: executablePath)
                     logs.append(LogEntry(timestamp: Date(), level: "INFO", message: "Fixed permissions for binary"))
                }
            }
        } catch {
             logs.append(LogEntry(timestamp: Date(), level: "WARN", message: "Could not set permissions: \(error)"))
        }
        
        let process = Process()
        process.executableURL = URL(fileURLWithPath: executablePath)
        process.currentDirectoryURL = relayDataDir
        
        // Clean environment
        process.environment = ProcessInfo.processInfo.environment
        
        let pipe = Pipe()
        process.standardOutput = pipe
        process.standardError = pipe
        
        self.outputPipe = pipe
        self.process = process
        
        pipe.fileHandleForReading.readabilityHandler = { [weak self] handle in
            let data = handle.availableData
            if let str = String(data: data, encoding: .utf8), !str.isEmpty {
                DispatchQueue.main.async {
                    self?.processOutput(str)
                }
            }
        }
        
        process.terminationHandler = { [weak self] proc in
            DispatchQueue.main.async {
                guard let self = self else { return }
                self.outputPipe?.fileHandleForReading.readabilityHandler = nil
                self.logs.append(LogEntry(timestamp: Date(), level: "WARN", message: "Relay Process Terminated with code: \(proc.terminationStatus)"))
                self.isRunning = false
                self.isBooting = false
                self.isImporting = false
                if let importConfig = self.pendingImportConfig {
                     self.importNotes(config: importConfig)
                }
            }
        }
        
        process.launch()
        isRunning = true
        isBooting = true
        bootStatusMessage = "Starting system..."
        isLocked = false
        startDate = Date()
        startMetricsTimer()
    }
    
    func clearDatabaseLocks() {
        DispatchQueue.global(qos: .userInitiated).async { [weak self] in
            let relayDataDir = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("haven_relay")
            let dbDir = relayDataDir.appendingPathComponent("db")
            
            // Relays list from init.go: blossom, chat, inbox, outbox, private
            let dbNames = ["blossom", "chat", "inbox", "outbox", "private"]
            
            for name in dbNames {
                let lockFile = dbDir.appendingPathComponent(name).appendingPathComponent("LOCK")
                if FileManager.default.fileExists(atPath: lockFile.path) {
                    try? FileManager.default.removeItem(at: lockFile)
                    let entry = LogEntry(timestamp: Date(), level: "INFO", message: "Deleted lock file: \(lockFile.lastPathComponent) in \(name)")
                    DispatchQueue.main.async {
                        self?.logs.append(entry)
                    }
                }
            }
            
            // Also kill any existing haven processes
            self?.killAllHavenProcesses()
            
            DispatchQueue.main.async {
                self?.isLocked = false
                self?.logs.append(LogEntry(timestamp: Date(), level: "INFO", message: "Database locks cleared. You can now try starting the relay again."))
            }
        }
    }
    
    private func killAllHavenProcesses() {
        let task = Process()
        task.launchPath = "/usr/bin/pkill"
        task.arguments = ["-9", "haven"]
        task.launch()
        task.waitUntilExit()
    }
    
    func stopRelay(completion: (() -> Void)? = nil) {
        guard let process = process, process.isRunning else {
            completion?()
            return
        }
        stopMetricsTimer()
        self.outputPipe?.fileHandleForReading.readabilityHandler = nil
        process.terminate()
        isBooting = false
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
            completion?()
        }
    }
    
    func cancelImport() {
        guard let process = process, process.isRunning else {
            isImporting = false
            return
        }
        process.terminate()
        DispatchQueue.main.async {
            self.isImporting = false
            self.importProgress = 0.0
            self.importStatusMessage = "Import cancelled"
            self.pendingImportConfig = nil
        }
    }
    
    func dismissImport() {
        if let process = process, process.isRunning {
            process.terminate()
        }
        DispatchQueue.main.async {
            self.isImporting = false
            // If we were waiting to restart, do it now
            if let config = self.pendingImportConfig {
                self.pendingImportConfig = nil
                self.startRelay(config: config)
            }
        }
    }
    
    private func startMetricsTimer() {
        metricsTimer?.invalidate()
        metricsTimer = Timer.scheduledTimer(withTimeInterval: 2.0, repeats: true) { [weak self] _ in
            self?.updateMetrics()
        }
        // Update immediately
        updateMetrics()
    }
    
    private func stopMetricsTimer() {
        metricsTimer?.invalidate()
        metricsTimer = nil
    }
    
    private func updateMetrics() {
        // For now, we'll track events through log parsing
        // A proper implementation would require querying the database directly
        // which would need either:
        // 1. A metrics endpoint in the Go backend
        // 2. Direct database access (complex for LMDB/Badger from Swift)
        // 3. Parsing structured output from the relay
        
        // Keep the current log-based counting for now
        // The eventsStored counter is updated in processOutput when we see "stored" messages
    }
    
    private func countEventsInDatabase(at path: URL) -> Int? {
        // Disabled for now - file size is not a reliable indicator
        return nil
    }
    
    func importNotes(config: HavenConfig) {
        if isRunning {
            pendingImportConfig = config
            stopRelay()
            return
        }
        
        // Ensure all required files exist before import
        let relayDataDir = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("haven_relay")
        try? FileManager.default.createDirectory(at: relayDataDir, withIntermediateDirectories: true)
        
        // Create .env and relay JSON files if they don't exist
        let envURL = relayDataDir.appendingPathComponent(".env")
        if !FileManager.default.fileExists(atPath: envURL.path) {
            // We need to generate the .env file - but we don't have access to ConfigService here
            // So we'll create a minimal .env with the config passed in
            let envContent = generateMinimalEnv(config: config)
            try? envContent.write(to: envURL, atomically: true, encoding: .utf8)
        }
        
        // Create seed relays file if it doesn't exist
        let importRelaysURL = relayDataDir.appendingPathComponent(config.importSeedRelaysFile)
        if !FileManager.default.fileExists(atPath: importRelaysURL.path) {
            let encoder = JSONEncoder()
            encoder.outputFormatting = .prettyPrinted
            if let data = try? encoder.encode(config.importSeedRelays) {
                try? data.write(to: importRelaysURL)
            }
        }
        
        // Create blastr relays file if it doesn't exist (required by Go binary)
        let blastrRelaysURL = relayDataDir.appendingPathComponent(config.blastrRelaysFile)
        if !FileManager.default.fileExists(atPath: blastrRelaysURL.path) {
            let encoder = JSONEncoder()
            encoder.outputFormatting = .prettyPrinted
            // Use empty array or config blastr relays
            let blastrRelays = config.blastrRelays.isEmpty ? [] : config.blastrRelays
            if let data = try? encoder.encode(blastrRelays) {
                try? data.write(to: blastrRelaysURL)
            }
        }
        
        let executablePath = Bundle.main.path(forResource: "haven", ofType: "") ?? "/usr/local/bin/haven"
        
        guard FileManager.default.fileExists(atPath: executablePath) else {
            logs.append(LogEntry(timestamp: Date(), level: "ERROR", message: "haven binary not found"))
            return
        }
        
        let process = Process()
        process.executableURL = URL(fileURLWithPath: executablePath)
        process.arguments = ["--import"]
        process.currentDirectoryURL = relayDataDir
        
        let pipe = Pipe()
        process.standardOutput = pipe
        process.standardError = pipe
        
        self.outputPipe = pipe
        self.process = process
        
        // Update @Published properties on main thread
        DispatchQueue.main.async {
            self.importProgress = 0.0
            self.importStatusMessage = "Starting import..."
            self.isImporting = true
        }
        
        pipe.fileHandleForReading.readabilityHandler = { [weak self] handle in
            let data = handle.availableData
            if let str = String(data: data, encoding: .utf8) {
                DispatchQueue.main.async {
                    self?.processOutput(str)
                }
            }
        }
        
        process.terminationHandler = { [weak self] proc in
            DispatchQueue.main.async {
                guard let self = self else { return }
                self.isImporting = false
                let config = self.pendingImportConfig
                self.pendingImportConfig = nil
                
                if proc.terminationStatus == 0 {
                    self.importProgress = 1.0
                    self.importStatusMessage = "Import Complete - Restarting relay..."
                    
                    // Restart the relay after successful import
                    DispatchQueue.main.asyncAfter(deadline: .now() + 1.0) {
                        if let config = config {
                            self.startRelay(config: config)
                        }
                    }
                } else {
                    self.importStatusMessage = "Import Failed (Code \(proc.terminationStatus))"
                }
            }
        }
        
        // Launch the process (non-blocking)
        process.launch()
    }
    
    private func generateMinimalEnv(config: HavenConfig) -> String {
        return """
        OWNER_NPUB="\(config.ownerNpub)"
        RELAY_URL="\(config.relayURL)"
        RELAY_PORT=\(config.relayPort)
        DB_ENGINE="\(config.dbEngine)"
        IMPORT_START_DATE="\(config.importStartDate)"
        IMPORT_SEED_RELAYS_FILE="\(config.importSeedRelaysFile)"
        HAVEN_LOG_LEVEL="\(config.logLevel)"
        """
    }
    
    private func processOutput(_ output: String) {
        let lines = output.components(separatedBy: .newlines).filter { !$0.isEmpty }
        guard !lines.isEmpty else { return }
        
        var newEntries: [LogEntry] = []
        for line in lines {
            let entry = LogEntry.parse(line)
            newEntries.append(entry)
            
            if isImporting {
               if line.contains("connected successfully") {
                   importProgress = 0.1
                   importStatusMessage = "Connected to relays..."
               } else if line.contains("Imported") && line.contains("notes") {
                   if let dateStr = line.components(separatedBy: "to ").last?.prefix(10) {
                        calculateProgress(currentDateStr: String(dateStr))
                   }
                   if let rangeStart = line.range(of: "from ")?.upperBound,
                      let rangeEnd = line.range(of: " to")?.lowerBound {
                       importStatusMessage = "Found notes from \(line[rangeStart..<rangeEnd])..."
                   } else {
                       importStatusMessage = "Found notes..."
                   }
               } else if line.contains("importing inbox notes") || line.contains("Importing inbox notes") {
                   importStatusMessage = "Importing tagged notes..."
                   importProgress = 0.85 // More conservative estimate
               } else if line.contains("imported") && line.contains("tagged notes") {
                   // Tagged notes import completed
                   importProgress = 0.95
                   if let count = line.components(separatedBy: " ").first(where: { Int($0) != nil }) {
                       importStatusMessage = "Imported \(count) tagged notes"
                   } else {
                       importStatusMessage = "Tagged notes imported"
                   }
               } else if line.contains("Import complete") || line.contains("import complete") {
                   importProgress = 1.0
                   importStatusMessage = "Import Complete!"
               } else if line.contains("No notes found") {
                   if let dateStr = line.components(separatedBy: "to ").last?.prefix(10) {
                       calculateProgress(currentDateStr: String(dateStr))
                       if let fromIndex = line.components(separatedBy: "for ").last?.prefix(10) {
                            importStatusMessage = "Checking \(fromIndex)... (No notes found)"
                       }
                   } else {
                        importProgress = min(importProgress + 0.03, 0.85) // Smaller increments, cap at 85%
                   }
               }
            }
            
            // Parse event counts from logs
            if line.contains("Imported") && line.contains("notes") {
                // Extract number from "Imported X notes from..."
                let components = line.components(separatedBy: " ")
                if let importedIndex = components.firstIndex(of: "Imported"),
                   importedIndex + 1 < components.count,
                   let count = Int(components[importedIndex + 1]) {
                    eventsStored += count
                }
            } else if line.contains("imported") && line.contains("tagged notes") {
                // Extract from "imported X tagged notes"
                let components = line.components(separatedBy: " ")
                if let importedIndex = components.firstIndex(of: "imported"),
                   importedIndex + 1 < components.count,
                   let count = Int(components[importedIndex + 1]) {
                    eventsStored += count
                }
            } else if line.contains("new note") || line.contains("new reaction") || 
                      line.contains("new zap") || line.contains("new encrypted message") ||
                      line.contains("new gift-wrapped") || line.contains("new repost") {
                // Individual events coming in
                eventsStored += 1
            }
            
            // Booting status
            if isBooting {
                let lowerLine = line.lowercased()
                
                if lowerLine.contains("subscribing to") {
                    if let topic = line.components(separatedBy: "to ").last {
                        bootStatusMessage = "Subscribing to \(topic.trimmingCharacters(in: .punctuationCharacters))..."
                    }
                } else if lowerLine.contains("starting") {
                    if let service = line.components(separatedBy: "starting ").last ?? line.components(separatedBy: "Starting ").last {
                         bootStatusMessage = "Starting \(service.trimmingCharacters(in: .punctuationCharacters))..."
                    }
                } else if lowerLine.contains("loading") {
                     bootStatusMessage = "Loading databases..."
                } else if lowerLine.contains("listening on") {
                     bootStatusMessage = "Establishing listener..."
                } else if lowerLine.contains("wot") || lowerLine.contains("pubkeys") || lowerLine.contains("analysed") || lowerLine.contains("network size") {
                    let pattern = "(\\d+)" // Just find digits
                    if let regex = try? NSRegularExpression(pattern: pattern, options: []),
                       let _ = regex.firstMatch(in: line, options: [], range: NSRange(line.startIndex..., in: line)) {
                        let matches = regex.matches(in: line, options: [], range: NSRange(line.startIndex..., in: line))
                        if let lastMatch = matches.last, let countRange = Range(lastMatch.range(at: 0), in: line) {
                            let count = line[countRange]
                            if count.count < 10 { // Avoid long hex/hashes
                                if lowerLine.contains("analysed") {
                                    bootStatusMessage = "Analysing \(count) pubkeys..."
                                } else if lowerLine.contains("network size") {
                                    bootStatusMessage = "Network: \(count) profiles..."
                                } else if lowerLine.contains("minimum followers") {
                                    bootStatusMessage = "WoT: \(count) trusted keys..."
                                } else {
                                    bootStatusMessage = "WoT: Loading \(count) keys..."
                                }
                            }
                        }
                    }
                }
            }
            
            // Connection tracking
            if line.contains("subscribing to inbox") || line.contains("Subscribing to inbox") {
                isBooting = false
                bootStatusMessage = ""
            }
            
            if line.contains("accepted connection") || line.contains("new connection") || line.contains("WS connect") {
                activeConnections += 1
            } else if line.contains("connection closed") || line.contains("WS disconnect") || line.contains("disconnected") {
                activeConnections = max(0, activeConnections - 1)
            }
            
            if line.contains("Cannot acquire directory lock") || line.contains("Another process is using this Badger database") {
                isLocked = true
            }
        }
        
        // Batch append logs
        logs.append(contentsOf: newEntries)
        if logs.count > 1000 {
            logs.removeFirst(max(0, logs.count - 1000))
        }
    }
    
    private func calculateProgress(currentDateStr: String) {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        if let currentDate = formatter.date(from: currentDateStr),
           let pending = pendingImportConfig,
           let start = formatter.date(from: pending.importStartDate) {
            let totalInterval = Date().timeIntervalSince(start)
            let currentInterval = currentDate.timeIntervalSince(start)
            if totalInterval > 0 {
                let completion = currentInterval / totalInterval
                let scaled = 0.1 + (completion * 0.8)
                importProgress = min(max(scaled, 0.1), 0.9)
            }
        }
    }
}
