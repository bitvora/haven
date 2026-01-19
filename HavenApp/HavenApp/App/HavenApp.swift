import SwiftUI

@main
struct HavenApp: App {
    @StateObject private var configService = ConfigService()
    @StateObject private var relayManager = RelayProcessManager()
    @StateObject private var nostrService = NostrService()
    
    var body: some Scene {
        MenuBarExtra("Haven", systemImage: "server.rack") {
            Group {
                if !configService.config.hasCompletedSetup {
                    SetupWizardView {
                        Task { @MainActor in
                            relayManager.startRelay(config: configService.config)
                        }
                    }
                } else {
                    MenuBarView(configService: configService, relayManager: relayManager)
                }
            }
            .environmentObject(configService)
            .environmentObject(relayManager)
            .environmentObject(nostrService)
        }
        .menuBarExtraStyle(.window)
        
        // Keep Settings window for advanced usage if needed, or open via a button in MenuBarView
        Settings {
            SettingsView()
                .environmentObject(configService)
                .environmentObject(relayManager)
                .environmentObject(nostrService)
        }
    }
}

