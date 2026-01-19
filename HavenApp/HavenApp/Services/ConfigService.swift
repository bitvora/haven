import Foundation
import ServiceManagement

/// Manages Haven configuration persistence
@MainActor
class ConfigService: ObservableObject {
    @Published var config: HavenConfig
    
    // Config stored in App Support (standard macOS location for app preferences/state)
    private let configURL: URL
    // Relay data stored in separate directory to avoid conflicts with source code
    private let relayDataDir: URL
    
    init() {
        // Store config in Application Support
        let appSupport = FileManager.default.urls(for: .applicationSupportDirectory, in: .userDomainMask).first!
        let havenAppSupport = appSupport.appendingPathComponent("Haven", isDirectory: true)
        
        // Create config directory if needed
        try? FileManager.default.createDirectory(at: havenAppSupport, withIntermediateDirectories: true)
        
        configURL = havenAppSupport.appendingPathComponent("config.json")
        
        // Use ~/haven_relay for data to avoid destroying the ~/haven project folder
        relayDataDir = FileManager.default.homeDirectoryForCurrentUser.appendingPathComponent("haven_relay")
        
        // Load existing config or create default
        if let data = try? Data(contentsOf: configURL),
           let loaded = try? JSONDecoder().decode(HavenConfig.self, from: data) {
            config = loaded
            
            // Ensure defaults are applied for empty arrays
            if config.importSeedRelays.isEmpty {
                config.importSeedRelays = HavenConfig.default.importSeedRelays
            }
            if config.blastrRelays.isEmpty {
                config.blastrRelays = HavenConfig.default.blastrRelays
            }
        } else {
            config = HavenConfig.default
        }
        
        loadRelayLists()
    }
    
    private func loadRelayLists() {
        // Use separate data dir
        let importURL = relayDataDir.appendingPathComponent(config.importSeedRelaysFile)
        if let data = try? Data(contentsOf: importURL),
           let list = try? JSONDecoder().decode([String].self, from: data),
           !list.isEmpty {
            config.importSeedRelays = list
        }
        // If file doesn't exist or is empty, keep the defaults from HavenConfig
        
        let blastrURL = relayDataDir.appendingPathComponent(config.blastrRelaysFile)
        if let data = try? Data(contentsOf: blastrURL),
           let list = try? JSONDecoder().decode([String].self, from: data),
           !list.isEmpty {
            config.blastrRelays = list
        }
        // If file doesn't exist or is empty, keep the defaults from HavenConfig
    }
    
    func save() {
        do {
            let data = try JSONEncoder().encode(config)
            try data.write(to: configURL)
            
            saveRelayLists()
            
            // Update launch at login if changed
            updateLaunchAtLogin()
        } catch {
            print("Failed to save config: \(error)")
        }
    }
    
    private func saveRelayLists() {
        // Ensure relayURL includes the port if it's localhost or empty
        let trimmedURL = config.relayURL.trimmingCharacters(in: .whitespacesAndNewlines)
        if trimmedURL.isEmpty || trimmedURL == "localhost" || trimmedURL == "127.0.0.1" {
            config.relayURL = "localhost:\(config.relayPort)"
        }
        
        // Ensure data dir exists
        try? FileManager.default.createDirectory(at: relayDataDir, withIntermediateDirectories: true)
        
        let encoder = JSONEncoder()
        encoder.outputFormatting = .prettyPrinted
        if let data = try? encoder.encode(config) {
            try? data.write(to: configURL) // Save main config again just in case
        }
        
        if !config.importSeedRelays.isEmpty {
            let importURL = relayDataDir.appendingPathComponent(config.importSeedRelaysFile)
            if let data = try? encoder.encode(config.importSeedRelays) {
                try? data.write(to: importURL)
            }
        }
        
        if !config.blastrRelays.isEmpty {
            let blastrURL = relayDataDir.appendingPathComponent(config.blastrRelaysFile)
            if let data = try? encoder.encode(config.blastrRelays) {
                try? data.write(to: blastrURL)
            }
        }
        
        // Also update .env just in case important paths changed
        let envContent = generateEnvFile()
        let envURL = relayDataDir.appendingPathComponent(".env")
        try? envContent.write(to: envURL, atomically: true, encoding: String.Encoding.utf8)
    }
    
    private func updateLaunchAtLogin() {
        if #available(macOS 13.0, *) {
            do {
                if config.launchAtLogin {
                    try SMAppService.mainApp.register()
                } else {
                    try SMAppService.mainApp.unregister()
                }
            } catch {
                // Expected to fail in development without proper code signing
            }
        }
    }
    
    /// Create the required files for Haven to run (.env, relay JSON files)
    func createRequiredFiles() {
        // Create relay data directory if needed
        try? FileManager.default.createDirectory(at: relayDataDir, withIntermediateDirectories: true)
        
        // Create .env file
        let envContent = generateEnvFile()
        let envURL = relayDataDir.appendingPathComponent(".env")
        try? envContent.write(to: envURL, atomically: true, encoding: .utf8)
        
        // Create relays_import.json
        let importRelays = """
        [
            "wss://relay.damus.io",
            "wss://relay.primal.net",
            "wss://nos.lol",
            "wss://nostr.wine"
        ]
        """
        let importURL = relayDataDir.appendingPathComponent("relays_import.json")
        try? importRelays.write(to: importURL, atomically: true, encoding: .utf8)
        
        // Create relays_blastr.json
        let blastrRelays = """
        [
            "wss://relay.damus.io",
            "wss://relay.primal.net",
            "wss://nos.lol",
            "wss://nostr.wine"
        ]
        """
        let blastrURL = relayDataDir.appendingPathComponent("relays_blastr.json")
        try? blastrRelays.write(to: blastrURL, atomically: true, encoding: .utf8)
        
        // Create blossom directory
        let blossomDir = relayDataDir.appendingPathComponent("blossom")
        try? FileManager.default.createDirectory(at: blossomDir, withIntermediateDirectories: true)
        
        print("Created Haven config files at: \(relayDataDir.path)")
    }
    
    /// Perform a factory reset: delete data and config using FileManager
    func resetApp() {
        let fileManager = FileManager.default
        
        // 1. Delete relay data directory (contains DB, logs, .env)
        // Checks to ensure we aren't deleting root or home by accident
        if relayDataDir.path.count > 10 && fileManager.fileExists(atPath: relayDataDir.path) {
            try? fileManager.removeItem(at: relayDataDir)
        }
        
        // 2. Delete config.json in Application Support
        if fileManager.fileExists(atPath: configURL.path) {
            try? fileManager.removeItem(at: configURL)
        }
        
        // 3. Reset in-memory config
        config = HavenConfig.default
    }
    
    private func generateEnvFile() -> String {
        return """
        OWNER_NPUB="\(config.ownerNpub)"
        RELAY_URL="\(config.relayURL)"
        RELAY_PORT=\(config.relayPort)
        RELAY_BIND_ADDRESS="0.0.0.0"
        DB_ENGINE="\(config.dbEngine)"
        LMDB_MAPSIZE=0
        BLOSSOM_PATH="blossom/"
        
        ## Private Relay Settings
        PRIVATE_RELAY_NAME="\(config.privateRelayName)"
        PRIVATE_RELAY_NPUB="\(config.ownerNpub)"
        PRIVATE_RELAY_DESCRIPTION="\(config.privateRelayDescription)"
        PRIVATE_RELAY_ICON="\(config.privateRelayIcon)"
        PRIVATE_RELAY_EVENT_IP_LIMITER_TOKENS_PER_INTERVAL=50
        PRIVATE_RELAY_EVENT_IP_LIMITER_INTERVAL=1
        PRIVATE_RELAY_EVENT_IP_LIMITER_MAX_TOKENS=100
        PRIVATE_RELAY_ALLOW_EMPTY_FILTERS=true
        PRIVATE_RELAY_ALLOW_COMPLEX_FILTERS=true
        PRIVATE_RELAY_CONNECTION_RATE_LIMITER_TOKENS_PER_INTERVAL=3
        PRIVATE_RELAY_CONNECTION_RATE_LIMITER_INTERVAL=5
        PRIVATE_RELAY_CONNECTION_RATE_LIMITER_MAX_TOKENS=9
        
        ## Chat Relay Settings
        CHAT_RELAY_NAME="\(config.chatRelayName)"
        CHAT_RELAY_NPUB="\(config.ownerNpub)"
        CHAT_RELAY_DESCRIPTION="\(config.chatRelayDescription)"
        CHAT_RELAY_ICON="\(config.chatRelayIcon)"
        CHAT_RELAY_WOT_DEPTH=\(config.chatRelayWotDepth)
        CHAT_RELAY_WOT_REFRESH_INTERVAL_HOURS=\(config.chatRelayWotRefreshHours)
        CHAT_RELAY_MINIMUM_FOLLOWERS=\(config.chatRelayMinFollowers)
        CHAT_RELAY_EVENT_IP_LIMITER_TOKENS_PER_INTERVAL=50
        CHAT_RELAY_EVENT_IP_LIMITER_INTERVAL=1
        CHAT_RELAY_EVENT_IP_LIMITER_MAX_TOKENS=100
        CHAT_RELAY_ALLOW_EMPTY_FILTERS=false
        CHAT_RELAY_ALLOW_COMPLEX_FILTERS=false
        CHAT_RELAY_CONNECTION_RATE_LIMITER_TOKENS_PER_INTERVAL=3
        CHAT_RELAY_CONNECTION_RATE_LIMITER_INTERVAL=3
        CHAT_RELAY_CONNECTION_RATE_LIMITER_MAX_TOKENS=9
        
        ## Outbox Relay Settings
        OUTBOX_RELAY_NAME="\(config.outboxRelayName)"
        OUTBOX_RELAY_NPUB="\(config.ownerNpub)"
        OUTBOX_RELAY_DESCRIPTION="\(config.outboxRelayDescription)"
        OUTBOX_RELAY_ICON="\(config.outboxRelayIcon)"
        OUTBOX_RELAY_EVENT_IP_LIMITER_TOKENS_PER_INTERVAL=10
        OUTBOX_RELAY_EVENT_IP_LIMITER_INTERVAL=60
        OUTBOX_RELAY_EVENT_IP_LIMITER_MAX_TOKENS=100
        OUTBOX_RELAY_ALLOW_EMPTY_FILTERS=false
        OUTBOX_RELAY_ALLOW_COMPLEX_FILTERS=false
        OUTBOX_RELAY_CONNECTION_RATE_LIMITER_TOKENS_PER_INTERVAL=3
        OUTBOX_RELAY_CONNECTION_RATE_LIMITER_INTERVAL=1
        OUTBOX_RELAY_CONNECTION_RATE_LIMITER_MAX_TOKENS=9
        
        ## Inbox Relay Settings
        INBOX_RELAY_NAME="\(config.inboxRelayName)"
        INBOX_RELAY_NPUB="\(config.ownerNpub)"
        INBOX_RELAY_DESCRIPTION="\(config.inboxRelayDescription)"
        INBOX_RELAY_ICON="\(config.inboxRelayIcon)"
        INBOX_PULL_INTERVAL_SECONDS=\(config.inboxPullIntervalSeconds)
        INBOX_RELAY_EVENT_IP_LIMITER_TOKENS_PER_INTERVAL=10
        INBOX_RELAY_EVENT_IP_LIMITER_INTERVAL=1
        INBOX_RELAY_EVENT_IP_LIMITER_MAX_TOKENS=20
        INBOX_RELAY_ALLOW_EMPTY_FILTERS=false
        INBOX_RELAY_ALLOW_COMPLEX_FILTERS=false
        INBOX_RELAY_CONNECTION_RATE_LIMITER_TOKENS_PER_INTERVAL=3
        INBOX_RELAY_CONNECTION_RATE_LIMITER_INTERVAL=1
        INBOX_RELAY_CONNECTION_RATE_LIMITER_MAX_TOKENS=9
        
        ## Import Settings
        IMPORT_START_DATE="\(config.importStartDate)"
        IMPORT_QUERY_INTERVAL_SECONDS=600
        IMPORT_OWNER_NOTES_FETCH_TIMEOUT_SECONDS=60
        IMPORT_TAGGED_NOTES_FETCH_TIMEOUT_SECONDS=120
        IMPORT_SEED_RELAYS_FILE="relays_import.json"
        
        ## Backup Settings
        BACKUP_PROVIDER="\(config.backupProvider)"
        BACKUP_INTERVAL_HOURS=\(config.backupIntervalHours)
        
        ## Blastr Settings
        BLASTR_RELAYS_FILE="relays_blastr.json"
        
        ## WOT Settings
        WOT_FETCH_TIMEOUT_SECONDS=60
        
        ## LOGGING
        HAVEN_LOG_LEVEL="\(config.logLevel)"
        
        ## Cloud Backup Settings
        AWS_ACCESS_KEY_ID="\(config.awsAccessKeyId)"
        AWS_SECRET_ACCESS_KEY="\(config.awsSecretAccessKey)"
        AWS_REGION="\(config.awsRegion)"
        AWS_BUCKET="\(config.awsBucket)"
        
        S3_ACCESS_KEY_ID="\(config.s3AccessKeyId)"
        S3_SECRET_KEY="\(config.s3SecretKey)"
        S3_ENDPOINT="\(config.s3Endpoint)"
        S3_BUCKET_NAME="\(config.s3BucketName)"
        S3_REGION="\(config.s3Region)"
        
        GCP_BUCKET_NAME="\(config.gcpBucketName)"
        """
    }
}
