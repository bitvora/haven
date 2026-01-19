import SwiftUI

extension NumberFormatter {
    static var noSeparator: NumberFormatter {
        let formatter = NumberFormatter()
        formatter.usesGroupingSeparator = false
        return formatter
    }
}

struct SettingsView: View {
    @EnvironmentObject var configService: ConfigService
    @EnvironmentObject var relayManager: RelayProcessManager
    @State private var selectedTab: SettingsTab = .identity
    
    enum SettingsTab: String, CaseIterable {
        case identity = "Identity"
        case relays = "Relays"
        case importNotes = "Import"
        case blastr = "Blastr"
        case advanced = "Advanced"
        case backup = "Backup"
        case logs = "Logs"
    }
    
    var body: some View {
        TabView(selection: $selectedTab) {
            IdentitySettingsView()
                .tabItem { Label("Identity", systemImage: "person") }
                .tag(SettingsTab.identity)
            
            RelaySettingsView()
                .tabItem { Label("Relays", systemImage: "server.rack") }
                .tag(SettingsTab.relays)
            
            ImportSettingsView()
                .tabItem { Label("Import", systemImage: "square.and.arrow.down") }
                .tag(SettingsTab.importNotes)
            
            BlastrSettingsView()
                .tabItem { Label("Blastr", systemImage: "paperplane") }
                .tag(SettingsTab.blastr)
            
            AdvancedSettingsView()
                .tabItem { Label("Advanced", systemImage: "gearshape.2") }
                .tag(SettingsTab.advanced)
            
            BackupSettingsView()
                .tabItem { Label("Backup", systemImage: "icloud") }
                .tag(SettingsTab.backup)
            
            LogsView()
                .tabItem { Label("Logs", systemImage: "list.bullet.rectangle") }
                .tag(SettingsTab.logs)
        }
        .environmentObject(configService)
        .environmentObject(relayManager)
        .frame(width: 600, height: 500)
    }
}

struct IdentitySettingsView: View {
    @EnvironmentObject var configService: ConfigService
    
    var body: some View {
        Form {
            Section("Owner Identity") {
                TextField("Owner npub", text: $configService.config.ownerNpub)
                    .help("Your Nostr public key in npub format")
            }
            
            Section("Connection") {
                TextField("Hostname", text: $configService.config.relayURL)
                    .help("Public hostname for your relay (e.g., relay.example.com). Do not include the port.")
                
                TextField("Port", value: $configService.config.relayPort, formatter: NumberFormatter.noSeparator)
            }
        }
        .formStyle(.grouped)
        .padding()
    }
}

struct RelaySettingsView: View {
    @EnvironmentObject var configService: ConfigService
    @State private var selectedRelay: RelayType = .outbox
    
    enum RelayType: String, CaseIterable {
        case outbox = "Outbox"
        case inbox = "Inbox"
        case privateRelay = "Private"
        case chat = "Chat"
    }
    
    var body: some View {
        HStack(spacing: 0) {
            // Sidebar
            List(RelayType.allCases, id: \.self, selection: $selectedRelay) { relay in
                Label(relay.rawValue, systemImage: iconFor(relay))
            }
            .frame(width: 120)
            .listStyle(.sidebar)
            
            Divider()
            
            // Detail
            Form {
                switch selectedRelay {
                case .outbox:
                    RelayConfigForm(
                        name: $configService.config.outboxRelayName,
                        description: $configService.config.outboxRelayDescription,
                        icon: $configService.config.outboxRelayIcon
                    )
                case .inbox:
                    RelayConfigForm(
                        name: $configService.config.inboxRelayName,
                        description: $configService.config.inboxRelayDescription,
                        icon: $configService.config.inboxRelayIcon
                    )
                case .privateRelay:
                    RelayConfigForm(
                        name: $configService.config.privateRelayName,
                        description: $configService.config.privateRelayDescription,
                        icon: $configService.config.privateRelayIcon
                    )
                case .chat:
                    RelayConfigForm(
                        name: $configService.config.chatRelayName,
                        description: $configService.config.chatRelayDescription,
                        icon: $configService.config.chatRelayIcon
                    )
                    
                    Section("Web of Trust") {
                        Stepper("WoT Depth: \(configService.config.chatRelayWotDepth)", 
                               value: $configService.config.chatRelayWotDepth, in: 1...5)
                        Stepper("Min Followers: \(configService.config.chatRelayMinFollowers)",
                               value: $configService.config.chatRelayMinFollowers, in: 0...100)
                    }
                }
            }
            .formStyle(.grouped)
            .padding()
        }
    }
    
    func iconFor(_ relay: RelayType) -> String {
        switch relay {
        case .outbox: return "arrow.up.doc"
        case .inbox: return "arrow.down.doc"
        case .privateRelay: return "lock.fill"
        case .chat: return "bubble.left.and.bubble.right"
        }
    }
}

struct RelayConfigForm: View {
    @Binding var name: String
    @Binding var description: String
    @Binding var icon: String
    
    var body: some View {
        Section("Relay Info") {
            TextField("Name", text: $name)
            TextField("Description", text: $description)
            TextField("Icon URL", text: $icon)
        }
    }
}

struct AdvancedSettingsView: View {
    @EnvironmentObject var configService: ConfigService
    @EnvironmentObject var relayManager: RelayProcessManager
    @State private var showResetConfirmation = false
    
    var body: some View {
        Form {
            Section("Public Relay Rate Limits") {
                Stepper("Max Events: \(configService.config.outboxMaxEventsPerMinute) / min", 
                       value: $configService.config.outboxMaxEventsPerMinute, in: 10...1000, step: 10)
                
                Stepper("Max Connections: \(configService.config.outboxMaxConnectionsPerMinute) / min",
                       value: $configService.config.outboxMaxConnectionsPerMinute, in: 1...100)
                
                Text("These limits help protect your relay from spam and abuse.")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
            
            Section("Database") {
                Picker("Engine", selection: $configService.config.dbEngine) {
                    Text("BadgerDB").tag("badger")
                    Text("LMDB").tag("lmdb")
                }
                
                TextField("Blossom Path", text: $configService.config.blossomPath)
            }
            
            Section("Logging") {
                Picker("Log Level", selection: $configService.config.logLevel) {
                    Text("Debug").tag("DEBUG")
                    Text("Info").tag("INFO")
                    Text("Warning").tag("WARN")
                    Text("Error").tag("ERROR")
                }
            }
            
            Section("Startup") {
                Toggle("Launch at Login", isOn: $configService.config.launchAtLogin)
                Toggle("Auto-start Relay", isOn: $configService.config.autoStartRelay)
            }
            
            Section("Danger Zone") {
                Button(role: .destructive) {
                    showResetConfirmation = true
                } label: {
                    Label("Factory Reset", systemImage: "trash")
                        .foregroundColor(.red)
                }
                Text("This will stop the relay, delete all data (database, logs), and reset settings to default.")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }
        }
        .formStyle(.grouped)
        .padding()
        .alert("Are you sure?", isPresented: $showResetConfirmation) {
            Button("Cancel", role: .cancel) { }
            Button("Reset Everything", role: .destructive) {
                // 1. Stop relay
                relayManager.stopRelay {
                    // 2. Reset app data
                    Task { @MainActor in
                        configService.resetApp()
                        // Application layout checks config.hasCompletedSetup, so this should trigger a view switch
                    }
                }
            }
        } message: {
            Text("This action cannot be undone. all your relay data will be lost.")
        }
    }
}

struct ImportSettingsView: View {
    @EnvironmentObject var configService: ConfigService
    
    var body: some View {
        VStack(spacing: 0) {
            Form {
                Section("Import Configuration") {
                    TextField("Start Date", text: $configService.config.importStartDate)
                        .help("Format: YYYY-MM-DD. Notes will be fetched starting from this date.")
                    
                    TextField("Seed Relays File", text: $configService.config.importSeedRelaysFile)
                        .help("The JSON file containing relays to fetch notes from.")
                }
            }
            .formStyle(.grouped)
            .frame(height: 150)
            
            Divider()
            
            VStack(alignment: .leading) {
                Text("Seed Relays")
                    .font(.headline)
                    .padding(.horizontal)
                    .padding(.top)
                
                RelayListEditor(relays: $configService.config.importSeedRelays)
            }
            
            Text("The import process will fetch your own notes and notes where you are tagged. Make sure you have your npub set correctly in the Identity tab.")
                .font(.caption)
                .foregroundColor(.secondary)
                .padding()
        }
    }
}

struct BlastrSettingsView: View {
    @EnvironmentObject var configService: ConfigService
    
    var body: some View {
        VStack(spacing: 0) {
            Form {
                Section("Blastr Configuration") {
                    TextField("Blastr Relays File", text: $configService.config.blastrRelaysFile)
                        .help("The JSON file containing relays to broadcast notes to.")
                }
            }
            .formStyle(.grouped)
            .frame(height: 100)
            
            Divider()
            
            VStack(alignment: .leading) {
                Text("Broadcast Relays")
                    .font(.headline)
                    .padding(.horizontal)
                    .padding(.top)
                
                RelayListEditor(relays: $configService.config.blastrRelays)
            }
            
            Text("Blastr automatically broadcasts your local notes to these external relays.")
                .font(.caption)
                .foregroundColor(.secondary)
                .padding()
        }
    }
}


struct BackupSettingsView: View {
    @EnvironmentObject var configService: ConfigService
    @State private var showFileImporter = false
    
    var body: some View {
        Form {
            Section("Backup Provider") {
                Picker("Provider", selection: $configService.config.backupProvider) {
                    Text("None").tag("none")
                    Text("S3 Compatible").tag("s3")
                    Text("AWS S3").tag("aws")
                    Text("Google Cloud Storage").tag("gcp")
                }
            }
            
            if configService.config.backupProvider != "none" {
                 Section("Schedule") {
                    Stepper("Backup every \(configService.config.backupIntervalHours) hours",
                           value: $configService.config.backupIntervalHours, in: 1...168)
                }
            }
            
            if configService.config.backupProvider == "s3" {
                Section("S3 Configuration") {
                    TextField("Access Key ID", text: $configService.config.s3AccessKeyId)
                    SecureField("Secret Key", text: $configService.config.s3SecretKey)
                    TextField("Endpoint", text: $configService.config.s3Endpoint)
                    TextField("Region", text: $configService.config.s3Region)
                    TextField("Bucket Name", text: $configService.config.s3BucketName)
                }
            } else if configService.config.backupProvider == "aws" {
                Section("AWS Configuration") {
                    TextField("Access Key ID", text: $configService.config.awsAccessKeyId)
                    SecureField("Secret Access Key", text: $configService.config.awsSecretAccessKey)
                    TextField("Region", text: $configService.config.awsRegion)
                    TextField("Bucket Name", text: $configService.config.awsBucket)
                }
            } else if configService.config.backupProvider == "gcp" {
                Section("GCP Configuration") {
                    TextField("Bucket Name", text: $configService.config.gcpBucketName)
                    
                    HStack {
                        TextField("Credentials JSON", text: $configService.config.gcpCredentialsPath)
                            .disabled(true) // Read-only, set via button
                        
                        Button("Browse...") {
                            showFileImporter = true
                        }
                    }
                    Text("Select your Service Account JSON key file.")
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
            }
        }
        .formStyle(.grouped)
        .padding()
        .fileImporter(
            isPresented: $showFileImporter,
            allowedContentTypes: [.json],
            allowsMultipleSelection: false
        ) { result in
            switch result {
            case .success(let urls):
                if let url = urls.first {
                    // We need to access security scoped resource
                    guard url.startAccessingSecurityScopedResource() else { return }
                    configService.config.gcpCredentialsPath = url.path
                    url.stopAccessingSecurityScopedResource()
                }
            case .failure(let error):
                print("Error selecting file: \(error.localizedDescription)")
            }
        }
    }
}

// RelayListEditor and LogsView moved to separate files

