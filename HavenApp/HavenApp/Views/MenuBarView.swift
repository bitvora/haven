import SwiftUI

struct MenuBarView: View {
    @ObservedObject var configService: ConfigService
    @ObservedObject var relayManager: RelayProcessManager
    @State private var selectedTab: Tab = .dashboard
    @Environment(\.openSettings) var openSettings
    
    enum Tab {
        case dashboard
        case viewer
    }
    
    var body: some View {
        ZStack {
            // MARK: - Main Content
            VStack(spacing: 0) {
                // MARK: - Header
                HStack {
                    Label("Haven", systemImage: "server.rack")
                        .font(.system(size: 16, weight: .semibold))
                        .foregroundColor(.havenPurple)
                    
                    Spacer()
                    
                    if relayManager.isBooting {
                        Text(relayManager.bootStatusMessage)
                            .font(.system(size: 11, weight: .medium))
                            .foregroundColor(.secondary)
                            .transition(.opacity)
                    }
                    
                    if relayManager.isLocked {
                        Button(action: {
                            relayManager.clearDatabaseLocks()
                        }) {
                            HStack(spacing: 6) {
                                Image(systemName: "wrench.and.screwdriver.fill")
                                Text("Fix Database Locks")
                                    .font(.system(size: 12, weight: .semibold))
                            }
                            .padding(.horizontal, 12)
                            .padding(.vertical, 6)
                            .background(Color.orange.opacity(0.2))
                            .foregroundColor(.orange)
                            .cornerRadius(12)
                        }
                        .buttonStyle(.plain)
                    }

                    Button(action: {
                        if relayManager.isRunning {
                            relayManager.stopRelay()
                        } else {
                            relayManager.startRelay(config: configService.config)
                        }
                    }) {
                        HStack(spacing: 6) {
                            Circle()
                                .fill(relayManager.isBooting ? Color.yellow : (relayManager.isRunning ? Color.green : Color.red))
                                .frame(width: 8, height: 8)
                            Text(relayManager.isBooting ? "Booting Relay" : (relayManager.isRunning ? "Stop Relay" : "Start Relay"))
                                .font(.system(size: 12, weight: .semibold))
                        }
                        .padding(.horizontal, 12)
                        .padding(.vertical, 6)
                        .background(relayManager.isBooting ? Color.yellow.opacity(0.2) : Color.havenPurplePale)
                        .foregroundColor(relayManager.isBooting ? Color.orange : Color.primary)
                        .cornerRadius(12)
                    }
                    .buttonStyle(.plain)
                }
                .padding()
                .background(Color(NSColor.windowBackgroundColor))
                
                // MARK: - Tabs
                HStack(spacing: 0) {
                    TabButton(icon: "gauge", title: "Dashboard", isSelected: selectedTab == .dashboard) {
                        selectedTab = .dashboard
                    }
                    
                    TabButton(icon: "doc.text.image", title: "Viewer", isSelected: selectedTab == .viewer) {
                        selectedTab = .viewer
                    }
                }
                .padding(.horizontal)
                .padding(.bottom)
                .background(Color(NSColor.windowBackgroundColor))
                
                Divider()
                
                // MARK: - Content
                ZStack {
                    Color(NSColor.textBackgroundColor) // Darker background
                        .ignoresSafeArea()
                    
                    switch selectedTab {
                    case .dashboard:
                        DashboardView()
                    case .viewer:
                        ViewerView()
                    }
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
                
                Divider()
                
                // MARK: - Footer
                HStack {
                    Button("Settings") {
                        NSApp.activate(ignoringOtherApps: true)
                        if #available(macOS 14.0, *) {
                            openSettings()
                        } else {
                            NSApp.sendAction(Selector(("showSettingsWindow:")), to: nil, from: nil)
                        }
                    }
                    .buttonStyle(.plain)
                    .foregroundColor(.secondary)
                    
                    Spacer()
                    
                    Button("Quit Haven") {
                        NSApp.terminate(nil)
                    }
                    .buttonStyle(.plain)
                    .foregroundColor(.secondary)
                }
                .padding()
                .background(Color(NSColor.windowBackgroundColor))
            }
            .disabled(relayManager.isImporting) // Disable interaction when importing
            
            // MARK: - Import Overlay
            if relayManager.isImporting {
                ZStack {
                    Color.black.opacity(0.8)
                        .ignoresSafeArea()
                    
                    VStack(spacing: 24) {
                        Text("Importing Notes")
                            .font(.title2.bold())
                            .foregroundColor(.white)
                        
                        // Progress Bar Custom Style
                        VStack(alignment: .leading, spacing: 8) {
                            HStack {
                                Text(relayManager.importStatusMessage)
                                    .font(.caption)
                                    .foregroundColor(.white.opacity(0.8))
                                Spacer()
                                Text("\(Int(relayManager.importProgress * 100))%")
                                    .font(.caption.monospaced())
                                    .foregroundColor(.white)
                            }
                            
                            GeometryReader { geo in
                                ZStack(alignment: .leading) {
                                    RoundedRectangle(cornerRadius: 4)
                                        .fill(Color.havenPurplePale)
                                        .frame(height: 6)
                                    
                                    RoundedRectangle(cornerRadius: 4)
                                        .fill(
                                            LinearGradient(
                                                gradient: Gradient(colors: [.havenPurple, .havenPurpleLight]),
                                                startPoint: .leading,
                                                endPoint: .trailing
                                            )
                                        )
                                        .frame(width: geo.size.width * relayManager.importProgress, height: 6)
                                }
                            }
                            .frame(height: 6)
                        }
                        .frame(width: 300)
                        
                        Text("Please keep the app open.")
                            .font(.footnote)
                            .foregroundColor(.white.opacity(0.5))
                        
                        Divider()
                            .background(Color.white.opacity(0.2))
                        
                        if relayManager.importProgress >= 1.0 || relayManager.importStatusMessage.contains("Failed") || relayManager.importStatusMessage.contains("Complete") {
                            Button(action: {
                                relayManager.dismissImport()
                            }) {
                                Text("Close")
                                    .font(.system(size: 14, weight: .bold))
                                    .foregroundColor(.white)
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 10)
                                    .background(Color.havenPurple)
                                    .cornerRadius(8)
                            }
                            .buttonStyle(.plain)
                            .frame(width: 200)
                        } else {
                            Button(action: {
                                relayManager.cancelImport()
                            }) {
                                Text("Cancel Import")
                                    .font(.system(size: 13))
                                    .foregroundColor(.red)
                                    .padding(.horizontal, 16)
                                    .padding(.vertical, 8)
                                    .background(Color.red.opacity(0.1))
                                    .cornerRadius(8)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                    .padding(24)
                    .background(Color(NSColor.windowBackgroundColor))
                    .cornerRadius(12)
                    .shadow(color: Color.black.opacity(0.3), radius: 20)
                }
                .transition(.opacity)
            }
        }
        .frame(width: 480, height: 640)
    }
}

struct TabButton: View {
    let icon: String
    let title: String
    let isSelected: Bool
    let action: () -> Void
    
    var body: some View {
        Button(action: action) {
            VStack(spacing: 4) {
                Image(systemName: icon)
                    .font(.system(size: 16, weight: .medium))
                Text(title)
                    .font(.system(size: 11, weight: .semibold))
            }
            .foregroundColor(isSelected ? .havenPurple : .secondary)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 10)
            .background(isSelected ? Color.havenPurplePale : Color.clear)
            .cornerRadius(6)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}
