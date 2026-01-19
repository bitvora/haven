import SwiftUI

struct DashboardView: View {
    @EnvironmentObject var relayManager: RelayProcessManager
    @EnvironmentObject var configService: ConfigService
    @EnvironmentObject var nostrService: NostrService
    
    var body: some View {
        ScrollView {
            VStack(spacing: 20) {
                // MARK: - Relays List
                VStack(alignment: .leading, spacing: 8) {
                    Text("Relays")
                        .font(.system(size: 13, weight: .semibold))
                        .foregroundColor(.secondary)
                        .padding(.horizontal)
                    
                    VStack(spacing: 1) {
                        RelayRow(
                            name: "Outbox",
                            subtitle: "Public notes",
                            icon: "arrow.up.doc",
                            uri: "ws://\(configService.config.relayURL.isEmpty ? "localhost:\(configService.config.relayPort)" : configService.config.relayURL)",
                            endpoint: ""
                        )
                        
                        RelayRow(
                            name: "Private",
                            subtitle: "Drafts & eCash",
                            icon: "lock.fill",
                            uri: "ws://\(configService.config.relayURL.isEmpty ? "localhost:\(configService.config.relayPort)" : configService.config.relayURL)",
                            endpoint: "/private"
                        )
                        
                        RelayRow(
                            name: "Inbox",
                            subtitle: "Tagged notes",
                            icon: "arrow.down.doc",
                            uri: "ws://\(configService.config.relayURL.isEmpty ? "localhost:\(configService.config.relayPort)" : configService.config.relayURL)",
                            endpoint: "/inbox"
                        )
                        
                        RelayRow(
                            name: "Chat",
                            subtitle: "Private DMs",
                            icon: "bubble.left.and.bubble.right",
                            uri: "ws://\(configService.config.relayURL.isEmpty ? "localhost:\(configService.config.relayPort)" : configService.config.relayURL)",
                            endpoint: "/chat"
                        )
                        
                        RelayRow(
                            name: "Blossom",
                            subtitle: "Media Storage",
                            icon: "photo.stack",
                            uri: "http://\(configService.config.relayURL.isEmpty ? "localhost:\(configService.config.relayPort)" : configService.config.relayURL)", // Blossom is usually HTTP
                            endpoint: "/blossom" // Or just root? usually root for simple servers
                        )
                    }
                    .background(Color(NSColor.controlBackgroundColor))
                    .cornerRadius(8)
                    .padding(.horizontal)
                }

                // MARK: - Actions
                HStack(spacing: 12) {
                    ActionButton(icon: "safari", title: "Browser") {
                        if let url = URL(string: "http://\(configService.config.relayURL.isEmpty ? "localhost:\(configService.config.relayPort)" : configService.config.relayURL)") {
                            NSWorkspace.shared.open(url)
                        }
                    }
                    
                    ActionButton(icon: "arrow.down.circle", title: "Import") {
                        let config = configService.config
                        // Run import on background thread to avoid UI freeze
                        DispatchQueue.global(qos: .userInitiated).async {
                            relayManager.importNotes(config: config)
                        }
                    }
                }
                .padding(.horizontal)
                
                .padding(.horizontal)
            }
            .padding(.vertical)
        }
    }
}


struct RelayRow: View {
    let name: String
    let subtitle: String
    let icon: String
    let uri: String
    let endpoint: String
    
    var fullURI: String {
        return uri + endpoint
    }
    
    var body: some View {
        HStack {
            Image(systemName: icon)
                .font(.system(size: 16, weight: .medium))
                .foregroundColor(.havenPurple)
                .frame(width: 28)
            
            VStack(alignment: .leading, spacing: 2) {
                Text(name)
                    .font(.system(size: 13, weight: .semibold))
                Text(subtitle)
                    .font(.system(size: 11))
                    .foregroundColor(.secondary)
            }
            
            Spacer()
            
            Text(fullURI)
                .font(.system(size: 10, design: .monospaced))
                .foregroundColor(.secondary)
                .padding(.horizontal, 6)
                .padding(.vertical, 3)
                .background(Color.havenPurplePale)
                .cornerRadius(4)
                
            Button(action: {
                NSPasteboard.general.clearContents()
                NSPasteboard.general.setString(fullURI, forType: .string)
            }) {
                Image(systemName: "doc.on.doc")
                    .foregroundColor(.secondary)
            }
            .buttonStyle(.plain)
            .padding(.leading, 8)
        }
        .padding(.horizontal, 12)
        .padding(.vertical, 10)
        .background(Color(NSColor.controlBackgroundColor))
    }
}

struct ActionButton: View {
    let icon: String
    let title: String
    let action: () -> Void
    
    var body: some View {
        Button(action: action) {
            VStack(spacing: 6) {
                Image(systemName: icon)
                    .font(.system(size: 18, weight: .medium))
                    .foregroundColor(.white)
                Text(title)
                    .font(.system(size: 13, weight: .semibold))
                    .foregroundColor(.white)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 14)
            .background(
                LinearGradient(
                    gradient: Gradient(colors: [.havenPurple, .havenPurpleDark]),
                    startPoint: .top,
                    endPoint: .bottom
                )
            )
            .cornerRadius(8)
        }
        .buttonStyle(.plain)
    }
}
