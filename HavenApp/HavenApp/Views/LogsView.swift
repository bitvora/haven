import SwiftUI

struct LogsView: View {
    @EnvironmentObject var relayManager: RelayProcessManager
    
    var body: some View {
        ScrollViewReader { proxy in
            List(relayManager.logs) { log in
                HStack(alignment: .top) {
                    Text(log.timestamp, style: .time)
                        .font(.caption2)
                        .foregroundColor(.secondary)
                        .frame(width: 60, alignment: .leading)
                    
                    Text(log.level)
                        .font(.caption2)
                        .fontWeight(.bold)
                        .foregroundColor(colorFor(level: log.level))
                        .frame(width: 50, alignment: .leading)
                    
                    Text(log.message)
                        .font(.callout)
                        .fontDesign(.monospaced)
                }
                .id(log.id) // Important for identifying the row
            }
            .listStyle(.plain)
            .padding(.bottom, 20)
            .onChange(of: relayManager.logs.count) { oldValue, newValue in
                // Auto-scroll to bottom directly
                if let lastId = relayManager.logs.last?.id {
                    proxy.scrollTo(lastId, anchor: .bottom)
                }
            }
            // Also scroll on appear just in case
            .onAppear {
                if let lastId = relayManager.logs.last?.id {
                     proxy.scrollTo(lastId, anchor: .center)
                }
            }
        }
    }
    
    func colorFor(level: String) -> Color {
        switch level {
        case "ERROR": return .red
        case "WARN": return .orange
        default: return .primary
        }
    }
}
